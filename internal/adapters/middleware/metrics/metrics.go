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

// ResultHit and ResultMiss are the values of the result label on the
// search counter.
const (
	ResultHit  string = "hit"
	ResultMiss string = "miss"
)

type Recorder struct {
	enabled                 bool
	knowledgeDocuments      *prometheus.GaugeVec
	knowledgeChunks         *prometheus.GaugeVec
	knowledgeSearchRequests prometheus.Counter
	knowledgeReadRequests   prometheus.Counter
	knowledgeSearches       *prometheus.CounterVec
	knowledgeSearchResults  *prometheus.HistogramVec
	knowledgeSearchTopScore *prometheus.HistogramVec
	knowledgeReads          *prometheus.CounterVec
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
		Help:      "Number of documents known to the knowledge store, by store.",
	}, knowledgeLabels)
	recorder.knowledgeChunks = factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: SubsystemKnowledge,
		Name:      "chunks",
		Help:      "Number of chunks known to the knowledge store, by store.",
	}, knowledgeLabels)
	recorder.knowledgeSearchRequests = factory.NewCounter(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: SubsystemKnowledge,
		Name:      "search_requests_total",
		Help:      "Deprecated: total number of knowledge searches, superseded by searches_total.",
	})
	recorder.knowledgeReadRequests = factory.NewCounter(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: SubsystemKnowledge,
		Name:      "read_requests_total",
		Help:      "Deprecated: total number of knowledge reads, superseded by reads_total.",
	})
	recorder.knowledgeSearches = factory.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: SubsystemKnowledge,
		Name:      "searches_total",
		Help:      "Number of knowledge searches, by store and result (hit or miss).",
	}, []string{"store", "result"})
	recorder.knowledgeSearchResults = factory.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: Namespace,
		Subsystem: SubsystemKnowledge,
		Name:      "search_results",
		Help:      "Number of chunks returned by a knowledge search, by store.",
		Buckets:   []float64{0, 1, 2, 3, 5, 10, 25},
	}, []string{"store"})
	recorder.knowledgeSearchTopScore = factory.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: Namespace,
		Subsystem: SubsystemKnowledge,
		Name:      "search_top_score",
		Help:      "Best similarity score of a knowledge search, by store.",
		Buckets:   []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9},
	}, []string{"store"})
	recorder.knowledgeReads = factory.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: SubsystemKnowledge,
		Name:      "reads_total",
		Help:      "Number of knowledge document reads, by store.",
	}, []string{"store"})
	return recorder
}

// ObserveSearch implements knowledge.MetricsRecorder. A search that returns no
// chunks counts as a miss, so hit and miss together always equal the number of
// searches performed.
func (r *Recorder) ObserveSearch(store string, results int, topScore float64) {
	if !r.enabled {
		return
	}
	result := ResultMiss
	if results > 0 {
		result = ResultHit
	}
	r.knowledgeSearches.WithLabelValues(store, result).Inc()
	r.knowledgeSearchResults.WithLabelValues(store).Observe(float64(results))
	r.knowledgeSearchTopScore.WithLabelValues(store).Observe(topScore)
}

// ObserveRead implements knowledge.MetricsRecorder.
func (r *Recorder) ObserveRead(store string) {
	if !r.enabled {
		return
	}
	r.knowledgeReads.WithLabelValues(store).Inc()
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
