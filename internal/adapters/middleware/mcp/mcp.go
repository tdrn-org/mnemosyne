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
	"context"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tdrn-org/mnemosyne/internal/buildinfo"
	"github.com/tdrn-org/mnemosyne/internal/domain"
)

// Runtime provides the MCP handler access to the server's services and configuration.
type Runtime interface {
	BaseURL() *url.URL
	Knowledge() domain.Knowledge
	Logger() *slog.Logger
}

func NewHandler(runtime Runtime) http.Handler {
	impl := &mcp.Implementation{
		Name:       buildinfo.Cmd(),
		Version:    buildinfo.Version(),
		WebsiteURL: runtime.BaseURL().String(),
	}
	options := &mcp.ServerOptions{
		Logger: runtime.Logger(),
	}
	server := mcp.NewServer(impl, options)

	// Log tool failures in a generic manner
	server.AddReceivingMiddleware(func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			result, err := next(ctx, method, req)
			if err != nil {
				runtime.Logger().Warn("mcp tool call failure", slog.String("method", method), slog.Any("err", err))
			}
			return result, err
		}
	})

	getServer := func(_ *http.Request) *mcp.Server {
		return server
	}
	return mcp.NewStreamableHTTPHandler(getServer, nil)
}
