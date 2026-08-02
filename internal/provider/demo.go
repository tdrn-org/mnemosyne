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

package provider

import (
	"context"
	"crypto/md5"
	"encoding/binary"
	"math"
	"math/rand/v2"

	"github.com/tdrn-org/mnemosyne/config"
)

type DemoProvider struct {
	cfg *config.DemoProviderConfig
}

func NewDemoProvider(cfg *config.DemoProviderConfig) *DemoProvider {
	return &DemoProvider{
		cfg: cfg,
	}
}

func (p *DemoProvider) EmbeddingDimension() uint64 {
	return p.cfg.EmbeddingDimension
}

func (p *DemoProvider) Embed(_ context.Context, text string) ([]float32, error) {
	embedding := make([]float32, p.cfg.EmbeddingDimension)
	hash := md5.Sum([]byte(text))
	seed := binary.BigEndian.Uint64(hash[:8])
	rng := rand.New(rand.NewPCG(seed, 0))
	var sumSq float64
	for i := range embedding {
		val := rng.NormFloat64()
		embedding[i] = float32(val)
		sumSq += val * val
	}
	magnitude := math.Sqrt(sumSq)
	if magnitude > 0 {
		for i := range embedding {
			embedding[i] = float32(float64(embedding[i]) / magnitude)
		}
	}
	return embedding, nil
}
