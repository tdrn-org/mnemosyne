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

type ServerConfig struct {
	Address            string              `toml:"address"`
	Protocol           ServerProtocol      `toml:"protocol"`
	CertFile           string              `toml:"cert_file"`
	KeyFile            string              `toml:"key_file"`
	PublicURL          config.URLSpec      `toml:"public_url"`
	TrustedProxies     config.NetworkSpecs `toml:"trusted_proxies"`
	TrustedHeaders     []string            `toml:"trusted_headers"`
	AllowedOrigins     []string            `toml:"allowed_origins"`
	AccessLog          string              `toml:"access_log"`
	AccessLogSizeLimit int64               `toml:"access_log_size_limit"`
}

type ServerProtocol string

const (
	ServerProtocolHttp  ServerProtocol = "http"
	ServerProtocolHttps ServerProtocol = "https"
)

var serverProtocolMarshalMap map[ServerProtocol]string = map[ServerProtocol]string{
	ServerProtocolHttp:  string(ServerProtocolHttp),
	ServerProtocolHttps: string(ServerProtocolHttps),
}

var serverProtocolUnmarshalMap map[string]ServerProtocol = map[string]ServerProtocol{
	string(ServerProtocolHttp):  ServerProtocolHttp,
	string(ServerProtocolHttps): ServerProtocolHttps,
}

func (p ServerProtocol) MarshalText() ([]byte, error) {
	return config.MarshalEnum(p, serverProtocolMarshalMap)
}

func (p *ServerProtocol) UnmarshalText(text []byte) error {
	protocol, err := config.UnmarshalEnum(serverProtocolUnmarshalMap, text)
	if err != nil {
		return err
	}
	*p = protocol
	return nil
}
