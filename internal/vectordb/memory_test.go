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

	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/mnemosyne/internal/domain"
)

func TestMemories(t *testing.T) {
	vectorDB, provider := testVectorDB(t)
	defer vectorDB.Close()

	now := time.Now()
	content := "A memory of something"
	typ := "moment"
	memory := &domain.Memory{
		ID:         domain.MemoryID(typ, content),
		Content:    content,
		Type:       typ,
		Trust:      0.5,
		CreatedAt:  now,
		ExpiresAt:  now.Add(time.Hour),
		LastAccess: now,
	}
	vector, err := provider.Embed(t.Context(), memory.Content)
	require.NoError(t, err)

	m1, err := vectorDB.LookupMemory(t.Context(), memory.ID)
	require.NoError(t, err)
	require.Nil(t, m1)

	err = vectorDB.UpsertMemory(t.Context(), memory, vector...)
	require.NoError(t, err)

	m2, err := vectorDB.LookupMemory(t.Context(), memory.ID)
	require.NoError(t, err)
	require.Equal(t, memory.ID, m2.ID)

	memories, err := vectorDB.SearchMemories(t.Context(), nil, nil, nil, vector...)
	require.NoError(t, err)
	require.Len(t, memories, 1)

	err = vectorDB.DeleteMemory(t.Context(), memory.ID)
	require.NoError(t, err)
}
