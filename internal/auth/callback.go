package auth

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"
)

type CallbackResult struct {
	Code string
}

type CallbackServer struct {
	RedirectURI string
	server      *http.Server
	listener    net.Listener
	results     chan callbackOutcome
}

type callbackOutcome struct {
	result CallbackResult
	err    error
}

func StartCallbackServer(expectedState string) (*CallbackServer, error) {
	return StartCallbackServerAt(expectedState, DefaultRedirectURI)
}

func StartCallbackServerAt(expectedState string, redirectURI string) (*CallbackServer, error) {
	parsed, err := url.Parse(redirectURI)
	if err != nil {
		return nil, fmt.Errorf("parse OAuth redirect URI: %w", err)
	}
	if parsed.Scheme != "http" {
		return nil, fmt.Errorf("OAuth redirect URI must use http for the local callback server")
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("OAuth redirect URI must include a host")
	}
	callbackPath := parsed.EscapedPath()
	if callbackPath == "" {
		callbackPath = "/"
	}

	listener, err := net.Listen("tcp", parsed.Host)
	if err != nil {
		return nil, fmt.Errorf("start OAuth callback server: %w", err)
	}

	callback := &CallbackServer{
		RedirectURI: redirectURI,
		listener:    listener,
		results:     make(chan callbackOutcome, 1),
	}
	if parsed.Port() == "0" {
		callback.RedirectURI = "http://" + listener.Addr().String() + callbackPath
	}

	mux := http.NewServeMux()
	mux.HandleFunc(callbackPath, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if got := query.Get("state"); got != expectedState {
			http.Error(w, "OAuth state did not match. You can close this tab.", http.StatusBadRequest)
			callback.send(callbackOutcome{err: fmt.Errorf("OAuth callback state mismatch")})
			return
		}
		if oauthErr := query.Get("error"); oauthErr != "" {
			http.Error(w, "Roblox returned an OAuth error. You can close this tab.", http.StatusBadRequest)
			callback.send(callbackOutcome{err: fmt.Errorf("OAuth callback error: %s", oauthErr)})
			return
		}
		code := query.Get("code")
		if code == "" {
			http.Error(w, "OAuth callback did not include a code. You can close this tab.", http.StatusBadRequest)
			callback.send(callbackOutcome{err: fmt.Errorf("OAuth callback did not include a code")})
			return
		}

		fmt.Fprint(w, "ERLCX login complete. You can close this tab.")
		callback.send(callbackOutcome{result: CallbackResult{Code: code}})
	})

	callback.server = &http.Server{Handler: mux}
	go func() {
		_ = callback.server.Serve(listener)
	}()

	return callback, nil
}

func (callback *CallbackServer) Wait(ctx context.Context, timeout time.Duration) (CallbackResult, error) {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	select {
	case outcome := <-callback.results:
		_ = callback.Close(context.Background())
		return outcome.result, outcome.err
	case <-ctx.Done():
		_ = callback.Close(context.Background())
		return CallbackResult{}, fmt.Errorf("OAuth callback timed out: %w", ctx.Err())
	}
}

func (callback *CallbackServer) Close(ctx context.Context) error {
	if callback.server == nil {
		return nil
	}
	return callback.server.Shutdown(ctx)
}

func (callback *CallbackServer) send(outcome callbackOutcome) {
	select {
	case callback.results <- outcome:
	default:
	}
}
