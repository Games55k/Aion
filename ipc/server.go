package ipc

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"
)

const maxMessageBytes = 8 * 1024 * 1024 // 8 MiB per local RPC frame

// Server exposes a Handler on a local Unix domain socket owned by the running GUI process.
type Server struct {
	socketPath string
	ln         net.Listener
	handler    *Handler

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	once   sync.Once
}

// StartServer listens on the Unix domain socket for dataDir.
// Callers must already hold the data directory's .aion.lock (via backend.Open).
func StartServer(dataDir string, handler *Handler) (*Server, error) {
	if handler == nil {
		return nil, errors.New("ipc handler is required")
	}
	sockPath, err := SocketPath(dataDir)
	if err != nil {
		return nil, err
	}
	if err := validateSocketPath(sockPath); err != nil {
		return nil, err
	}
	// Because the caller already owns .aion.lock, any existing socket file for this
	// dataDir is stale from a previous crashed process and safe to remove.
	if _, err := os.Lstat(sockPath); err == nil {
		if err := os.Remove(sockPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("remove stale socket: %w", err)
		}
	}
	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		return nil, fmt.Errorf("listen unix socket: %w", err)
	}
	if err := os.Chmod(sockPath, 0o600); err != nil {
		_ = ln.Close()
		_ = os.Remove(sockPath)
		return nil, fmt.Errorf("chmod unix socket: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	s := &Server{
		socketPath: sockPath,
		ln:         ln,
		handler:    handler,
		ctx:        ctx,
		cancel:     cancel,
	}
	s.wg.Add(1)
	go s.acceptLoop()
	return s, nil
}

// SocketPath returns the active Unix domain socket path.
func (s *Server) SocketPath() string {
	if s == nil {
		return ""
	}
	return s.socketPath
}

func (s *Server) acceptLoop() {
	defer s.wg.Done()
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			if s.ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return
			}
			continue
		}
		s.wg.Add(1)
		go func(c net.Conn) {
			defer s.wg.Done()
			s.serveConn(c)
		}(conn)
	}
}

func (s *Server) serveConn(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReaderSize(conn, 64*1024)
	encoder := json.NewEncoder(conn)
	encoder.SetEscapeHTML(false)

	for {
		_ = conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		line, err := readLineLimited(reader, maxMessageBytes)
		if err != nil {
			if errors.Is(err, io.EOF) || s.ctx.Err() != nil {
				return
			}
			_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			_ = encoder.Encode(rpcResponse{Error: &RPCError{Code: ErrCodeInvalid, Message: err.Error()}})
			return
		}
		if len(line) == 0 {
			continue
		}
		var req rpcRequest
		if err := json.Unmarshal(line, &req); err != nil {
			_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			_ = encoder.Encode(rpcResponse{Error: &RPCError{Code: ErrCodeInvalid, Message: "invalid JSON request"}})
			return
		}
		res, callErr := s.handler.Call(s.ctx, req.Method, req.Params)
		resp := rpcResponse{}
		if callErr != nil {
			resp.Error = encodeError(callErr)
		} else {
			payload, mErr := json.Marshal(res)
			if mErr != nil {
				resp.Error = &RPCError{Code: ErrCodeInternal, Message: mErr.Error()}
			} else {
				resp.Result = payload
			}
		}
		_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := encoder.Encode(resp); err != nil {
			return
		}
	}
}

// Close stops accepting new connections, cancels in-flight handlers, waits for
// active connections to drain, and removes the socket file. Repeated calls are safe.
func (s *Server) Close() error {
	if s == nil {
		return nil
	}
	var closeErr error
	s.once.Do(func() {
		s.cancel()
		if s.ln != nil {
			closeErr = s.ln.Close()
		}
		s.wg.Wait()
		if err := os.Remove(s.socketPath); err != nil && !errors.Is(err, os.ErrNotExist) && closeErr == nil {
			closeErr = err
		}
	})
	return closeErr
}

func readLineLimited(r *bufio.Reader, limit int) ([]byte, error) {
	var out []byte
	for {
		frag, isPrefix, err := r.ReadLine()
		if err != nil {
			if len(out) > 0 && errors.Is(err, io.EOF) {
				return out, nil
			}
			return nil, err
		}
		if len(out)+len(frag) > limit {
			return nil, errors.New("request exceeds maximum size")
		}
		out = append(out, frag...)
		if !isPrefix {
			return out, nil
		}
	}
}
