package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
)

const callbackPath = "/callback"
const defaultCallbackAddress = "127.0.0.1:8080"

// CallbackResult captures the OAuth callback query parameters.
type CallbackResult struct {
	Code             string
	State            string
	Error            string
	ErrorDescription string
}

// CallbackWaiter waits for a single OAuth callback result.
type CallbackWaiter interface {
	URL() string
	Wait(context.Context) (CallbackResult, error)
}

type waiterFunc struct {
	url  string
	wait func(context.Context) (CallbackResult, error)
}

func (f waiterFunc) URL() string {
	return f.url
}

func (f waiterFunc) Wait(ctx context.Context) (CallbackResult, error) {
	return f.wait(ctx)
}

// CallbackServer listens on localhost for a single OAuth redirect callback.
type CallbackServer struct {
	url      string
	server   *http.Server
	listener net.Listener
	resultCh chan CallbackResult
	errCh    chan error
	once     sync.Once
}

// NewCallbackServer starts a localhost callback server on the provided address.
func NewCallbackServer(address string) (*CallbackServer, error) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}

	srv := &CallbackServer{
		url:      fmt.Sprintf("http://%s%s", listener.Addr().String(), callbackPath),
		listener: listener,
		resultCh: make(chan CallbackResult, 1),
		errCh:    make(chan error, 1),
	}

	mux := http.NewServeMux()
	mux.HandleFunc(callbackPath, srv.handleCallback)
	srv.server = &http.Server{Handler: mux}

	go func() {
		err := srv.server.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			select {
			case srv.errCh <- err:
			default:
			}
		}
	}()

	return srv, nil
}

// NewEphemeralCallbackServer starts a localhost callback server on an ephemeral port.
func NewEphemeralCallbackServer() (*CallbackServer, error) {
	return NewCallbackServer("127.0.0.1:0")
}

// StartCallbackServer starts a localhost callback server on the default runtime callback address.
func StartCallbackServer() (CallbackWaiter, error) {
	return NewCallbackServer(defaultCallbackAddress)
}

// URL returns the callback endpoint for the running server.
func (s *CallbackServer) URL() string {
	return s.url
}

// Wait blocks until a callback request arrives or ctx is canceled.
func (s *CallbackServer) Wait(ctx context.Context) (CallbackResult, error) {
	select {
	case result := <-s.resultCh:
		s.shutdown()
		return result, nil
	case err := <-s.errCh:
		s.shutdown()
		return CallbackResult{}, err
	case <-ctx.Done():
		s.shutdown()
		return CallbackResult{}, ctx.Err()
	}
}

func (s *CallbackServer) handleCallback(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != callbackPath {
		http.NotFound(w, r)
		return
	}

	result := CallbackResult{
		Code:             r.URL.Query().Get("code"),
		State:            r.URL.Query().Get("state"),
		Error:            r.URL.Query().Get("error"),
		ErrorDescription: r.URL.Query().Get("error_description"),
	}

	select {
	case s.resultCh <- result:
	default:
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("Authorization received. You can close this tab."))
}

func (s *CallbackServer) shutdown() {
	s.once.Do(func() {
		_ = s.server.Shutdown(context.Background())
		_ = s.listener.Close()
	})
}
