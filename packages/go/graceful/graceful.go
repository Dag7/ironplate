// Package graceful provides utilities for running HTTP servers with
// graceful shutdown on OS signals.
package graceful

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Options configures the graceful server.
type Options struct {
	// ShutdownTimeout is the maximum duration to wait for in-flight
	// requests to complete during shutdown. Default: 30s.
	ShutdownTimeout time.Duration

	// Logger for startup and shutdown messages. If nil, slog.Default() is used.
	Logger *slog.Logger

	// OnShutdown is called after the shutdown signal is received but
	// before the HTTP server begins draining. Use it to close database
	// connections, flush buffers, etc.
	OnShutdown func(ctx context.Context)
}

// ListenAndServe starts the HTTP server and blocks until a SIGINT or SIGTERM
// signal is received, then performs a graceful shutdown.
func ListenAndServe(server *http.Server, opts *Options) error {
	if opts == nil {
		opts = &Options{}
	}
	if opts.ShutdownTimeout == 0 {
		opts.ShutdownTimeout = 30 * time.Second
	}
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	// Channel for server errors.
	errCh := make(chan error, 1)
	go func() {
		logger.Info("server starting", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	// Wait for interrupt signal.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case sig := <-quit:
		logger.Info("shutdown signal received", "signal", sig.String())
	}

	ctx, cancel := context.WithTimeout(context.Background(), opts.ShutdownTimeout)
	defer cancel()

	if opts.OnShutdown != nil {
		opts.OnShutdown(ctx)
	}

	logger.Info("shutting down server", "timeout", opts.ShutdownTimeout)
	if err := server.Shutdown(ctx); err != nil {
		return err
	}

	logger.Info("server stopped")
	return nil
}
