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
	"github.com/tdrn-org/go-jobticker"
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
	PathFilter
	Nature              MarkdownNature         `toml:"nature"`
	Schedule            jobticker.ScheduleSpec `toml:"schedule"`
	ChunkTokenLimit     int                    `toml:"chunk_token_limit"`
	ChunkRenderTemplate string                 `toml:"chunk_render_template"`
}

var markdownNatureMarshalMap map[MarkdownNature]string = map[MarkdownNature]string{
	MarkdownNatureGeneric:  string(MarkdownNatureGeneric),
	MarkdownNatureObsidian: string(MarkdownNatureObsidian),
}

var markdownNatureUnmarshalMap map[string]MarkdownNature = map[string]MarkdownNature{
	string(MarkdownNatureGeneric):  MarkdownNatureGeneric,
	string(MarkdownNatureObsidian): MarkdownNatureObsidian,
}

func (n MarkdownNature) MarshalText() ([]byte, error) {
	return config.MarshalEnum(n, markdownNatureMarshalMap)
}

func (n *MarkdownNature) UnmarshalText(text []byte) error {
	nature, err := config.UnmarshalEnum(markdownNatureUnmarshalMap, text)
	if err != nil {
		return err
	}
	*n = nature
	return nil
}
