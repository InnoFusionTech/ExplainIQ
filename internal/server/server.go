package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
)

// Server wraps http.Server with graceful shutdown
type Server struct {
	*http.Server
	logger   *logrus.Logger
	timeout  time.Duration
}

// Config holds server configuration
type Config struct {
	Addr         string
	Handler      http.Handler
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
	ShutdownTimeout time.Duration
	Logger       *logrus.Logger
}

// New creates a new server with configuration
func New(config Config) *Server {
	if config.Logger == nil {
		config.Logger = logrus.New()
	}

	if config.ShutdownTimeout == 0 {
		config.ShutdownTimeout = 30 * time.Second
	}

	srv := &http.Server{
		Addr:         config.Addr,
		Handler:      config.Handler,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		IdleTimeout:  config.IdleTimeout,
	}

	return &Server{
		Server:  srv,
		logger:  config.Logger,
		timeout: config.ShutdownTimeout,
	}
}

// Start starts the server in a goroutine
func (s *Server) Start() error {
	go func() {
		s.logger.Infof("Server starting on %s", s.Addr)
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Fatalf("Server failed to start: %v", err)
		}
	}()
	return nil
}

// Wait waits for interrupt signal and gracefully shuts down the server
func (s *Server) Wait() error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	s.logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	if err := s.Shutdown(ctx); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}

	s.logger.Info("Server exited")
	return nil
}

// StartAndWait starts the server and waits for shutdown signal
func (s *Server) StartAndWait() error {
	if err := s.Start(); err != nil {
		return err
	}
	return s.Wait()
}





