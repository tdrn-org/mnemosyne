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

// MemoryConfig holds the configuration for the memory store.
type MemoryConfig struct {
	// Types defines the known memory types and their default TTLs.
	// A TTL of 0 means the memory entry never expires.
	Types []MemoryTypeConfig `toml:"type"`
}

// MemoryTypeConfig defines a known memory type with its default expiry.
type MemoryTypeConfig struct {
	Name        string       `toml:"name"`
	TTL         DurationSpec `toml:"ttl"`
	Description string       `toml:"description"`
}
