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

package memory

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/tdrn-org/mnemosyne/config"
	"github.com/tdrn-org/mnemosyne/internal/domain"
	"github.com/tdrn-org/mnemosyne/internal/provider"
	"github.com/tdrn-org/mnemosyne/internal/vectordb"
)

type Memory struct {
	cfg      *config.MemoryConfig
	vectorDB *vectordb.Store
	embedder provider.Embedder
	logger   *slog.Logger
}

func NewMemory(cfg *config.MemoryConfig, vectorDB *vectordb.Store, embedder provider.Embedder) *Memory {
	logger := slog.With("collection", "memory")
	return &Memory{
		cfg:      cfg,
		vectorDB: vectorDB,
		embedder: embedder,
		logger:   logger,
	}
}

func (m *Memory) ListMemoryTypes(ctx context.Context) ([]domain.MemoryType, error) {
	memoryTypes := make([]domain.MemoryType, 0, len(m.cfg.Types))
	for _, typ := range m.cfg.Types {
		memoryTypes = append(memoryTypes, domain.MemoryType{
			Name:        typ.Name,
			TTL:         time.Duration(typ.TTL),
			Description: typ.Description,
		})
	}
	slices.SortFunc(memoryTypes, func(a, b domain.MemoryType) int {
		return strings.Compare(a.Name, b.Name)
	})
	return memoryTypes, nil
}

func (m *Memory) RememberMemory(ctx context.Context, memory *domain.Memory) error {
	if memory.ID == "" {
		memory.ID = domain.MemoryID(memory.Type, memory.Content)
	}
	for _, memoryType := range m.cfg.Types {
		if memory.Type == memoryType.Name {
			memory.ExpiresAt = memory.LastAccess.Add(time.Duration(memoryType.TTL))
		}
	}
	embedding, err := m.embedder.Embed(ctx, memory.Content)
	if err != nil {
		return fmt.Errorf("failed to generate embedding for memory (cause: %w)", err)
	}
	return m.vectorDB.UpsertMemory(ctx, memory, embedding...)
}

func (m *Memory) ForgetMemory(ctx context.Context, id string) error {
	return m.vectorDB.DeleteMemory(ctx, id)
}

func (m *Memory) RecallMemories(ctx context.Context, query string, opts domain.RecallOptions) ([]domain.Memory, error) {
	embedding, err := m.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding for query (cause: %w)", err)
	}
	memories, err := m.vectorDB.SearchMemories(ctx, opts.TypeFilter, opts.MinTrust, opts.Limit, embedding...)
	if err != nil {
		return nil, fmt.Errorf("memory search failure (cause: %w)", err)
	}
	return memories, nil
}

func (m *Memory) ReinforceMemory(ctx context.Context, id string, trustDelta float64) error {
	memory, vector, err := m.vectorDB.LookupMemory(ctx, id)
	if err != nil {
		return err
	}
	if memory == nil {
		return domain.ErrNoSuchMemory
	}
	memory.AdjustTrust(trustDelta)
	memory.Touch()
	return m.vectorDB.UpsertMemory(ctx, memory, vector...)
}

func (m *Memory) TouchMemory(ctx context.Context, id string) error {
	memory, vector, err := m.vectorDB.LookupMemory(ctx, id)
	if err != nil {
		return err
	}
	if memory == nil {
		return domain.ErrNoSuchMemory
	}
	memory.Touch()
	return m.vectorDB.UpsertMemory(ctx, memory, vector...)
}

func (m *Memory) Sync(ctx context.Context, _ time.Duration) {
	m.logger.Info("discarding expired memories...")
	err := m.vectorDB.DeleteExpiredMemories(ctx, time.Now())
	if err != nil {
		m.logger.Warn("failed to discard expired memories", slog.Any("err", err))
	}
}
