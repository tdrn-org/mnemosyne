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

const memoryCollection string = "memory"

func (s *Store) memoryCollectionUpdates(dimension uint64, reset bool) []schemaUpdate {
	updates := make([]schemaUpdate, 0)
	collectionName := s.collectionName(memoryCollection)
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
		FieldName:      "type",
		FieldType:      qdrant.FieldType_FieldTypeKeyword.Enum(),
		Reset:          reset,
	})
	updates = append(updates, &schemaUpdateFieldIndex{
		CollectionName: collectionName,
		FieldName:      "trust",
		FieldType:      qdrant.FieldType_FieldTypeFloat.Enum(),
		Reset:          reset,
	})
	return updates
}

func (s *Store) UpsertMemory(ctx context.Context, memory *domain.Memory, embedding ...float32) error {
	collectionName := s.collectionName(memoryCollection)
	s.logger.Info("upserting memory...", slog.String("collection", collectionName))
	point, err := EncodeToPoint(memory)
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
		return fmt.Errorf("failed to upsert memory '%s' (cause: %w)", memory.ID, err)
	}
	return nil
}

func (s *Store) LookupMemory(ctx context.Context, id string) (*domain.Memory, []float32, error) {
	res, err := s.client.Get(ctx, &qdrant.GetPoints{
		CollectionName: s.collectionName(memoryCollection),
		Ids:            []*qdrant.PointId{qdrant.NewID(id)},
		WithPayload:    qdrant.NewWithPayload(true),
		WithVectors:    qdrant.NewWithVectors(true),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to lookup memory '%s' (cause: %w)", id, err)
	}
	if len(res) == 0 {
		return nil, nil, nil
	}
	memory := &domain.Memory{}
	res0 := res[0]
	err = DecodeFromPoint(memory, res0)
	if err != nil {
		return nil, nil, err
	}
	vector := res0.GetVectors().GetVector().GetDense().GetData()
	return memory, vector, nil
}

func (s *Store) SearchMemories(ctx context.Context, typeFilter *string, minTrust *float64, limit *uint64, vector ...float32) ([]domain.Memory, error) {
	collectionName := s.collectionName(memoryCollection)
	s.logger.Info("searching collection...", slog.String("collection", collectionName))
	filter := &qdrant.Filter{}
	if typeFilter != nil {
		filter.Must = append(filter.Must, qdrant.NewMatch("type", *typeFilter))
	}
	if minTrust != nil {
		filter.Must = append(filter.Must, &qdrant.Condition{
			ConditionOneOf: &qdrant.Condition_Field{
				Field: &qdrant.FieldCondition{
					Key: "trust",
					Range: &qdrant.Range{
						Gte: minTrust,
					},
				},
			},
		})
	}
	res, err := s.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: collectionName,
		Query:          qdrant.NewQuery(vector...),
		Filter:         filter,
		Limit:          limit,
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query memories (cause: %w)", err)
	}
	memories := make([]domain.Memory, 0, len(res))
	for _, found := range res {
		memory := &domain.Memory{}
		err = DecodeFromPoint(memory, found)
		if err != nil {
			return nil, fmt.Errorf("failed to decode memory query result (cause: %w)", err)
		}
		memories = append(memories, *memory)
	}
	return memories, nil
}

func (s *Store) DeleteMemory(ctx context.Context, id string) error {
	_, err := s.client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: s.collectionName(memoryCollection),
		Wait:           &waitDelete,
		Points:         qdrant.NewPointsSelector(qdrant.NewIDUUID(id)),
	})
	if err != nil {
		return fmt.Errorf("failed to delete memory (cause: %w)", err)
	}
	return nil
}
