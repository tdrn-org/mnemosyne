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
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/tdrn-org/mnemosyne/internal/application/knowledge"
)

const Namespace string = "mnemosyne"
const SubsystemKnowledge string = "knowledge"
const SubsystemMemory string = "memory"

type Recorder struct {
	enabled                 bool
	knowledgeDocuments      *prometheus.GaugeVec
	knowledgeChunks         *prometheus.GaugeVec
	knowledgeSearchRequests prometheus.Counter
	knowledgeReadRequests   prometheus.Counter
}

func NewRecorder(registry *prometheus.Registry) *Recorder {
	recorder := &Recorder{
		enabled: registry != nil,
	}
	if !recorder.enabled {
		return recorder
	}
	factory := promauto.With(registry)
	knowledgeLabels := []string{"store"}
	recorder.knowledgeDocuments = factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: SubsystemKnowledge,
		Name:      "documents",
	}, knowledgeLabels)
	recorder.knowledgeChunks = factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: SubsystemKnowledge,
		Name:      "chunks",
	}, knowledgeLabels)
	recorder.knowledgeSearchRequests = factory.NewCounter(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: SubsystemKnowledge,
		Name:      "search_requests_total",
	})
	recorder.knowledgeReadRequests = factory.NewCounter(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: SubsystemKnowledge,
		Name:      "read_requests_total",
	})
	return recorder
}

func (r *Recorder) RecordKnowledgeStats(stats *knowledge.Stats) {
	if !r.enabled {
		return
	}

	for store, count := range stats.DocumentCounts {
		knowledgeLabels := []string{store}
		r.setGauge("current", r.knowledgeDocuments, float64(count), knowledgeLabels...)
	}
	for store, count := range stats.ChunkCounts {
		knowledgeLabels := []string{store}
		r.setGauge("current", r.knowledgeChunks, float64(count), knowledgeLabels...)
	}
	r.knowledgeSearchRequests.Add(float64(stats.SearchRequestsDelta))
	r.knowledgeReadRequests.Add(float64(stats.ReadRequestsDelta))
}

func (r *Recorder) setGauge(name string, gaugeVec *prometheus.GaugeVec, value float64, labels ...string) {
	gauge, err := gaugeVec.GetMetricWithLabelValues(labels...)
	if err == nil {
		gauge.Set(value)
	} else {
		slog.Warn("collect metric failure", slog.String("name", name), slog.Any("err", err))
	}
}
