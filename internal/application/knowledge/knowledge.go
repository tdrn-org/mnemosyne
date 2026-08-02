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

	"github.com/tdrn-org/mnemosyne/config"
	"github.com/tdrn-org/mnemosyne/internal/domain"
	"github.com/tdrn-org/mnemosyne/internal/parser/markdown"
	"github.com/tdrn-org/mnemosyne/internal/provider"
	"github.com/tdrn-org/mnemosyne/internal/vectordb"
)

const TokenLimit int = 512

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
			store:    store,
			embedder: embedder,
			Parser:   markdown.NewParser(markdownSource.Store, tokenizer),
			Path:     markdownSource.Path,
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
	if k.store == nil || k.embedder == nil {
		return nil, fmt.Errorf("knowledge store not initialized")
	}
	embedding, err := k.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("embedding query: %w", err)
	}
	chunks, err := k.store.SearchChunks(ctx, limit, embedding...)
	if err != nil {
		return nil, fmt.Errorf("searching chunks: %w", err)
	}
	if store != nil {
		filtered := make([]domain.Chunk, 0, len(chunks))
		for _, chunk := range chunks {
			if chunk.Store == *store {
				filtered = append(filtered, chunk)
			}
		}
		chunks = filtered
	}
	return chunks, nil
}

func (k *Knowledge) Sync(ctx context.Context) {
	k.logger.Info("syncing sources...")
	for _, markdownSync := range k.markdownSyncs {
		markdownSync.Run(ctx)
	}
}

type markdownSync struct {
	store    *vectordb.Store
	embedder provider.Embedder
	Parser   *markdown.Parser
	Path     string
	logger   *slog.Logger
}

func (s *markdownSync) Run(ctx context.Context) {
	s.logger.Info("syncing source...")
	err := filepath.WalkDir(s.Path, func(path string, d fs.DirEntry, err error) error {
		return s.walkDir(ctx, path, d, err)
	})
	if err != nil {
		s.logger.Error("failed to sync source", slog.Any("err", err))
	}
}

func (s *markdownSync) walkDir(ctx context.Context, path string, d fs.DirEntry, _ error) error {
	if !d.Type().IsRegular() {
		return nil
	}
	fileLogger := s.logger.With(slog.String("file", path))
	extension := filepath.Ext(path)
	if extension != ".md" {
		fileLogger.Debug("sync: skipping Non-Markdown file")
		return nil
	}
	fileLogger.Info("sync: processing Markdown file")
	source, err := os.ReadFile(path)
	if err != nil {
		fileLogger.Warn("failed to read Markdown file", slog.Any("err", err))
		return nil
	}
	chunks, err := s.Parser.Parse(path, source, TokenLimit)
	if err != nil {
		fileLogger.Error("failed to parse Markdown file", slog.Any("err", err))
		return nil
	}
	s.syncChunks(ctx, chunks)
	return nil
}

func (s *markdownSync) syncChunks(ctx context.Context, chunks []domain.Chunk) {
	for _, chunk := range chunks {
		embedding, err := s.embedder.Embed(ctx, chunk.Content)
		if err != nil {
			s.logger.Warn("failed to generate embedding for Markdown chunk", slog.Any("err", err))
		}
		err = s.store.UpsertChunk(ctx, &chunk, embedding...)
		if err != nil {
			s.logger.Warn("failed to upsert Markdown chunk", slog.Any("err", err))
			continue
		}
	}
}
