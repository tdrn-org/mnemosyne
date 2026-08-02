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

package vectordb

import (
	"context"
	"fmt"

	"github.com/qdrant/go-client/qdrant"
	"github.com/tdrn-org/mnemosyne/internal/domain"
)

const knowledgeCollection string = "knowledge"

func (s *Store) initKnowledgeCollection(ctx context.Context, dimension int, reset bool) error {
	collectionName := s.collectionName(knowledgeCollection)
	exists, err := s.client.CollectionExists(ctx, collectionName)
	if err != nil {
		return fmt.Errorf("failed to check collection '%s' (cause: %w)", collectionName, err)
	}
	if exists {
		if !reset {
			return nil
		}
		err = s.client.DeleteCollection(ctx, collectionName)
		if err != nil {
			return fmt.Errorf("failed to delete collection '%s' (cause: %w)", collectionName, err)
		}
	}
	err = s.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: collectionName,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     uint64(dimension),
			Distance: qdrant.Distance_Cosine,
		}),
	})
	if err != nil {
		return fmt.Errorf("failed to create collection '%s' (cause: %w)", collectionName, err)
	}
	return nil
}

func (s *Store) UpsertChunk(ctx context.Context, chunk *domain.Chunk, embedding ...float32) error {
	point, err := EncodeToPoint(chunk)
	if err != nil {
		return err
	}
	point.Vectors = qdrant.NewVectors(embedding...)
	_, err = s.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: s.collectionName(knowledgeCollection),
		Points: []*qdrant.PointStruct{
			point,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to upsert chunk '%s' (cause: %w)", chunk.DocumentTitle, err)
	}
	return nil

}

func (s *Store) SearchChunks(ctx context.Context, limit *uint64, vector ...float32) ([]domain.Chunk, error) {
	res, err := s.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: s.collectionName(knowledgeCollection),
		Query:          qdrant.NewQuery(vector...),
		Limit:          limit,
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query chunks (cause: %w)", err)
	}
	chunks := make([]domain.Chunk, 0, len(res))
	for _, found := range res {
		chunk := &domain.Chunk{}
		err = DecodeFromPoint(chunk, found)
		if err != nil {
			return nil, fmt.Errorf("failed to decode chunk query result (cause: %w)", err)
		}
		chunks = append(chunks, *chunk)
	}
	return chunks, nil
}
