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

package config

import (
	"github.com/tdrn-org/go-config-toml"
)

type ProviderName string

const (
	ProviderNameDemo        ProviderName = "demo"
	ProviderNameOllamaCloud ProviderName = "ollama-cloud"
	ProviderNameOllama      ProviderName = "ollama"
)

type ProviderConfig struct {
	Name        ProviderName         `toml:"name"`
	Demo        DemoProviderConfig   `toml:"demo"`
	OllamaCloud OllamaProviderConfig `toml:"ollama_cloud"`
	Ollama      OllamaProviderConfig `toml:"ollama"`
}

type DemoProviderConfig struct {
	EmbeddingDimension uint64 `toml:"embedding_dimension"`
}

type OllamaProviderConfig struct {
	BaseURL            config.URLSpec `toml:"base_url"`
	APIKey             string         `toml:"api_key"`
	EmbeddingModel     string         `toml:"embedding_model"`
	EmbeddingDimension uint64         `toml:"embedding_dimension"`
}

var providerNameMarshalMap map[ProviderName]string = map[ProviderName]string{
	ProviderNameDemo:        string(ProviderNameDemo),
	ProviderNameOllamaCloud: string(ProviderNameOllamaCloud),
	ProviderNameOllama:      string(ProviderNameOllama),
}

var providerNameUnmarshalMap map[string]ProviderName = map[string]ProviderName{
	string(ProviderNameDemo):        ProviderNameDemo,
	string(ProviderNameOllamaCloud): ProviderNameOllamaCloud,
	string(ProviderNameOllama):      ProviderNameOllama,
}

func (n ProviderName) MarshalText() ([]byte, error) {
	return config.MarshalEnum(n, providerNameMarshalMap)
}

func (n *ProviderName) UnmarshalText(text []byte) error {
	name, err := config.UnmarshalEnum(providerNameUnmarshalMap, text)
	if err != nil {
		return err
	}
	*n = name
	return nil
}
