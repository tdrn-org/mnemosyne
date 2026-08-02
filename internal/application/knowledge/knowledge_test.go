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

package knowledge_test

import (
	"testing"

	"github.com/gorhill/cronexpr"
	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/mnemosyne/config"
	"github.com/tdrn-org/mnemosyne/internal/application/knowledge"
	"github.com/tdrn-org/mnemosyne/internal/provider"
	"github.com/tdrn-org/mnemosyne/internal/tokenizer"
	"github.com/tdrn-org/mnemosyne/internal/vectordb"
)

func TestKnowledgeSync(t *testing.T) {
	store, embedder := testStore(t)
	tokenizer := &tokenizer.EstimateTokenizer{RunesPerToken: 4}
	cfg := &config.KnowledgeConfig{
		MarkdownSources: []config.MarkdownSourceConfig{
			{
				Store:  t.Name(),
				Path:   "testdata/",
				Nature: config.MarkdownNatureObsidian,
				Schedule: config.ScheduleSpec{
					Expression: cronexpr.MustParse("0 * * * *"),
				},
			},
		},
	}
	knowledge := knowledge.NewKnowledge(cfg, store, tokenizer, embedder)
	knowledge.Sync(t.Context())
}

func testStore(t *testing.T) (*vectordb.Store, provider.Embedder) {
	embedder := testProvider(t)
	_ = embedder // used below, but test may skip before reaching
	store, err := vectordb.Open(testConfig(t), embedder.EmbeddingDimension(), true)
	require.NoError(t, err)
	return store, embedder
}

func testConfig(t *testing.T) *config.VectorDBConfig {
	t.SkipNow()
	return &config.VectorDBConfig{
		Address:                "localhost:6334",
		TLS:                    true,
		SkipCompatibilityCheck: true,
		Tenant:                 "test",
	}
}

func testProvider(_ *testing.T) provider.Embedder {
	return provider.NewDemoProvider(&config.DemoProviderConfig{EmbeddingDimension: 256})
}
