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
	"fmt"
	"io/fs"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/template"

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
	store         *vectordb.Store
	embedder      provider.Embedder
	markdownSyncs []markdownSync
	logger        *slog.Logger
}

func NewKnowledge(cfg *config.KnowledgeConfig, store *vectordb.Store, tokenizer markdown.Tokenizer, embedder provider.Embedder) *Knowledge {
	logger := slog.With("collection", "knowledge")
	markdownSyncs := make([]markdownSync, 0, len(cfg.MarkdownSources))
	for _, markdownSource := range cfg.MarkdownSources {
		markdownSyncs = append(markdownSyncs, markdownSync{
			cfg:      &markdownSource,
			store:    store,
			embedder: embedder,
			Parser:   markdown.NewParser(markdownSource.Store, tokenizer),
			logger:   logger.With(slog.String("source", fmt.Sprintf("markdown[%s]", markdownSource.Nature)), slog.String("path", markdownSource.Path)),
		})
	}
	return &Knowledge{
		store:         store,
		embedder:      embedder,
		markdownSyncs: markdownSyncs,
		logger:        logger,
	}
}

func (k *Knowledge) ListStores(ctx context.Context) ([]string, error) {
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
	chunks, err := k.store.SearchChunks(ctx, store, limit, embedding...)
	if err != nil {
		return nil, fmt.Errorf("chunk search failure (cause: %w)", err)
	}
	return chunks, nil
}

func (k *Knowledge) ReadDocument(ctx context.Context, store, path string, limit *int) (string, error) {
	return "", nil
}

func (k *Knowledge) Sync(ctx context.Context) {
	k.logger.Info("syncing sources...")
	for _, markdownSync := range k.markdownSyncs {
		markdownSync.Run(ctx)
	}
}

type markdownSync struct {
	cfg      *config.MarkdownSourceConfig
	store    *vectordb.Store
	embedder provider.Embedder
	Parser   *markdown.Parser
	logger   *slog.Logger
}

func (s *markdownSync) Run(ctx context.Context) {
	s.logger.Info("syncing source...")
	err := filepath.WalkDir(s.cfg.Path, func(path string, d fs.DirEntry, err error) error {
		return s.walkDir(ctx, path, d, err)
	})
	if err != nil {
		s.logger.Error("failed to sync source", slog.Any("err", err))
	}
}

func (s *markdownSync) walkDir(ctx context.Context, path string, d fs.DirEntry, err0 error) error {
	pathLogger := s.logger.With(slog.String("path", path))
	consider, err := s.considerPath(pathLogger, path, d, err0)
	if !consider {
		return err
	}
	pathLogger.Info("sync: processing Markdown file")
	relPath, err := filepath.Rel(s.cfg.Path, path)
	if err != nil {
		pathLogger.Warn("failed to resolve relative path of Markdown file", slog.Any("err", err))
		return nil
	}
	source, err := os.ReadFile(path)
	if err != nil {
		pathLogger.Warn("failed to read Markdown file", slog.Any("err", err))
		return nil
	}
	sourceHash := crypto.HashData(source)
	document, err := s.store.LookupDocument(ctx, path)
	if err != nil {
		pathLogger.Error("failed to lookup document", slog.String("path", path), slog.Any("err", err))
		return nil
	}
	if document != nil && document.Path == relPath && document.Hash == sourceHash {
		pathLogger.Debug("sync: skipping unchanged Markdown file")
		return nil
	}
	tokenLimit := s.cfg.ChunkTokenLimit
	if tokenLimit == 0 {
		tokenLimit = DefaultTokenLimit
	}
	chunks, err := s.Parser.Parse(relPath, source, tokenLimit)
	if err != nil {
		pathLogger.Error("failed to parse Markdown file", slog.Any("err", err))
		return nil
	}
	s.syncChunks(ctx, chunks)
	document = &domain.Document{
		ID:   vectordb.DocumentID(relPath),
		Path: relPath,
		Hash: sourceHash,
	}
	//TODO: Update hash only if all chunks are processed successfully
	err = s.store.UpsertDocument(ctx, document)
	if err != nil {
		pathLogger.Error("failed to upsert document", slog.String("path", path), slog.Any("err", err))
		return nil
	}
	return nil
}

func (s *markdownSync) considerPath(pathLogger *slog.Logger, path string, d fs.DirEntry, err0 error) (bool, error) {
	if err0 != nil {
		return false, err0
	}
	if !s.cfg.PathFilter.Match(path) {
		if d.IsDir() {
			pathLogger.Debug("sync: ignoring directory")
			return false, filepath.SkipDir
		} else {
			pathLogger.Debug("sync: ignoring file")
			return false, nil
		}
	}
	if !d.Type().IsRegular() {
		return false, nil
	}
	extension := filepath.Ext(path)
	if extension != ".md" {
		pathLogger.Debug("sync: skipping Non-Markdown file")
		return false, nil
	}
	return true, nil
}

func (s *markdownSync) syncChunks(ctx context.Context, chunks []domain.Chunk) {
	for _, chunk := range chunks {
		chunkText, err := s.renderChunk(&chunk)
		if err != nil {
			s.logger.Warn("failed to render Markdown chunk", slog.Any("err", err))
			continue
		}
		embedding, err := s.embedder.Embed(ctx, chunkText)
		if err != nil {
			s.logger.Warn("failed to generate embedding for Markdown chunk", slog.Any("err", err))
			continue
		}
		err = s.store.UpsertChunk(ctx, &chunk, embedding...)
		if err != nil {
			s.logger.Warn("failed to upsert Markdown chunk", slog.Any("err", err))
			continue
		}
	}
}

func (s *markdownSync) renderChunk(chunk *domain.Chunk) (string, error) {
	templText := s.cfg.ChunkRenderTemplate
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
