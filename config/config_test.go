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

package config_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/mnemosyne/config"
)

func TestEmptyPathFilter(t *testing.T) {
	filter := &config.PathFilter{}

	match := filter.Match(".trash")
	require.True(t, match)
}

func TestIncludePathFilter(t *testing.T) {
	filter := &config.PathFilter{
		Include: []string{"HowTo/"},
	}

	match := filter.Match(".trash/Ideas.md")
	require.False(t, match)

	match = filter.Match("HowTo/Testing.md")
	require.True(t, match)
}

func TestExcludePathFilter(t *testing.T) {
	filter := &config.PathFilter{
		Exclude: []string{".trash/"},
	}

	match := filter.Match(".trash/Ideas.md")
	require.False(t, match)

	match = filter.Match("HowTo/Testing.md")
	require.True(t, match)
}

func TestIncludeExcludePathFilter(t *testing.T) {
	filter := &config.PathFilter{
		Include: []string{"Vault/"},
		Exclude: []string{"Vault/.trash/"},
	}

	match := filter.Match("Vault/.trash/Ideas.md")
	require.False(t, match)

	match = filter.Match("Vault/HowTo/Testing.md")
	require.True(t, match)
}
