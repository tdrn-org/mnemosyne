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

import "context"

type KnowledgeStore interface {
	// ListStore lists the known knowledge stores.
	// Each knowledge store contains a collection of [Chunk] instances feed from
	// an external knowledge source (e.g. Obsidian).
	ListStores(ctx context.Context) ([]string, error)
	// SearchStore searches the given store for the given query and returns
	// the matching [Chunk] instances.
	SearchStore(ctx context.Context, query string, store *string, limit *uint64) ([]Chunk, error)
}
