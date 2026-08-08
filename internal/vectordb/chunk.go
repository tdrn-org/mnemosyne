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
	"log/slog"

	"github.com/qdrant/go-client/qdrant"
	"github.com/tdrn-org/mnemosyne/internal/domain"
)

const knowledgeCollection string = "knowledge"

func (s *Store) knowledgeCollectionUpdates(dimension uint64, reset bool) []schemaUpdate {
	updates := make([]schemaUpdate, 0)
	collectionName := s.collectionName(knowledgeCollection)
	updates = append(updates, &schemaUpdateCollection{
		CollectionName: collectionName,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     dimension,
			Distance: qdrant.Distance_Cosine,
		}),
		Reset: reset,
	})
	updates = append(updates, &schemaUpdateFieldIndex{
		CollectionName: collectionName,
		FieldName:      "store",
		FieldType:      qdrant.FieldType_FieldTypeKeyword.Enum(),
		Reset:          reset,
	})
	return updates
}

func (s *Store) UpsertChunk(ctx context.Context, chunk *domain.Chunk, embedding ...float32) error {
	collectionName := s.collectionName(knowledgeCollection)
	s.logger.Info("upserting chunk...", slog.String("collection", collectionName))
	point, err := EncodeToPoint(chunk)
	if err != nil {
		return err
	}
	point.Vectors = qdrant.NewVectors(embedding...)
	_, err = s.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: collectionName,
		Points: []*qdrant.PointStruct{
			point,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to upsert chunk '%s' (cause: %w)", chunk.DocumentTitle, err)
	}
	return nil
}

func (s *Store) SearchChunks(ctx context.Context, store *string, limit *uint64, vector ...float32) ([]domain.Chunk, error) {
	collectionName := s.collectionName(knowledgeCollection)
	s.logger.Info("searching collection...", slog.String("collection", collectionName))
	filter := &qdrant.Filter{}
	if store != nil {
		filter.Must = append(filter.Must, qdrant.NewMatch("store", *store))
	}
	res, err := s.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: collectionName,
		Query:          qdrant.NewQuery(vector...),
		Filter:         filter,
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
