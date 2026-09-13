// Package nmhost implements the Chrome native-messaging host: length-prefixed
// JSON on stdio, forwarded to the daemon over the unix socket.
package nmhost

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"dm/internal/config"
	"dm/internal/ipc"
)

const maxOutbound = 1 << 20 // Chrome rejects host->browser messages over 1 MB

type inMessage struct {
	Type      string `json:"type"` // add|ping|cancel
	URL       string `json:"url,omitempty"`
	Filename  string `json:"filename,omitempty"`
	Cookies   string `json:"cookies,omitempty"`
	Referrer  string `json:"referrer,omitempty"`
	UserAgent string `json:"userAgent,omitempty"`
	ID        string `json:"id,omitempty"`
}

type outMessage struct {
	OK     bool   `json:"ok"`
	Error  string `json:"error,omitempty"`
	ID     string `json:"id,omitempty"`
	Active int    `json:"active,omitempty"`
}

// Run services one Chrome connection until stdin EOF.
func Run() error {
	for {
		msg, err := read(os.Stdin)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		write(os.Stdout, handle(msg))
	}
}

func handle(msg *inMessage) outMessage {
	client, err := ipc.Dial(config.SocketPath(), config.LogPath(), true)
	if err != nil {
		return outMessage{OK: false, Error: "daemon unreachable: " + err.Error()}
	}
	defer client.Close()

	var req *ipc.Request
	switch msg.Type {
	case "add":
		headers := map[string]string{}
		if msg.Cookies != "" {
			headers["Cookie"] = msg.Cookies
		}
		if msg.Referrer != "" {
			headers["Referer"] = msg.Referrer
		}
		if msg.UserAgent != "" {
			headers["User-Agent"] = msg.UserAgent
		}
		req = &ipc.Request{Cmd: "add", URL: msg.URL, Filename: msg.Filename, Headers: headers}
	case "ping":
		req = &ipc.Request{Cmd: "ping"}
	case "cancel":
		req = &ipc.Request{Cmd: "cancel", ID: msg.ID}
	default:
		return outMessage{OK: false, Error: "unknown message type: " + msg.Type}
	}

	resp, err := client.Do(req)
	if err != nil {
		return outMessage{OK: false, Error: err.Error()}
	}
	out := outMessage{OK: resp.OK, Error: resp.Error, ID: resp.ID}
	if resp.Snapshot != nil {
		out.Active = resp.Snapshot.Active
	}
	return out
}

// read decodes one native-messaging frame: 4-byte native-endian length + JSON.
func read(r io.Reader) (*inMessage, error) {
	var length uint32
	if err := binary.Read(r, binary.NativeEndian, &length); err != nil {
		return nil, err
	}
	if length == 0 || length > 64<<20 {
		return nil, fmt.Errorf("bad frame length %d", length)
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	var msg inMessage
	if err := json.Unmarshal(buf, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

func write(w io.Writer, msg outMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	if len(data) > maxOutbound {
		data, _ = json.Marshal(outMessage{OK: false, Error: "reply too large"})
	}
	if err := binary.Write(w, binary.NativeEndian, uint32(len(data))); err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}
