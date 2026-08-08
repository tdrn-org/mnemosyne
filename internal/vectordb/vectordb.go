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
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strconv"

	"github.com/qdrant/go-client/qdrant"
	"github.com/tdrn-org/mnemosyne/config"
)

var waitAlyways bool = true
var waitCreateIndex bool = true
var waitDelete bool = true

type Store struct {
	client *qdrant.Client
	tenant string
	logger *slog.Logger
}

func Open(cfg *config.VectorDBConfig, dimension uint64, reset bool) (*Store, error) {
	client, err := qdrant.NewClient(qdrantConfig(cfg))
	if err != nil {
		return nil, fmt.Errorf("failed to create Qdrant client (cause: %w)", err)
	}
	store := &Store{
		client: client,
		tenant: cfg.Tenant,
		logger: slog.With(slog.String("vectordb", "qdrant"), slog.String("address", cfg.Address)),
	}
	err = store.update(context.Background(), dimension, reset)
	if err != nil {
		return nil, errors.Join(err, store.Close())
	}
	return store, nil
}

func qdrantConfig(cfg *config.VectorDBConfig) *qdrant.Config {
	host, portString, err := net.SplitHostPort(cfg.Address)
	port := 0
	if err == nil {
		port, err = strconv.Atoi(portString)
		if err != nil {
			host = cfg.Address
		}
	}
	config := &qdrant.Config{
		Host:                   host,
		Port:                   port,
		APIKey:                 cfg.APIKey,
		UseTLS:                 cfg.TLS,
		SkipCompatibilityCheck: cfg.SkipCompatibilityCheck,
	}
	return config
}

func (s *Store) Close() error {
	return s.client.Close()
}

func (s *Store) collectionName(name string) string {
	if s.tenant == "" {
		return name
	}
	return s.tenant + "_" + name
}

func (s *Store) update(ctx context.Context, dimension uint64, reset bool) error {
	updates := s.documentCollectionUpdates(reset)
	updates = append(updates, s.knowledgeCollectionUpdates(dimension, reset)...)
	updates = append(updates, s.memoryCollectionUpdates(dimension, reset)...)
	for _, update := range updates {
		err := update.Apply(s.client, ctx, s.logger)
		if err != nil {
			return err
		}
	}
	return nil
}

type schemaUpdate interface {
	Apply(client *qdrant.Client, ctx context.Context, logger *slog.Logger) error
}

type schemaUpdateCollection struct {
	CollectionName string
	VectorsConfig  *qdrant.VectorsConfig
	Reset          bool
}

func (u *schemaUpdateCollection) Apply(client *qdrant.Client, ctx context.Context, logger *slog.Logger) error {
	collectionLogger := logger.With(slog.String("collection", u.CollectionName))
	collectionLogger.Info("updating collection...")
	exists, err := client.CollectionExists(ctx, u.CollectionName)
	if err != nil {
		return fmt.Errorf("failed to check collection '%s' (cause: %w)", u.CollectionName, err)
	}
	if exists && u.Reset {
		collectionLogger.Info("deleting collection...")
		err = client.DeleteCollection(ctx, u.CollectionName)
		if err != nil {
			return fmt.Errorf("failed to delete collection '%s' (cause: %w)", u.CollectionName, err)
		}
	}
	if !exists || u.Reset {
		collectionLogger.Info("creating collection...")
		err = client.CreateCollection(ctx, &qdrant.CreateCollection{
			CollectionName: u.CollectionName,
			VectorsConfig:  u.VectorsConfig,
		})
		if err != nil {
			return fmt.Errorf("failed to create collection '%s' (cause: %w)", u.CollectionName, err)
		}
	}
	return nil
}

type schemaUpdateFieldIndex struct {
	CollectionName string
	FieldName      string
	FieldType      *qdrant.FieldType
	Reset          bool
}

func (u *schemaUpdateFieldIndex) Apply(client *qdrant.Client, ctx context.Context, logger *slog.Logger) error {
	indexLogger := logger.With(slog.String("collection", u.CollectionName), slog.String("field", u.FieldName))
	indexLogger.Info("updating index...")
	info, err := client.GetCollectionInfo(ctx, u.CollectionName)
	if err != nil {
		return fmt.Errorf("failed to get info for collection '%s' (cause: %w)", u.CollectionName, err)
	}
	exists := false
	if info.PayloadSchema != nil {
		_, exists = info.PayloadSchema[u.FieldName]
	}
	if exists && u.Reset {
		indexLogger.Info("deleting index...")
		_, err = client.DeleteFieldIndex(ctx, &qdrant.DeleteFieldIndexCollection{
			CollectionName: u.CollectionName,
			Wait:           &waitAlyways,
			FieldName:      u.FieldName,
		})
		if err != nil {
			return fmt.Errorf("failed to delete index '%s:%s' (cause: %w)", u.CollectionName, u.FieldName, err)
		}
	}
	if !exists || u.Reset {
		indexLogger.Info("creating index...")
		_, err := client.CreateFieldIndex(ctx, &qdrant.CreateFieldIndexCollection{
			CollectionName: u.CollectionName,
			Wait:           &waitCreateIndex,
			FieldName:      u.FieldName,
			FieldType:      u.FieldType,
		})
		if err != nil {
			return fmt.Errorf("failed to create index '%s:%s' (cause: %w)", u.CollectionName, u.FieldName, err)
		}
	}
	return nil
}
