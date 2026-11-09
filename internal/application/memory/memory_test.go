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
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/mnemosyne/config"
	"github.com/tdrn-org/mnemosyne/internal/domain"
)

type searchObservation struct {
	memoryType string
	results    int
}

// recordingMetricsRecorder captures all observations so they can be asserted on
// without a running vector database.
type recordingMetricsRecorder struct {
	searches []searchObservation
	writes   []string
	deletes  int
	touches  int
}

func (r *recordingMetricsRecorder) ObserveRecall(memoryType string, results int) {
	r.searches = append(r.searches, searchObservation{memoryType: memoryType, results: results})
}

func (r *recordingMetricsRecorder) ObserveWrite(memoryType string) {
	r.writes = append(r.writes, memoryType)
}

func (r *recordingMetricsRecorder) ObserveDelete() {
	r.deletes++
}

func (r *recordingMetricsRecorder) ObserveTouch() {
	r.touches++
}

func TestObserveSearchUnscoped(t *testing.T) {
	recorder := &recordingMetricsRecorder{}
	m := &Memory{metricsRecorder: recorder}

	m.observeSearch(nil, []domain.Memory{{ID: "a"}, {ID: "b"}})

	require.Len(t, recorder.searches, 1)
	require.Equal(t, AllMemoryTypes, recorder.searches[0].memoryType)
	require.Equal(t, 2, recorder.searches[0].results)
}

func TestObserveSearchScoped(t *testing.T) {
	recorder := &recordingMetricsRecorder{}
	m := &Memory{metricsRecorder: recorder}

	memoryType := "presence"
	m.observeSearch(&memoryType, nil)

	require.Len(t, recorder.searches, 1)
	require.Equal(t, "presence", recorder.searches[0].memoryType)
	require.Equal(t, 0, recorder.searches[0].results)
}

func TestListMemoryTypes(t *testing.T) {
	cfg := &config.MemoryConfig{
		Types: []config.MemoryTypeConfig{
			{Name: "presence", Description: "presence memories"},
			{Name: "context", Description: "context memories"},
		},
	}
	m := NewMemory(cfg, nil, nil, &recordingMetricsRecorder{})

	memoryTypes, err := m.ListMemoryTypes(t.Context())
	require.NoError(t, err)
	require.Len(t, memoryTypes, 2)
	// Types are returned sorted by name.
	require.Equal(t, "context", memoryTypes[0].Name)
	require.Equal(t, "presence", memoryTypes[1].Name)
}
