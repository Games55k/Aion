package ipc

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"aion/backend"
)

// Router executes local RPC calls transparently against a running Aion GUI
// (via its Unix domain socket) or falls back to opening backend.App for the
// duration of a single call when the GUI is not running.
//
// Because long-lived MCP processes must never hold .aion.lock while idle,
// Router holds zero persistent locks or database handles between calls.
type Router struct {
	dataDir      string
	now          func() time.Time
	retryTimeout time.Duration
}

// RouterOptions configures optional dependencies for Router.
type RouterOptions struct {
	DataDir      string
	Now          func() time.Time
	RetryTimeout time.Duration
}

// NewRouter creates a lock-free IPC/direct router for the given data directory.
func NewRouter(opts RouterOptions) (*Router, error) {
	dir := opts.DataDir
	if dir == "" {
		var err error
		dir, err = backend.DefaultDataDir()
		if err != nil {
			return nil, err
		}
	}
	timeout := opts.RetryTimeout
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	return &Router{
		dataDir:      dir,
		now:          opts.Now,
		retryTimeout: timeout,
	}, nil
}

// DataDir returns the resolved data directory targeted by this router.
func (r *Router) DataDir() string { return r.dataDir }

// Call executes method with JSON params and returns the JSON-encoded result.
func (r *Router) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	var rawParams json.RawMessage
	if params != nil {
		if raw, ok := params.(json.RawMessage); ok {
			rawParams = raw
		} else {
			b, err := json.Marshal(params)
			if err != nil {
				return nil, fmt.Errorf("%w: marshal params: %v", &RPCError{Code: ErrCodeInvalid, Message: err.Error()}, err)
			}
			rawParams = b
		}
	}

	sockPath, err := SocketPath(r.dataDir)
	if err != nil {
		return nil, err
	}

	deadline := time.Now().Add(r.retryTimeout)
	backoff := 20 * time.Millisecond

	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		// 1. Prefer live GUI over Unix domain socket when available.
		res, dialUnavailable, callErr := callViaSocket(ctx, sockPath, method, rawParams)
		if !dialUnavailable {
			return res, callErr
		}

		// 2. Fall back to short-lived direct backend access when GUI is not running.
		app, openErr := backend.Open(ctx, backend.Options{
			DataDir:          r.dataDir,
			DisableScheduler: true,
			Now:              r.now,
		})
		if openErr == nil {
			val, execErr := NewHandler(app, nil).Call(ctx, method, rawParams)
			closeErr := app.Close()
			if execErr != nil {
				return nil, execErr
			}
			if closeErr != nil {
				return nil, closeErr
			}
			b, mErr := json.Marshal(val)
			if mErr != nil {
				return nil, mErr
			}
			return b, nil
		}

		// 3. If .aion.lock is transiently held (e.g. GUI starting up or another
		// concurrent MCP call executing a short write), retry with bounded backoff.
		if !isLockBusyError(openErr) || time.Now().After(deadline) {
			return nil, openErr
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
		}
		if backoff < 150*time.Millisecond {
			backoff *= 2
		}
	}
}

func callViaSocket(ctx context.Context, sockPath, method string, rawParams json.RawMessage) (json.RawMessage, bool, error) {
	if err := validateSocketPath(sockPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, true, nil
		}
		return nil, false, err
	}
	dialer := net.Dialer{Timeout: 500 * time.Millisecond}
	conn, err := dialer.DialContext(ctx, "unix", sockPath)
	if err != nil {
		if isSocketUnavailable(err) {
			return nil, true, nil
		}
		return nil, false, err
	}
	defer conn.Close()

	if dl, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(dl)
	} else {
		_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
	}

	reqBytes, err := json.Marshal(rpcRequest{Method: method, Params: rawParams})
	if err != nil {
		return nil, false, err
	}
	reqBytes = append(reqBytes, '\n')
	if _, err := conn.Write(reqBytes); err != nil {
		return nil, false, err
	}

	line, err := readLineLimited(bufio.NewReaderSize(conn, 64*1024), maxMessageBytes)
	if err != nil {
		return nil, false, err
	}
	var resp rpcResponse
	if err := json.Unmarshal(line, &resp); err != nil {
		return nil, false, fmt.Errorf("decode ipc response: %w", err)
	}
	if resp.Error != nil {
		return nil, false, resp.Error
	}
	return resp.Result, false, nil
}
