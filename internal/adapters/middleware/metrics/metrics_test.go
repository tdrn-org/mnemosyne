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

package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

// snapshot gathers all metrics from the registry and returns them keyed by
// metric name plus labels, e.g. "mnemosyne_memory_searches_total|result=hit|type=presence".
// Histograms are reduced to their sample count, which is what the tests assert on.
func snapshot(t *testing.T, registry *prometheus.Registry) map[string]float64 {
	t.Helper()
	values := make(map[string]float64)
	families, err := registry.Gather()
	require.NoError(t, err)
	for _, family := range families {
		for _, metric := range family.GetMetric() {
			key := family.GetName()
			for _, label := range metric.GetLabel() {
				key += "|" + label.GetName() + "=" + label.GetValue()
			}
			switch {
			case metric.GetCounter() != nil:
				values[key] = metric.GetCounter().GetValue()
			case metric.GetGauge() != nil:
				values[key] = metric.GetGauge().GetValue()
			case metric.GetHistogram() != nil:
				values[key] = float64(metric.GetHistogram().GetSampleCount())
			}
		}
	}
	return values
}

func TestMemoryObservations(t *testing.T) {
	registry := prometheus.NewRegistry()
	recorder := NewRecorder(registry)

	recorder.ObserveRecall("presence", 3)
	recorder.ObserveRecall("all", 0)
	recorder.ObserveWrite("presence")
	recorder.ObserveWrite("context")
	recorder.ObserveDelete()
	recorder.ObserveTouch()

	values := snapshot(t, registry)
	// A recall returning memories counts as a hit, one returning none as a miss.
	require.Equal(t, 1.0, values["mnemosyne_memory_searches_total|result=hit|type=presence"])
	require.Equal(t, 0.0, values["mnemosyne_memory_searches_total|result=miss|type=presence"])
	require.Equal(t, 1.0, values["mnemosyne_memory_searches_total|result=miss|type=all"])
	require.Equal(t, 0.0, values["mnemosyne_memory_searches_total|result=hit|type=all"])
	// Writes are counted per memory type.
	require.Equal(t, 1.0, values["mnemosyne_memory_writes_total|type=presence"])
	require.Equal(t, 1.0, values["mnemosyne_memory_writes_total|type=context"])
	// Deletes and touches have no type label and are counted globally.
	require.Equal(t, 1.0, values["mnemosyne_memory_deletes_total"])
	require.Equal(t, 1.0, values["mnemosyne_memory_touches_total"])
	// Every recall is observed in the result histogram, including the empty one.
	require.Equal(t, 1.0, values["mnemosyne_memory_search_results|type=presence"])
	require.Equal(t, 1.0, values["mnemosyne_memory_search_results|type=all"])
}

func TestMemoryObservationsDisabled(t *testing.T) {
	recorder := NewRecorder(nil)

	// A disabled recorder must be a usable no-op.
	require.NotPanics(t, func() {
		recorder.ObserveRecall("presence", 1)
		recorder.ObserveWrite("presence")
		recorder.ObserveDelete()
		recorder.ObserveTouch()
	})
}

func TestKnowledgeObservations(t *testing.T) {
	registry := prometheus.NewRegistry()
	recorder := NewRecorder(registry)

	recorder.ObserveDocumentCount("Wissensspeicher", 100)
	recorder.ObserveChunkCount("Wissensspeicher", 1035)
	recorder.ObserveSearch("all", 3, 0.71)
	recorder.ObserveRead("Wissensspeicher", false)

	values := snapshot(t, registry)
	require.Equal(t, 100.0, values["mnemosyne_knowledge_documents|store=Wissensspeicher"])
	require.Equal(t, 1035.0, values["mnemosyne_knowledge_chunks|store=Wissensspeicher"])
	require.Equal(t, 1.0, values["mnemosyne_knowledge_searches_total|result=hit|store=all"])
	require.Equal(t, 1.0, values["mnemosyne_knowledge_reads_total|result=miss|store=Wissensspeicher"])
	require.Equal(t, 1.0, values["mnemosyne_knowledge_search_results|store=all"])
}
