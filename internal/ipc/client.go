package ipc

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

type Client struct {
	conn net.Conn
	rd   *bufio.Reader
}

// Dial connects to the daemon socket. With autoStart, a dead socket causes a
// detached `dm daemon` spawn and a short connect retry loop — identical
// behavior for the CLI, the native-messaging host, and the bar widget.
func Dial(socketPath, logPath string, autoStart bool) (*Client, error) {
	conn, err := net.DialTimeout("unix", socketPath, time.Second)
	if err == nil {
		return &Client{conn: conn, rd: bufio.NewReader(conn)}, nil
	}
	if !autoStart {
		return nil, err
	}
	os.Remove(socketPath) // stale socket from a dead daemon
	if err := spawnDaemon(logPath); err != nil {
		return nil, fmt.Errorf("start daemon: %w", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
		conn, err = net.DialTimeout("unix", socketPath, time.Second)
		if err == nil {
			return &Client{conn: conn, rd: bufio.NewReader(conn)}, nil
		}
	}
	return nil, fmt.Errorf("daemon did not come up: %w", err)
}

func spawnDaemon(logPath string) error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	os.MkdirAll(filepath.Dir(logPath), 0o755)
	logf, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer logf.Close()
	cmd := exec.Command(self, "daemon")
	cmd.Stdout = logf
	cmd.Stderr = logf
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

func (c *Client) Close() error { return c.conn.Close() }

func (c *Client) Do(req *Request) (*Response, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	if _, err := c.conn.Write(append(data, '\n')); err != nil {
		return nil, err
	}
	line, err := c.rd.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	var resp Response
	if err := json.Unmarshal(line, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Watch subscribes and invokes fn with each raw snapshot line (newline
// stripped) until the connection drops or fn returns false.
func (c *Client) Watch(fn func(line []byte) bool) error {
	if _, err := c.conn.Write([]byte(`{"cmd":"watch"}` + "\n")); err != nil {
		return err
	}
	for {
		line, err := c.rd.ReadBytes('\n')
		if err != nil {
			return err
		}
		if len(line) > 0 && !fn(line[:len(line)-1]) {
			return nil
		}
	}
}
