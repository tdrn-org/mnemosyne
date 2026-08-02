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
)

type Chunk struct {
	ID            string   `json:"id"`
	Store         string   `json:"store"`
	Path          string   `json:"path"`
	ChunkIndex    int64    `json:"chunk_index"`
	ChunkHash     string   `json:"chunk_hash"`
	DocumentTitle string   `json:"document_title"`
	HeadingPath   []string `json:"heading_path"`
	Tags          []string `json:"tags"`
	Links         []string `json:"links"`
	Content       string   `json:"content"`
}

type ChunkStore interface {
	UpsertChunk(ctx context.Context, chunk *Chunk, embedding ...float32) error
	SearchChunks(ctx context.Context, limit *uint64, vector ...float32) ([]Chunk, error)
}
