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
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMemoryTouchNever(t *testing.T) {
	now := time.Now()
	m := &Memory{
		ExpiresAt:  time.Time{},
		LastAccess: now.Add(-time.Hour),
	}
	m.Touch()
	require.True(t, m.ExpiresAt.IsZero(), "never memory must keep zero ExpiresAt after Touch")
	require.True(t, m.LastAccess.After(now.Add(-time.Minute)), "LastAccess must be refreshed")
}

func TestMemoryTouchExpiring(t *testing.T) {
	now := time.Now()
	ttl := 24 * time.Hour
	m := &Memory{
		ExpiresAt:  now.Add(ttl),
		LastAccess: now,
	}
	remainingBefore := m.ExpiresAt.Sub(m.LastAccess)
	m.Touch()
	remainingAfter := m.ExpiresAt.Sub(m.LastAccess)
	require.Equal(t, remainingBefore, remainingAfter, "remaining TTL must be preserved")
	require.True(t, m.LastAccess.After(now), "LastAccess must advance")
}
