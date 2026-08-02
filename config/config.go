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
	"fmt"
	"log/slog"
	"net/netip"
	"net/url"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/gorhill/cronexpr"
)

type Config struct {
	Logging   LoggingConfig   `toml:"logging"`
	Server    ServerConfig    `toml:"server"`
	VectorDB  VectorDBConfig  `toml:"vectordb"`
	Provider  ProviderConfig  `toml:"provider"`
	Knowledge KnowledgeConfig `toml:"knowledge"`
}

//go:embed defaults.toml
var defaultsData string

func Default() (*Config, error) {
	config := &Config{}
	meta, err := toml.Decode(defaultsData, config)
	if err != nil {
		return nil, fmt.Errorf("failed to decode config defaults (cause: %w)", err)
	}
	for _, key := range meta.Undecoded() {
		slog.Warn("unexpected default configuration key", slog.Any("key", key))
	}
	return config, nil
}

func Load(path string, strict bool) (*Config, error) {
	logger := slog.With(slog.String("path", path))
	logger.Info("loading config")
	config, err := Default()
	if err != nil {
		return nil, err
	}
	meta, err := toml.DecodeFile(path, config)
	if err != nil {
		return nil, fmt.Errorf("failed to decode config '%s' (cause: %w)", path, err)
	}
	strictViolation := false
	for _, key := range meta.Undecoded() {
		strictViolation = true
		logger.Warn("unexpected configuration key", slog.Any("key", key))
	}
	if strict && strictViolation {
		return nil, fmt.Errorf("config contains unexpected keys")
	}
	return config, nil
}

type DurationSpec time.Duration

func (spec *DurationSpec) Value() string {
	return time.Duration(*spec).String()
}

func (spec *DurationSpec) MarshalTOML() ([]byte, error) {
	return []byte(`"` + spec.Value() + `"`), nil
}

func (spec *DurationSpec) UnmarshalTOML(value any) error {
	durationString, ok := value.(string)
	if !ok {
		return fmt.Errorf("unexpected duration type %v", value)
	}
	parsedDuration, err := time.ParseDuration(durationString)
	if err != nil {
		return fmt.Errorf("invalid duration: '%s' (cause: %w)", durationString, err)
	}
	*spec = DurationSpec(parsedDuration)
	return nil
}

type URLSpec struct {
	*url.URL
}

func (spec *URLSpec) Value() string {
	if spec.URL == nil {
		return ""
	}
	return spec.String()
}

func (spec *URLSpec) MarshalTOML() ([]byte, error) {
	return []byte(`"` + spec.Value() + `"`), nil
}

func (spec *URLSpec) UnmarshalTOML(value any) error {
	urlString, ok := value.(string)
	if !ok {
		return fmt.Errorf("unexpected URL type %v", value)
	}
	if urlString == "" {
		return nil
	}
	parsedURL, err := url.Parse(urlString)
	if err != nil {
		return fmt.Errorf("invalid URL: '%s' (cause: %w)", urlString, err)
	}
	spec.URL = parsedURL
	return nil
}

type URLSpecs []URLSpec

func (specs URLSpecs) URLs() []*url.URL {
	urls := make([]*url.URL, 0, len(specs))
	for _, spec := range specs {
		urls = append(urls, spec.URL)
	}
	return urls
}

type ScheduleSpec struct {
	*cronexpr.Expression
}

func (spec *ScheduleSpec) Value() string {
	if spec.Expression == nil {
		return ""
	}
	return spec.Value()
}

func (spec *ScheduleSpec) MarshalTOML() ([]byte, error) {
	return []byte(`"` + spec.Value() + `"`), nil
}

func (spec *ScheduleSpec) UnmarshalTOML(value any) error {
	expressionString, ok := value.(string)
	if !ok {
		return fmt.Errorf("unexpected Cron expression type %v", value)
	}
	if expressionString == "" {
		return nil
	}
	parsedExpression, err := cronexpr.Parse(expressionString)
	if err != nil {
		return fmt.Errorf("invalid Cron expression: '%s' (cause: %w)", expressionString, err)
	}
	spec.Expression = parsedExpression
	return nil
}

type NetworkSpec struct {
	netip.Prefix
}

func (spec *NetworkSpec) Value() string {
	return spec.String()
}

func (spec *NetworkSpec) MarshalTOML() ([]byte, error) {
	return []byte(`"` + spec.String() + `"`), nil
}

func (spec *NetworkSpec) UnmarshalTOML(value any) error {
	networkString, ok := value.(string)
	if !ok {
		return fmt.Errorf("unexpected network type %v", value)
	}
	parsedNetwork, err := netip.ParsePrefix(networkString)
	if err != nil {
		return fmt.Errorf("invalid network: '%s' (cause: %w)", networkString, err)
	}
	spec.Prefix = parsedNetwork
	return nil
}

type NetworkSpecs []NetworkSpec

func (specs NetworkSpecs) Prefixes() []netip.Prefix {
	networks := make([]netip.Prefix, 0, len(specs))
	for _, spec := range specs {
		networks = append(networks, spec.Prefix)
	}
	return networks
}

type TimeLocationSpec struct {
	*time.Location
}

func (spec *TimeLocationSpec) Value() string {
	if spec.Location == nil {
		return ""
	}
	return spec.String()
}
