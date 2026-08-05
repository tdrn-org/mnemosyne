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
)

// MemoryType classifies a memory entry. Valid values are defined in
// the config through [config.MemoryTypeConfig], not hardcoded here.
type MemoryType string

// Well-known memory types that should be configured by default.
const (
	MemoryFact            MemoryType = "fact"
	MemoryDecision        MemoryType = "decision"
	MemorySessionSummary  MemoryType = "session_summary"
	MemoryEmotionalMoment MemoryType = "emotional"
	MemoryContextSwitch   MemoryType = "context"
)

// MemorySource describes where a memory entry originated.
type MemorySource struct {
	// Origin identifies the source kind: "holger:explicit", "judy:inference", "session".
	Origin string `json:"origin"`
	// SessionID is optionally set when the memory originates from a Hermes session.
	SessionID *string `json:"session_id,omitempty"`
	// Timestamp is when the memory was originally created or observed.
	Timestamp time.Time `json:"timestamp"`
}

// MemoryEntry is the core domain type for a single memory record.
type MemoryEntry struct {
	// ID is a deterministic UUIDv5 derived from content + source.
	ID string `json:"id"`
	// Content is the raw memory text.
	Content string `json:"content"`
	// Type classifies the memory (fact, decision, emotional, etc.).
	Type MemoryType `json:"type"`
	// Agents lists the agents involved (e.g. ["judy", "holger"]).
	Agents []string `json:"agents"`
	// Source records where this memory originated.
	Source MemorySource `json:"source"`
	// Trust is a 0.0–1.0 score indicating confidence in this memory.
	Trust float64 `json:"trust"`
	// CreatedAt is when the memory was stored.
	CreatedAt time.Time `json:"created_at"`
	// ExpiresAt is the calculated expiry. Zero time means never expires.
	ExpiresAt time.Time `json:"expires_at,omitempty"`
	// LastAccess is updated every time the memory is accessed (for TTL reset).
	LastAccess time.Time `json:"last_access"`
	// CrossRefs link to related knowledge stores.
	CrossRefs []CrossReference `json:"cross_refs,omitempty"`
	// Labels are free-form tags for filtering.
	Labels []string `json:"labels,omitempty"`
	// Embedding is the vector representation (set by the store).
	Embedding []float32 `json:"-"`
}

// CrossReference links a memory entry to a Knowledge store chunk.
type CrossReference struct {
	// Store is the knowledge store name (e.g. "idpd").
	Store string `json:"store"`
	// ChunkPath optionally points to a specific chunk path within the store.
	ChunkPath *string `json:"chunk_path,omitempty"`
	// Relevance is 0–1 indicating how relevant this link is.
	Relevance float64 `json:"relevance"`
}

// RecallOptions controls how memories are retrieved.
type RecallOptions struct {
	Limit       *uint64
	MinTrust    *float64
	Agent       *string
	TypeFilter  *MemoryType
	StoreFilter *string
}

// MemoryStore is the domain interface for the memory collection.
type MemoryStore interface {
	// Remember stores a new memory entry. The store is responsible for
	// generating the embedding and setting the ID if not provided.
	Remember(ctx context.Context, entry *MemoryEntry) error

	// Forget removes a memory entry by ID.
	Forget(ctx context.Context, id string) error

	// Recall searches memories by semantic similarity to the query.
	Recall(ctx context.Context, query string, opts RecallOptions) ([]MemoryEntry, error)

	// Reinforce adjusts the trust score of a memory entry.
	// Positive delta increases trust, negative decreases it.
	Reinforce(ctx context.Context, id string, trustDelta float64) error

	// Touch resets the expiry timer for a memory entry.
	Touch(ctx context.Context, id string) error

	// Expire removes all memories that have passed their expiry time.
	// Returns the count of removed entries.
	Expire(ctx context.Context) (int, error)

	// ListByType returns all memories of a given type.
	ListByType(ctx context.Context, memType MemoryType) ([]MemoryEntry, error)

	// ListByAgent returns all memories associated with a given agent.
	ListByAgent(ctx context.Context, agent string) ([]MemoryEntry, error)
}
