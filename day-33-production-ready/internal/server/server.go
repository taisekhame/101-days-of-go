// Package server starts an HTTP server and shuts it down cleanly.
package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

// Run serves until the context is cancelled (Ctrl+C, or SIGTERM from a host),
// then lets requests that are already running finish, for up to the grace period.
func Run(ctx context.Context, srv *http.Server, grace time.Duration) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		// The server stopped on its own, for example because the port is taken
		return err
	case <-ctx.Done():
	}

	slog.Info("shutting down, waiting for running requests to finish", "grace", grace)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), grace)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		_ = srv.Close() // out of patience: cut off whatever is still running
		return err
	}
	<-errCh // ListenAndServe returns http.ErrServerClosed once Shutdown has started
	slog.Info("server stopped cleanly")
	return nil
}
