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

var waitCreateIndex bool = true

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
	err = store.init(context.Background(), dimension, reset)
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

func (s *Store) init(ctx context.Context, dimension uint64, reset bool) error {
	err := s.initDocumentsCollection(ctx, reset)
	if err != nil {
		return err
	}
	err = s.initKnowledgeCollection(ctx, dimension, reset)
	if err != nil {
		return err
	}
	err = s.initMemoryCollection(ctx, dimension, reset)
	if err != nil {
		return err
	}
	return nil
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
