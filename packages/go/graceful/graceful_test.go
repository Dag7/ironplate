package graceful

import (
	"net/http"
	"os"
	"syscall"
	"testing"
	"time"
)

func TestListenAndServe_Shutdown(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := &http.Server{
		Addr:    ":0",
		Handler: mux,
	}

	done := make(chan error, 1)
	go func() {
		done <- ListenAndServe(server, &Options{
			ShutdownTimeout: 2 * time.Second,
		})
	}()

	// Give the server time to start.
	time.Sleep(100 * time.Millisecond)

	// Send SIGINT to trigger shutdown.
	p, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Signal(syscall.SIGINT); err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("ListenAndServe returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server did not shut down in time")
	}
}
