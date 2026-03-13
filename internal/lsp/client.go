package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
)

// Client is a JSON-RPC 2.0 client for LSP communication.
type Client struct {
	reader    *bufio.Reader
	rawReader io.Reader
	writer    io.Writer
	writeMu   sync.Mutex         // protects writer
	pendMu    sync.Mutex         // protects pending map and nextID
	nextID    int
	pending   map[int]chan *Response
	onNotify  func(method string, params json.RawMessage)
	done      chan struct{}
}

// NewClient creates a new LSP client using the given reader/writer pair.
func NewClient(r io.Reader, w io.Writer) *Client {
	return &Client{
		reader:    bufio.NewReader(r),
		rawReader: r,
		writer:    w,
		pending:   make(map[int]chan *Response),
		done:      make(chan struct{}),
	}
}

// SetNotificationHandler sets a callback for server notifications.
func (c *Client) SetNotificationHandler(handler func(method string, params json.RawMessage)) {
	c.onNotify = handler
}

// Start begins reading messages from the server. Must be called in a goroutine.
func (c *Client) Start() {
	defer close(c.done)
	for {
		msg, err := c.readMessage()
		if err != nil {
			return
		}
		c.handleMessage(msg)
	}
}

// Call sends a request and waits for a response.
func (c *Client) Call(method string, params interface{}) (*Response, error) {
	c.pendMu.Lock()
	id := c.nextID
	c.nextID++
	ch := make(chan *Response, 1)
	c.pending[id] = ch
	c.pendMu.Unlock()

	req := Request{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	if err := c.writeMessage(req); err != nil {
		c.pendMu.Lock()
		delete(c.pending, id)
		c.pendMu.Unlock()
		return nil, err
	}

	select {
	case resp := <-ch:
		return resp, nil
	case <-c.done:
		return nil, fmt.Errorf("client closed")
	}
}

// Notify sends a notification (no response expected).
func (c *Client) Notify(method string, params interface{}) error {
	notif := Notification{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}
	return c.writeMessage(notif)
}

// Close signals the client to stop and cleans up resources.
func (c *Client) Close() {
	// Close the underlying reader to unblock Start()'s read loop.
	if rc, ok := c.rawReader.(io.Closer); ok {
		rc.Close()
	}
	if wc, ok := c.writer.(io.Closer); ok {
		wc.Close()
	}
	// Wait for Start() to finish (it closes done when it returns).
	<-c.done
}

func (c *Client) writeMessage(msg interface{}) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))
	if _, err := io.WriteString(c.writer, header); err != nil {
		return err
	}
	_, err = c.writer.Write(data)
	return err
}

func (c *Client) readMessage() (json.RawMessage, error) {
	contentLength := 0
	for {
		line, err := c.reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		if after, ok := strings.CutPrefix(line, "Content-Length:"); ok {
			contentLength, _ = strconv.Atoi(strings.TrimSpace(after))
		}
	}

	if contentLength == 0 {
		return nil, fmt.Errorf("missing Content-Length header")
	}

	body := make([]byte, contentLength)
	if _, err := io.ReadFull(c.reader, body); err != nil {
		return nil, err
	}

	return json.RawMessage(body), nil
}

func (c *Client) handleMessage(raw json.RawMessage) {
	var probe struct {
		ID     *int            `json:"id"`
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return
	}

	if probe.ID != nil && probe.Method == "" {
		// It's a response
		var resp Response
		if err := json.Unmarshal(raw, &resp); err != nil {
			return
		}
		c.pendMu.Lock()
		ch, ok := c.pending[resp.ID]
		if ok {
			delete(c.pending, resp.ID)
		}
		c.pendMu.Unlock()
		if ok {
			ch <- &resp
		}
	} else if probe.Method != "" && probe.ID == nil {
		// It's a notification
		if c.onNotify != nil {
			c.onNotify(probe.Method, probe.Params)
		}
	}
}
