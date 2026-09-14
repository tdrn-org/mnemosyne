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

package mcp

import (
	"strings"
	"testing"
	"time"

	"github.com/tdrn-org/mnemosyne/internal/domain"
)

func TestFormatMemoryTypesText(t *testing.T) {
	types := []domain.MemoryType{
		{Name: "fact", TTL: 0, Description: "objective facts"},
		{Name: "context", TTL: 7 * 24 * time.Hour},
	}
	got := formatMemoryTypesText(types)
	for _, want := range []string{"fact", "never expires", "context", "168h0m0s"} {
		if !strings.Contains(got, want) {
			t.Errorf("formatMemoryTypesText missing %q in:\n%s", want, got)
		}
	}
	if got := formatMemoryTypesText(nil); got != "No memory types configured" {
		t.Errorf("empty types: got %q", got)
	}
}

func TestFormatMemoriesText(t *testing.T) {
	mems := []domain.Memory{
		{
			ID:        "abc-123",
			Content:   "Holger trinkt Augustiner",
			Type:      "fact",
			Trust:     0.9,
			Labels:    []string{"bier"},
			CreatedAt: time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC),
		},
	}
	got := formatMemoriesText(mems)
	for _, want := range []string{"abc-123", "Augustiner", "fact", "0.90", "bier"} {
		if !strings.Contains(got, want) {
			t.Errorf("formatMemoriesText missing %q in:\n%s", want, got)
		}
	}
	if got := formatMemoriesText(nil); got != "No memories found" {
		t.Errorf("empty memories: got %q", got)
	}
}

func TestFormatChunksText(t *testing.T) {
	chunks := []domain.Chunk{
		{
			Store:       "Wissensspeicher",
			Path:        "10_Holger/Ueber Holger.md",
			HeadingPath: []string{"Herkunft"},
			Tags:        []string{"profil"},
			Content:     "Mittler zwischen Hirn und Händen muss das Herz sein.",
		},
	}
	got := formatChunksText(chunks)
	for _, want := range []string{"Wissensspeicher", "10_Holger", "Herkunft", "profil", "Mittler zwischen Hirn"} {
		if !strings.Contains(got, want) {
			t.Errorf("formatChunksText missing %q in:\n%s", want, got)
		}
	}
	if got := formatChunksText(nil); got != "No chunks found" {
		t.Errorf("empty chunks: got %q", got)
	}
}
