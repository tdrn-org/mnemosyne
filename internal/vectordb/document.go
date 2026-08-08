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
	return updates
}

func DocumentID(path string) string {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(path)).String()
}

func (s *Store) UpsertDocument(ctx context.Context, document *domain.Document) error {
	point, err := EncodeToPoint(document)
	if err != nil {
		return err
	}
	point.Vectors = qdrant.NewVectorsMap(map[string]*qdrant.Vector{})
	_, err = s.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: s.collectionName(documentsCollection),
		Points: []*qdrant.PointStruct{
			point,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to upsert document '%s' (cause: %w)", document.Path, err)
	}
	return nil
}

func (s *Store) LookupDocument(ctx context.Context, path string) (*domain.Document, error) {
	res, err := s.client.Get(ctx, &qdrant.GetPoints{
		CollectionName: s.collectionName(documentsCollection),
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
