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

package ollama

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/ollama/ollama/api"
	"github.com/tdrn-org/mnemosyne/config"
)

type Provider struct {
	cfg    *config.OllamaProviderConfig
	client *api.Client
	logger *slog.Logger
}

func NewCloudProvider(cfg *config.OllamaProviderConfig) *Provider {
	logger := slog.With(slog.String("provider", string(config.ProviderNameOllamaCloud)))
	httpClient := &http.Client{}
	if cfg.APIKey != "" {
		httpClient.Transport = &authorizationTransport{
			apiKey: cfg.APIKey,
			base:   http.DefaultTransport,
		}
	} else {
		logger.Warn("no API key set")
	}
	client := api.NewClient(cfg.BaseURL.URL, httpClient)
	p := &Provider{
		cfg:    cfg,
		client: client,
		logger: logger,
	}
	return p
}

func NewProvider(cfg *config.OllamaProviderConfig) *Provider {
	logger := slog.With(slog.String("provider", string(config.ProviderNameOllama)))
	httpClient := &http.Client{}
	if cfg.APIKey != "" {
		httpClient.Transport = &authorizationTransport{
			apiKey: cfg.APIKey,
			base:   http.DefaultTransport,
		}
	}
	client := api.NewClient(cfg.BaseURL.URL, httpClient)
	p := &Provider{
		cfg:    cfg,
		client: client,
		logger: logger,
	}
	return p
}

func (p *Provider) EmbeddingDimension() uint64 {
	return p.cfg.EmbeddingDimension
}

func (p *Provider) Embed(ctx context.Context, text string) ([]float32, error) {
	req := &api.EmbedRequest{
		Model: p.cfg.EmbeddingModel,
		Input: []string{text},
	}
	rsp, err := p.client.Embed(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("embed failure on Ollama model: '%s' (cause: %w)", req.Model, err)
	}
	switch len(rsp.Embeddings) {
	case 0:
		return nil, fmt.Errorf("empty embed result from Ollama model: '%s'", req.Model)
	case 1:
		// as expected
	default:
		p.logger.Info("multiple embeddings returned; ignoring additional ones", slog.Int("embeddingsCount", len(rsp.Embeddings)))
	}
	return rsp.Embeddings[0], nil
}
