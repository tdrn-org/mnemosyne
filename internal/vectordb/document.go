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
	"time"

	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"
	"github.com/tdrn-org/mnemosyne/internal/domain"
)

const documentsCollection string = "documents"

func (s *Store) documentCollectionUpdates(reset bool) []schemaUpdate {
	updates := make([]schemaUpdate, 0)
	collectionName := s.collectionName(documentsCollection)
	updates = append(updates, &schemaUpdateCollection{
		CollectionName: collectionName,
		VectorsConfig:  qdrant.NewVectorsConfigMap(map[string]*qdrant.VectorParams{}),
		Reset:          reset,
	})
	updates = append(updates, &schemaUpdateFieldIndex{
		CollectionName: collectionName,
		FieldName:      "store",
		FieldType:      qdrant.FieldType_FieldTypeKeyword.Enum(),
		Reset:          reset,
	})
	updates = append(updates, &schemaUpdateFieldIndex{
		CollectionName: collectionName,
		FieldName:      "last_sync",
		FieldType:      qdrant.FieldType_FieldTypeDatetime.Enum(),
		Reset:          reset,
	})
	return updates
}

func DocumentID(path string) string {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(path)).String()
}

func (s *Store) UpsertDocument(ctx context.Context, document *domain.Document) error {
	collectionName := s.collectionName(documentsCollection)
	s.logger.Info("upserting document...", slog.String("collection", collectionName))
	point, err := EncodeToPoint(document)
	if err != nil {
		return err
	}
	point.Vectors = qdrant.NewVectorsMap(map[string]*qdrant.Vector{})
	_, err = s.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: collectionName,
		Points: []*qdrant.PointStruct{
			point,
		},
		Wait: &waitAlways,
	})
	if err != nil {
		return fmt.Errorf("failed to upsert document '%s' (cause: %w)", document.Path, err)
	}
	return nil
}

func (s *Store) TouchDocument(ctx context.Context, document *domain.Document) error {
	collectionName := s.collectionName(documentsCollection)
	s.logger.Info("touching document...", slog.String("collection", collectionName))
	now := domain.Timestamp(time.Now())
	_, err := s.client.SetPayload(ctx, &qdrant.SetPayloadPoints{
		CollectionName: collectionName,
		Payload: qdrant.NewValueMap(map[string]any{
			"last_sync": encodeDatetime(now),
		}),
		PointsSelector: qdrant.NewPointsSelector(qdrant.NewIDUUID(document.ID)),
	})
	if err != nil {
		return fmt.Errorf("failed to touch document '%s' (cause: %w)", document.Path, err)
	}
	document.LastSync = now
	return nil
}

func (s *Store) LookupDocument(ctx context.Context, path string) (*domain.Document, error) {
	collectionName := s.collectionName(documentsCollection)
	s.logger.Info("looking up document...", slog.String("collection", collectionName))
	res, err := s.client.Get(ctx, &qdrant.GetPoints{
		CollectionName: collectionName,
		Ids:            []*qdrant.PointId{qdrant.NewID(DocumentID(path))},
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to lookup document '%s' (cause: %w)", path, err)
	}
	if len(res) == 0 {
		return nil, nil
	}
	document := &domain.Document{}
	err = DecodeFromPoint(document, res[0])
	if err != nil {
		return nil, err
	}
	return document, nil
}

func (s *Store) CountDocuments(ctx context.Context, store *string) (uint64, error) {
	collectionName := s.collectionName(documentsCollection)
	s.logger.Info("counting documents...", slog.String("collection", collectionName))
	filter := &qdrant.Filter{}
	if store != nil {
		filter.Must = append(filter.Must, qdrant.NewMatch("store", *store))
	}
	exact := true
	res, err := s.client.Count(ctx, &qdrant.CountPoints{
		CollectionName: collectionName,
		Filter:         filter,
		Exact:          &exact,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to count documents (cause: %w)", err)
	}
	return res, nil
}
