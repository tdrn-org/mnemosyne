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
	"fmt"
	"log/slog"
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
	EmbeddingDimension int `toml:"embedding_dimension"`
}

type OllamaProviderConfig struct {
	BaseURL            URLSpec `toml:"base_url"`
	APIKey             string  `toml:"api_key"`
	EmbeddingModel     string  `toml:"embedding_model"`
	EmbeddingDimension int     `toml:"embedding_dimension"`
}

var knownProviderNames map[string]ProviderName = map[string]ProviderName{
	string(ProviderNameDemo):        ProviderNameDemo,
	string(ProviderNameOllamaCloud): ProviderNameOllamaCloud,
	string(ProviderNameOllama):      ProviderNameOllama,
}

func (n *ProviderName) Value() string {
	for value, name := range knownProviderNames {
		if *n == name {
			return value
		}
	}
	slog.Warn("unexpected provider name", slog.Any("n", *n))
	return ""
}

func (n *ProviderName) MarshalTOML() ([]byte, error) {
	return []byte(`"` + n.Value() + `"`), nil
}

func (n *ProviderName) UnmarshalTOML(value any) error {
	nameString, ok := value.(string)
	if !ok {
		return fmt.Errorf("unexpected provider name type %v", value)
	}
	name, ok := knownProviderNames[nameString]
	if !ok {
		return fmt.Errorf("unknown provider name: '%s'", nameString)
	}
	*n = name
	return nil
}
