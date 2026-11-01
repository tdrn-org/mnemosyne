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

package vectordb_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/mnemosyne/internal/domain"
)

func TestChunks(t *testing.T) {
	vectorDB, provider := testVectorDB(t)
	defer vectorDB.Close()

	path := "document.md"
	chunk := &domain.Chunk{
		ID:            uuid.NewString(),
		Path:          path,
		ChunkIndex:    0,
		ChunkHash:     "1234567890",
		LastSync:      domain.Timestamp(time.Now()),
		DocumentTitle: "A section about everything",
		HeadingPath:   []string{"A documment about everything"},
		Tags:          []string{},
		Links:         []string{},
		Content:       "Lorem ipsum dolor sit amet, consetetur sadipscing elitr, sed diam nonumy eirmod tempor invidunt ut labore et dolore magna aliquyam erat, sed diam voluptua.",
	}
	vector, err := provider.Embed(t.Context(), chunk.Content)
	require.NoError(t, err)

	err = vectorDB.UpsertChunk(t.Context(), chunk, vector...)
	require.NoError(t, err)

	chunks, err := vectorDB.SearchChunks(t.Context(), nil, nil, vector...)
	require.NoError(t, err)
	require.Len(t, chunks, 1)

	count, err := vectorDB.CountChunks(t.Context(), nil)
	require.NoError(t, err)
	require.Equal(t, uint64(1), count)
}
