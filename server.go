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

package mnemosyne

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/tdrn-org/go-httpserver"
	"github.com/tdrn-org/mnemosyne/config"
	"github.com/tdrn-org/mnemosyne/internal/adapters/middleware/mcp"
)

const serverJobTickerSchedule time.Duration = 5 * time.Minute

type Server struct {
	cfg                 *config.Config
	httpServer          *httpserver.Instance
	baseURL             *url.URL
	jobTicker           *time.Ticker
	jobTickerShutdown   chan any
	jobTickerShutdownWG sync.WaitGroup
	jobs                []jobFunc
	logger              *slog.Logger
}

func StartServer(ctx context.Context, cfg *config.Config) (*Server, error) {
	// Setup early logger with configuration address (which may not be the final one).
	// We will reset the logger after listener has been created.
	earlyLogger := slog.With(slog.String("server", cfg.Server.Address))
	s := &Server{
		cfg:    cfg,
		logger: earlyLogger,
	}
	startFuncs := []func(context.Context, *config.Config) error{
		s.startHttpServer,
		s.startMCPHandler,
		s.startJobTicker,
	}
	for _, startFunc := range startFuncs {
		err := startFunc(ctx, cfg)
		if err != nil {
			defer s.Close()
			return nil, err
		}
	}
	return s, nil
}

func (s *Server) Run(ctx context.Context) error {
	s.logger.Info("serving HTTP requests...")
	err := s.httpServer.Serve()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	shutdownFuncs := []func(context.Context) error{
		s.shutdownJobTicker,
		s.shutdownHttpServer,
	}
	shutdownErrs := make([]error, 0, len(shutdownFuncs))
	for _, shutdownFunc := range shutdownFuncs {
		shutdownErrs = append(shutdownErrs, shutdownFunc(ctx))
	}
	return errors.Join(shutdownErrs...)
}

func (s *Server) Close() error {
	closeFuncs := []func() error{
		s.closeHttpServer,
	}
	closeErrs := make([]error, 0, len(closeFuncs))
	for _, closeFunc := range closeFuncs {
		closeErrs = append(closeErrs, closeFunc())
	}
	return errors.Join(closeErrs...)
}

func (s *Server) Ping(ctx context.Context) error {
	if s.httpServer == nil {
		return fmt.Errorf("server not started")
	}
	return nil
}

func (s *Server) startHttpServer(ctx context.Context, cfg *config.Config) error {
	s.logger.Info("starting HTTP server...", slog.String("address", cfg.Server.Address))
	httpServerOptions := httpServerOptions(&cfg.Server)
	httpServer, err := httpserver.Listen(ctx, "tcp", cfg.Server.Address, httpServerOptions...)
	if err != nil {
		return err
	}
	s.httpServer = httpServer
	if cfg.Server.PublicURL.URL != nil {
		s.baseURL = cfg.Server.PublicURL.URL
	} else {
		s.baseURL = httpServer.BaseURL()
	}
	// Replace early logger by one attributed with actual URL
	s.logger = slog.With(slog.String("baseURL", s.baseURL.String()))
	return nil
}

func (s *Server) shutdownHttpServer(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}
	s.logger.Info("shutting down HTTP server...")
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) closeHttpServer() error {
	if s.httpServer == nil {
		return nil
	}
	s.logger.Info("closing HTTP server...")
	return s.httpServer.Close()
}

func (s *Server) startMCPHandler(_ context.Context, _ *config.Config) error {
	s.httpServer.Handle("/mcp", mcp.NewHandler(s.runtime()))
	return nil
}

func (s *Server) startJobTicker(_ context.Context, cfg *config.Config) error {
	schedule := serverJobTickerSchedule
	s.logger.Info("starting job ticker...", slog.String("schedule", schedule.String()))
	s.jobs = []jobFunc{}
	s.jobTicker = time.NewTicker(schedule)
	s.jobTickerShutdown = make(chan any)
	s.jobTickerShutdownWG.Go(func() {
		for stopped := false; !stopped; {
			select {
			case <-s.jobTickerShutdown:
				stopped = true
			case <-s.jobTicker.C:
				s.runJobs()
			}
		}
		s.logger.Info("job ticker stopped")
	})
	return nil
}

func (s *Server) shutdownJobTicker(_ context.Context) error {
	s.logger.Info("shutting down job ticker...")
	s.jobTicker.Stop()
	s.jobTickerShutdown <- true
	s.jobTickerShutdownWG.Wait()
	return nil
}

func (s *Server) runJobs() {
	s.logger.Info("running jobs...")
	ctx := context.Background()
	for _, job := range s.jobs {
		job(ctx)
	}
}

type jobFunc func(ctx context.Context)

func (s *Server) runtime() *serverRuntime {
	return &serverRuntime{server: s}
}

type serverRuntime struct {
	server *Server
}

func (runtime *serverRuntime) BaseURL() *url.URL {
	return runtime.server.baseURL
}

func (runtime *serverRuntime) Logger() *slog.Logger {
	return runtime.server.logger
}

func (runtime *serverRuntime) Ping(ctx context.Context) error {
	return nil
}
