/*
 * Copyright 2026 Holger de Carne
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package knowledge

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/template"
	"time"
	"unicode/utf8"

	"github.com/tdrn-org/mnemosyne/config"
	"github.com/tdrn-org/mnemosyne/internal/crypto"
	"github.com/tdrn-org/mnemosyne/internal/domain"
	"github.com/tdrn-org/mnemosyne/internal/parser/markdown"
	"github.com/tdrn-org/mnemosyne/internal/provider"
	"github.com/tdrn-org/mnemosyne/internal/vectordb"
)

const DefaultTokenLimit int = 512
const DefaultRenderTemplate string = `Document: {{.DocumentTitle}}
covers in section {{join .HeadingPath " > "}}

{{.Content}}`

type Knowledge struct {
	vectorDB        *vectordb.Store
	embedder        provider.Embedder
	markdownSyncs   []markdownSync
	metricsRecorder MetricsRecorder
	logger          *slog.Logger
}

// MetricsRecorder receives search and read observations. Implementations must be
// safe for concurrent use. A nil recorder disables instrumentation, which keeps
// this package free of any metrics dependency.
type MetricsRecorder interface {
	// ObserveDocumentCount reports an updated document count for the given store.
	ObserveDocumentCount(store string, count uint64)
	// ObserveChunkCount reports an updated chunk count for the given store.
	ObserveChunkCount(store string, count uint64)
	// ObserveSearch reports one completed search: the number of chunks found and
	// the best similarity score among them (0 when nothing was found).
	ObserveSearch(store string, results int, topScore float64)
	// ObserveRead reports one completed document read.
	ObserveRead(store string)
}

// AllStores is the store label used when a search is not scoped to a single store.
const AllStores string = "all"

func NewKnowledge(ctx context.Context, cfg *config.KnowledgeConfig, vectorDB *vectordb.Store, tokenizer markdown.Tokenizer, embedder provider.Embedder, metricsRecorder MetricsRecorder) *Knowledge {
	logger := slog.With("collection", "knowledge")
	markdownSyncs := make([]markdownSync, 0, len(cfg.MarkdownSources))
	for _, markdownSource := range cfg.MarkdownSources {
		markdownSyncs = append(markdownSyncs, markdownSync{
			Cfg:      &markdownSource,
			Parser:   markdown.NewParser(markdownSource.Store, tokenizer),
			vectorDB: vectorDB,
			embedder: embedder,
			logger:   logger.With(slog.String("source", fmt.Sprintf("markdown[%s]:%s", markdownSource.Nature, markdownSource.Path))),
		})
	}
	k := &Knowledge{
		vectorDB:        vectorDB,
		embedder:        embedder,
		markdownSyncs:   markdownSyncs,
		metricsRecorder: metricsRecorder,
		logger:          logger,
	}
	k.observeDocumentChunkCount(ctx)
	return k
}

func (k *Knowledge) ListStores(_ context.Context) ([]string, error) {
	storeMap := make(map[string]string)
	for _, markdownSync := range k.markdownSyncs {
		store := markdownSync.Parser.Store()
		storeMap[store] = store
	}
	stores := slices.Collect(maps.Keys(storeMap))
	slices.Sort(stores)
	return stores, nil
}

func (k *Knowledge) SearchStore(ctx context.Context, query string, store *string, limit *uint64) ([]domain.Chunk, error) {
	embedding, err := k.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding for query (cause: %w)", err)
	}
	chunks, err := k.vectorDB.SearchChunks(ctx, store, limit, embedding...)
	if err != nil {
		return nil, fmt.Errorf("chunk search failure (cause: %w)", err)
	}
	k.observeSearch(store, chunks)
	return chunks, nil
}

func (k *Knowledge) observeSearch(store *string, chunks []domain.Chunk) {
	storeName := AllStores
	if store != nil {
		storeName = *store
	}
	topScore := 0.0
	if len(chunks) > 0 {
		topScore = float64(chunks[0].Score)
		for _, chunk := range chunks[1:] {
			if score := float64(chunk.Score); score > topScore {
				topScore = score
			}
		}
	}
	k.metricsRecorder.ObserveSearch(storeName, len(chunks), topScore)
}

func (k *Knowledge) ReadDocument(ctx context.Context, store, path string, limit *int) (string, error) {
	for _, markdownSync := range k.markdownSyncs {
		if markdownSync.Parser.Store() == store {
			source, err := markdownSync.FindDocument(ctx, path)
			if err != nil {
				return "", err
			}
			if source == nil {
				return "", nil
			}
			document := k.limitDocument(string(source), limit)
			k.metricsRecorder.ObserveRead(store)
			return document, nil
		}
	}
	return "", fmt.Errorf("unknown store '%s'", store)
}

func (k *Knowledge) limitDocument(document string, limit *int) string {
	if limit == nil {
		return document
	}
	maxRunes := *limit
	if maxRunes <= 0 || len(document) < maxRunes || utf8.RuneCountInString(document) < maxRunes {
		return document
	}
	runeCount := 0
	for byteIndex := range document {
		if runeCount == maxRunes {
			return document[:byteIndex] + "..."
		}
		runeCount++
	}
	return document
}

func (k *Knowledge) Sync(ctx context.Context, last, now time.Time) error {
	k.logger.Info("syncing sources...")
	syncErrs := make([]error, 0, len(k.markdownSyncs))
	for _, markdownSync := range k.markdownSyncs {
		if markdownSync.Cfg.Schedule.InInterval(last, now) {
			syncErr := markdownSync.RunSync(ctx)
			if syncErr != nil {
				syncErrs = append(syncErrs, syncErr)
			}
			k.observeDocumentChunkCount(ctx)
		}
	}
	return errors.Join(syncErrs...)
}

func (k *Knowledge) observeDocumentChunkCount(ctx context.Context) {
	for _, markdownSync := range k.markdownSyncs {
		documentCount, err := k.vectorDB.CountDocuments(ctx, nil)
		if err == nil {
			k.metricsRecorder.ObserveDocumentCount(markdownSync.Cfg.Store, documentCount)
		} else {
			k.logger.Error("failed to query document count", slog.Any("err", err))
		}
		chunkCount, err := k.vectorDB.CountChunks(ctx, nil)
		if err == nil {
			k.metricsRecorder.ObserveChunkCount(markdownSync.Cfg.Store, chunkCount)
		} else {
			k.logger.Error("failed to query chunk count", slog.Any("err", err))
		}
	}
}

type markdownSync struct {
	Cfg      *config.MarkdownSourceConfig
	Parser   *markdown.Parser
	vectorDB *vectordb.Store
	embedder provider.Embedder
	logger   *slog.Logger
}

func (s *markdownSync) FindDocument(ctx context.Context, documentPath string) ([]byte, error) {
	s.logger.Info("searching document...", slog.String("document", documentPath))
	var foundSource []byte
	err := filepath.WalkDir(s.Cfg.Path, func(path string, d fs.DirEntry, err error) error {
		return s.walkSources(ctx, func(_ context.Context, path, relPath string, source []byte) error {
			if relPath != documentPath {
				return nil
			}
			foundSource = source
			return filepath.SkipAll
		}, path, d, err)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search document '%s' (cause: %w)", documentPath, err)
	}
	return foundSource, nil
}

func (s *markdownSync) RunSync(ctx context.Context) error {
	s.logger.Info("syncing source...")
	err := filepath.WalkDir(s.Cfg.Path, func(path string, d fs.DirEntry, err error) error {
		return s.walkSources(ctx, s.syncSource, path, d, err)
	})
	if err != nil {
		s.logger.Error("failed to sync source", slog.Any("err", err))
	}
	return err
}

func (s *markdownSync) walkSources(ctx context.Context, fn func(context.Context, string, string, []byte) error, path string, d fs.DirEntry, err0 error) error {
	pathLogger := s.logger.With(slog.String("path", path))
	pathLogger.Debug("processing Markdown file")
	relPath, err := filepath.Rel(s.Cfg.Path, path)
	if err != nil {
		pathLogger.Warn("failed to resolve relative path of Markdown file", slog.Any("err", err))
		return err
	}
	consider, err := s.considerPath(path, d, err0)
	if !consider {
		return err
	}
	source, err := os.ReadFile(path)
	if err != nil {
		pathLogger.Warn("failed to read Markdown file", slog.Any("err", err))
		return err
	}
	return fn(ctx, path, relPath, source)
}

func (s *markdownSync) syncSource(ctx context.Context, path, relPath string, source []byte) error {
	pathLogger := s.logger.With(slog.String("path", path))
	sourceHash := crypto.HashData(source)
	document, err := s.vectorDB.LookupDocument(ctx, relPath)
	if err != nil {
		pathLogger.Error("failed to lookup document", slog.Any("err", err))
		return err
	}
	if document != nil && document.Path == relPath && document.Hash == sourceHash {
		pathLogger.Debug("skipping unchanged Markdown file")
		return nil
	}
	tokenLimit := s.Cfg.ChunkTokenLimit
	if tokenLimit == 0 {
		tokenLimit = DefaultTokenLimit
	}
	chunks, err := s.Parser.Parse(relPath, source, tokenLimit)
	if err != nil {
		pathLogger.Error("failed to parse Markdown file", slog.Any("err", err))
		return err
	}
	err = s.syncChunks(ctx, chunks)
	if err != nil {
		pathLogger.Error("failed to sync chunks", slog.Any("err", err))
		return err
	}
	document = &domain.Document{
		ID:   vectordb.DocumentID(relPath),
		Path: relPath,
		Hash: sourceHash,
	}
	err = s.vectorDB.UpsertDocument(ctx, document)
	if err != nil {
		pathLogger.Error("failed to upsert document", slog.String("path", path), slog.Any("err", err))
		return err
	}
	return nil
}

func (s *markdownSync) considerPath(path string, d fs.DirEntry, err0 error) (bool, error) {
	pathLogger := s.logger.With(slog.String("path", path))
	if err0 != nil {
		return false, err0
	}
	if !s.Cfg.PathFilter.Match(path) {
		if d.IsDir() {
			pathLogger.Debug("ignoring directory")
			return false, filepath.SkipDir
		} else {
			pathLogger.Debug("ignoring file")
			return false, nil
		}
	}
	if !d.Type().IsRegular() {
		return false, nil
	}
	extension := filepath.Ext(path)
	if extension != ".md" {
		pathLogger.Debug("skipping Non-Markdown file")
		return false, nil
	}
	return true, nil
}

func (s *markdownSync) syncChunks(ctx context.Context, chunks []domain.Chunk) error {
	for _, chunk := range chunks {
		chunkText, err := s.renderChunk(&chunk)
		if err != nil {
			s.logger.Error("failed to render Markdown chunk", slog.Any("err", err))
			return err
		}
		embedding, err := s.embedder.Embed(ctx, chunkText)
		if err != nil {
			s.logger.Error("failed to generate embedding for Markdown chunk", slog.Any("err", err))
			return err
		}
		err = s.vectorDB.UpsertChunk(ctx, &chunk, embedding...)
		if err != nil {
			s.logger.Error("failed to upsert Markdown chunk", slog.Any("err", err))
			return err
		}
	}
	return nil
}

func (s *markdownSync) renderChunk(chunk *domain.Chunk) (string, error) {
	templText := s.Cfg.ChunkRenderTemplate
	if templText == "" {
		templText = DefaultRenderTemplate
	}
	funcs := template.FuncMap{
		"join": strings.Join,
	}
	templ, err := template.New("chunk_template").Funcs(funcs).Parse(templText)
	if err != nil {
		return "", fmt.Errorf("failed to parse chunk render template (cause: %w)", err)
	}
	buffer := &strings.Builder{}
	err = templ.Execute(buffer, chunk)
	if err != nil {
		return "", fmt.Errorf("failed to execute chunk render template (cause: %w)", err)
	}
	return buffer.String(), nil
}
