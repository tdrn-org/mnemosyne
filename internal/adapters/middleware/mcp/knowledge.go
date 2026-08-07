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
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerKnowledgeTools(server *mcp.Server, runtime Runtime) {
	// list_stores: list known knowledge stores
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_knowledge_stores",
		Description: "Lists all known knowledge stores. Each store contains document chunks ingested from an external knowledge source (e.g. Obsidian). Call this first to discover available stores before searching.",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input struct{}) (*mcp.CallToolResult, any, error) {
		stores, err := runtime.Knowledge().ListStores(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("listing stores: %w", err)
		}
		text := fmt.Sprintf("Found %d store(s): %v", len(stores), stores)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: text}},
		}, map[string]any{"stores": stores}, nil
	})

	// search_store: semantic search within a knowledge store
	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_knowledge_store",
		Description: "Searches a knowledge store for the given query and returns matching document chunks. The store parameter is optional — when omitted, all stores are searched. Use list_stores first to discover available store names.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{"type": "string", "description": "The search query. Natural language is supported — the query is embedded and compared against chunk vectors via cosine similarity."},
				"store": map[string]any{"type": "string", "description": "Optional store name to restrict the search to. Omit to search all stores."},
				"limit": map[string]any{"type": "number", "description": "Optional maximum number of chunks to return (default: implementation-defined)."},
			},
			"required": []string{"query"},
		},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input struct {
		Query string  `json:"query"`
		Store *string `json:"store"`
		Limit *uint64 `json:"limit"`
	}) (*mcp.CallToolResult, any, error) {
		chunks, err := runtime.Knowledge().SearchStore(ctx, input.Query, input.Store, input.Limit)
		if err != nil {
			return nil, nil, fmt.Errorf("searching store: %w", err)
		}
		text := fmt.Sprintf("Found %d chunk(s)", len(chunks))
		result := &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: text}},
		}
		var chunksMeta any
		if len(chunks) > 0 {
			chunksMeta = map[string]any{"chunks": chunks}
		}
		return result, chunksMeta, nil
	})

	// read_knowledge_store_document: read a full document from a knowledge store
	mcp.AddTool(server, &mcp.Tool{
		Name:        "read_knowledge_store_document",
		Description: "Reads the full content of a single document from a knowledge store. The document is returned as raw text, optionally truncated to the given rune limit. Use this after search_knowledge_store to retrieve the complete source of a matching chunk.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"store": map[string]any{"type": "string", "description": "The store name containing the document."},
				"path":  map[string]any{"type": "string", "description": "The relative path of the document within the store (e.g. '10_Holger/Ueber Holger.md')."},
				"limit": map[string]any{"type": "number", "description": "Optional rune limit for the returned content. When exceeded, the document is truncated with '...' appended."},
			},
			"required": []string{"store", "path"},
		},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input struct {
		Store string `json:"store"`
		Path  string `json:"path"`
		Limit *int   `json:"limit"`
	}) (*mcp.CallToolResult, any, error) {
		document, err := runtime.Knowledge().ReadDocument(ctx, input.Store, input.Path, input.Limit)
		if err != nil {
			return nil, nil, fmt.Errorf("reading document: %w", err)
		}
		if document == "" {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Document '%s' not found in store '%s'", input.Path, input.Store)}},
			}, nil, nil
		}
		text := fmt.Sprintf("Read document '%s' from store '%s' (%d runes)", input.Path, input.Store, len([]rune(document)))
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: text}},
		}, map[string]any{"document": document}, nil
	})
}
