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
	_ "embed"
	"path/filepath"
	"strings"

	"github.com/tdrn-org/go-config-toml"
)

type Config struct {
	Logging   LoggingConfig   `toml:"logging"`
	Server    ServerConfig    `toml:"server"`
	VectorDB  VectorDBConfig  `toml:"vectordb"`
	Provider  ProviderConfig  `toml:"provider"`
	Knowledge KnowledgeConfig `toml:"knowledge"`
	Memory    MemoryConfig    `toml:"memory"`
}

//go:embed defaults.toml
var defaultsData []byte

func Default() (*Config, error) {
	return config.Defaults(&Config{}, defaultsData)
}

func Load(path string, strict bool) (*Config, error) {
	return config.Load(&Config{}, path, defaultsData, strict)
}

type PathFilter struct {
	Path    string   `toml:"path"`
	Include []string `toml:"include"`
	Exclude []string `toml:"exclude"`
}

func (f PathFilter) Match(name string) bool {
	include := len(f.Include) == 0
	for _, prefix := range f.Include {
		fullPrefix := filepath.Join(f.Path, prefix)
		match := strings.HasPrefix(name, fullPrefix)
		if !match {
			continue
		}
		include = true
		break
	}
	if !include {
		return false
	}
	for _, prefix := range f.Exclude {
		fullPrefix := filepath.Join(f.Path, prefix)
		match := strings.HasPrefix(name, fullPrefix)
		if !match {
			continue
		}
		return false
	}
	return true
}
