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
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tdrn-org/mnemosyne/internal/domain"
)

func registerMemoryTools(server *mcp.Server, runtime Runtime) {
	// list_memory_types: list configured memory types
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_memory_types",
		Description: "Lists all configured memory types with their names, TTLs, and descriptions. Memory types define categories like 'fact', 'decision', or 'session_summary'. Use this to discover valid type values before creating or recalling memories.",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input struct{}) (*mcp.CallToolResult, any, error) {
		types, err := runtime.Memory().ListMemoryTypes(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("listing memory types: %w", err)
		}
		text := fmt.Sprintf("Found %d memory type(s)", len(types))
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: text}},
		}, map[string]any{"memory_types": types}, nil
	})

	// remember_memory: store a new memory
	mcp.AddTool(server, &mcp.Tool{
		Name:        "remember_memory",
		Description: "Stores a new memory entry. The content is embedded and stored for later semantic recall via recall_memories. Memories have a type (e.g. 'fact', 'decision'), an optional trust score (0.0–1.0, default 0.5), and optional labels for filtering.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"content": map[string]any{"type": "string", "description": "The memory content text. This is embedded and used for semantic search."},
				"type":    map[string]any{"type": "string", "description": "The memory type (e.g. 'fact', 'decision', 'session_summary'). Use list_memory_types to see available types."},
				"trust":   map[string]any{"type": "number", "description": "Optional trust score from 0.0 (unverified) to 1.0 (certain). Default: 0.5."},
				"labels":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Optional labels for filtering."},
			},
			"required": []string{"content", "type"},
		},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input struct {
		Content string   `json:"content"`
		Type    string   `json:"type"`
		Trust   *float64 `json:"trust"`
		Labels  []string `json:"labels"`
	}) (*mcp.CallToolResult, any, error) {
		now := time.Now()
		trust := 0.5
		if input.Trust != nil {
			trust = *input.Trust
		}
		memory := &domain.Memory{
			ID:         domain.MemoryID(),
			Content:    input.Content,
			Type:       input.Type,
			Trust:      trust,
			Labels:     input.Labels,
			CreatedAt:  now,
			LastAccess: now,
		}
		err := runtime.Memory().RememberMemory(ctx, memory)
		if err != nil {
			return nil, nil, fmt.Errorf("storing memory: %w", err)
		}
		text := fmt.Sprintf("Memory stored with ID: %s", memory.ID)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: text}},
		}, map[string]any{"id": memory.ID}, nil
	})

	// recall_memories: semantic search across memories
	mcp.AddTool(server, &mcp.Tool{
		Name:        "recall_memories",
		Description: "Searches stored memories by semantic similarity to the query. Optionally filter by memory type, minimum trust score, or result limit. Returns matching memories with their content, type, trust, and metadata.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query":       map[string]any{"type": "string", "description": "The search query. Natural language is supported — the query is embedded and compared against memory vectors via cosine similarity."},
				"type_filter": map[string]any{"type": "string", "description": "Optional memory type to restrict results to (e.g. 'fact', 'decision')."},
				"min_trust":   map[string]any{"type": "number", "description": "Optional minimum trust score (0.0–1.0). Only memories with trust >= this value are returned."},
				"limit":       map[string]any{"type": "number", "description": "Optional maximum number of memories to return."},
			},
			"required": []string{"query"},
		},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input struct {
		Query      string   `json:"query"`
		TypeFilter *string  `json:"type_filter"`
		MinTrust   *float64 `json:"min_trust"`
		Limit      *uint64  `json:"limit"`
	}) (*mcp.CallToolResult, any, error) {
		opts := domain.RecallOptions{
			TypeFilter: input.TypeFilter,
			MinTrust:   input.MinTrust,
			Limit:      input.Limit,
		}
		memories, err := runtime.Memory().RecallMemories(ctx, input.Query, opts)
		if err != nil {
			return nil, nil, fmt.Errorf("recalling memories: %w", err)
		}
		text := fmt.Sprintf("Found %d memory/memories", len(memories))
		result := &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: text}},
		}
		var memoriesMeta any
		if len(memories) > 0 {
			memoriesMeta = map[string]any{"memories": memories}
		}
		return result, memoriesMeta, nil
	})
}
