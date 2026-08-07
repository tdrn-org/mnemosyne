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

package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type MemoryType struct {
	Name        string        `json:"name"`
	TTL         time.Duration `json:"ttl"`
	Description string        `json:"description"`
}

// Memory is the core domain type for a single memory record.
type Memory struct {
	// ID is a deterministic UUIDv5 derived from content + source.
	ID string `json:"id"`
	// Content is the raw memory text.
	Content string `json:"content"`
	// Type classifies the memory (fact, decision, emotional, etc.).
	Type string `json:"type"`
	// Trust is a 0.0–1.0 score indicating confidence in this memory.
	Trust float64 `json:"trust"`
	// Labels are free-form tags for filtering.
	Labels []string `json:"labels,omitempty"`
	// CreatedAt is when the memory was stored.
	CreatedAt time.Time `json:"created_at"`
	// ExpiresAt is the calculated expiry. Zero time means never expires.
	ExpiresAt time.Time `json:"expires_at,omitempty"`
	// LastAccess is updated every time the memory is accessed (for TTL reset).
	LastAccess time.Time `json:"last_access"`
}

// RecallOptions controls how memories are retrieved.
type RecallOptions struct {
	TypeFilter *string
	MinTrust   *float64
	Limit      *uint64
}

func MemoryID() string {
	return uuid.NewString()
}

// MemoryStore is the domain interface for the memory collection.
type MemoryStore interface {

	// ListMemoryTypes lists the defined memory types.
	ListMemoryTypes(ctx context.Context) ([]MemoryType, error)

	// RememberMemory stores a new memory entry. The store is responsible for
	// generating the embedding and setting the ID if not provided.
	RememberMemory(ctx context.Context, entry *Memory) error

	// Forget removes a memory entry by ID.
	ForgetMemory(ctx context.Context, id string) error

	// Recall searches memories by semantic similarity to the query.
	RecallMemories(ctx context.Context, query string, opts RecallOptions) ([]Memory, error)

	// Reinforce adjusts the trust score of a memory entry.
	// Positive delta increases trust, negative decreases it.
	ReinforceMemory(ctx context.Context, id string, trustDelta float64) error

	// Touch resets the expiry timer for a memory entry.
	TouchMemory(ctx context.Context, id string) error

	// Expire removes all memories that have passed their expiry time.
	// Returns the count of removed entries.
	//	ExpireMemories(ctx context.Context) (int, error)

	// ListByType returns all memories of a given type.
	//	ListMemoriesByType(ctx context.Context, memType MemoryType) ([]Memory, error)
}
