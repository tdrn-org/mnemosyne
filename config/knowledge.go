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

type KnowledgeConfig struct {
	MarkdownSources []MarkdownSourceConfig `toml:"markdown"`
}

type MarkdownNature string

const (
	MarkdownNatureGeneric  MarkdownNature = "generic"
	MarkdownNatureObsidian MarkdownNature = "obsidian"
)

type MarkdownSourceConfig struct {
	Store string `toml:"store"`
	Path  string `toml:"path"`
	PathFilter
	Nature              MarkdownNature `toml:"nature"`
	Schedule            ScheduleSpec   `toml:"schedule"`
	ChunkTokenLimit     int            `toml:"chunk_token_limit"`
	ChunkRenderTemplate string         `toml:"chunk_render_template"`
}

var knownMarkdownNatures map[string]MarkdownNature = map[string]MarkdownNature{
	string(MarkdownNatureGeneric):  MarkdownNatureGeneric,
	string(MarkdownNatureObsidian): MarkdownNatureObsidian,
}

func (n *MarkdownNature) Value() string {
	for value, nature := range knownMarkdownNatures {
		if *n == nature {
			return value
		}
	}
	slog.Warn("unexpected Markdown nature", slog.Any("n", *n))
	return ""
}

func (n *MarkdownNature) MarshalTOML() ([]byte, error) {
	return []byte(`"` + n.Value() + `"`), nil
}

func (n *MarkdownNature) UnmarshalTOML(value any) error {
	natureString, ok := value.(string)
	if !ok {
		return fmt.Errorf("unexpected Markdown nature type %v", value)
	}
	nature, ok := knownMarkdownNatures[natureString]
	if !ok {
		return fmt.Errorf("unknown Markdown nature: '%s'", natureString)
	}
	*n = nature
	return nil
}
