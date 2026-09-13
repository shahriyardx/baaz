package ipc

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"os"
)

// Backend is what the socket server needs from the daemon.
type Backend interface {
	Add(url, filename string, headers map[string]string, format string) (string, error)
	AddBrowser(url, filename string, headers map[string]string, size int64) (string, error)
	Pause(id string) error
	Resume(id string) error
	Cancel(id string) error
	Clear() error
	Delete(id string) error
	Snapshot() *Snapshot
	Subscribe() (<-chan *Snapshot, func())
	SetSettings(kv map[string]string) error
}

type Server struct {
	backend Backend
	ln      net.Listener
}

// NewServer binds the socket. The caller must hold the daemon's single-
// instance lock first — that is what makes removing a stale socket safe.
func NewServer(socketPath string, backend Backend) (*Server, error) {
	os.Remove(socketPath)
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, err
	}
	return &Server{backend: backend, ln: ln}, nil
}

func (s *Server) Close() error { return s.ln.Close() }

func (s *Server) Serve(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		s.ln.Close()
	}()
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		go s.handle(ctx, conn)
	}
}

func (s *Server) handle(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	sc := bufio.NewScanner(conn)
	sc.Buffer(make([]byte, 0, 64<<10), 1<<20)
	enc := json.NewEncoder(conn)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			enc.Encode(Response{OK: false, Error: "bad request: " + err.Error()})
			continue
		}
		if req.Cmd == "watch" {
			s.stream(ctx, conn)
			return
		}
		enc.Encode(s.dispatch(&req))
	}
}

func (s *Server) dispatch(req *Request) Response {
	switch req.Cmd {
	case "ping":
		snap := s.backend.Snapshot()
		return Response{OK: true, Snapshot: snap}
	case "add":
		id, err := s.backend.Add(req.URL, req.Filename, req.Headers, req.Format)
		if err != nil {
			return Response{OK: false, Error: err.Error()}
		}
		return Response{OK: true, ID: id}
	case "add-browser":
		id, err := s.backend.AddBrowser(req.URL, req.Filename, req.Headers, req.Size)
		if err != nil {
			return Response{OK: false, Error: err.Error()}
		}
		return Response{OK: true, ID: id}
	case "setconfig":
		if err := s.backend.SetSettings(req.Settings); err != nil {
			return Response{OK: false, Error: err.Error()}
		}
		return Response{OK: true, Snapshot: s.backend.Snapshot()}
	case "getconfig":
		return Response{OK: true, Snapshot: s.backend.Snapshot()}
	case "pause":
		return errResp(s.backend.Pause(req.ID))
	case "resume":
		return errResp(s.backend.Resume(req.ID))
	case "cancel":
		return errResp(s.backend.Cancel(req.ID))
	case "clear":
		return errResp(s.backend.Clear())
	case "delete":
		return errResp(s.backend.Delete(req.ID))
	case "list", "status":
		return Response{OK: true, Snapshot: s.backend.Snapshot()}
	default:
		return Response{OK: false, Error: "unknown command: " + req.Cmd}
	}
}

// stream turns the connection into a snapshot feed until the peer hangs up.
func (s *Server) stream(ctx context.Context, conn net.Conn) {
	ch, unsub := s.backend.Subscribe()
	defer unsub()

	// Detect peer close: the client never writes again, so any read result
	// (EOF included) means the stream should end.
	closed := make(chan struct{})
	go func() {
		buf := make([]byte, 1)
		for {
			if _, err := conn.Read(buf); err != nil {
				close(closed)
				return
			}
		}
	}()

	enc := json.NewEncoder(conn)
	for {
		select {
		case <-ctx.Done():
			return
		case <-closed:
			return
		case snap := <-ch:
			if err := enc.Encode(snap); err != nil {
				return
			}
		}
	}
}

func errResp(err error) Response {
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	return Response{OK: true}
}
