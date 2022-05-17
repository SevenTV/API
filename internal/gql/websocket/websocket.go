package websocket

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/errcode"
	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

type (
	Websocket struct {
		Upgrader              websocket.FastHTTPUpgrader
		InitFunc              WebsocketInitFunc
		KeepAlivePingInterval time.Duration
		PingPongInterval      time.Duration

		didInjectSubprotocols bool
	}
	wsConnection struct {
		Websocket
		ctx             context.Context
		conn            *websocket.Conn
		me              messageExchanger
		active          map[string]context.CancelFunc
		mu              sync.Mutex
		keepAliveTicker *time.Ticker
		pingPongTicker  *time.Ticker
		exec            graphql.GraphExecutor

		initPayload InitPayload
	}

	WebsocketInitFunc func(ctx context.Context, initPayload InitPayload) (context.Context, error)
)

func jsonDecode(r io.Reader, val interface{}) error {
	dec := json.NewDecoder(r)
	dec.UseNumber()
	return dec.Decode(val)
}

// SendError sends a best effort error to a raw response writer. It assumes the client can understand the standard
// json error response
func SendError(ctx *fasthttp.RequestCtx, code int, errors ...*gqlerror.Error) {
	ctx.SetStatusCode(code)
	b, err := json.Marshal(&graphql.Response{Errors: errors})
	if err != nil {
		panic(err)
	}
	ctx.SetBody(b)
}

// SendErrorf wraps SendError to add formatted messages
func SendErrorf(ctx *fasthttp.RequestCtx, code int, format string, args ...interface{}) {
	SendError(ctx, code, &gqlerror.Error{Message: fmt.Sprintf(format, args...)})
}

func (t Websocket) Supports(ctx *fasthttp.RequestCtx) bool {
	return websocket.FastHTTPIsWebSocketUpgrade(ctx)
}

func (t Websocket) Do(r *fasthttp.RequestCtx, ctx context.Context, exec graphql.GraphExecutor) {
	t.injectGraphQLWSSubprotocols()
	err := t.Upgrader.Upgrade(r, func(ws *websocket.Conn) {
		var me messageExchanger
		switch ws.Subprotocol() {
		default:
			msg := websocket.FormatCloseMessage(websocket.CloseProtocolError, fmt.Sprintf("unsupported negotiated subprotocol %s", ws.Subprotocol()))
			_ = ws.WriteMessage(websocket.CloseMessage, msg)
			return
		case graphqlwsSubprotocol, "":
			// clients are required to send a subprotocol, to be backward compatible with the previous implementation we select
			// "graphql-ws" by default
			me = graphqlwsMessageExchanger{c: ws}
		case graphqltransportwsSubprotocol:
			me = graphqltransportwsMessageExchanger{c: ws}
		}

		conn := wsConnection{
			active:    map[string]context.CancelFunc{},
			conn:      ws,
			ctx:       ctx,
			exec:      exec,
			me:        me,
			Websocket: t,
		}

		if !conn.init() {
			return
		}

		conn.run()
	})
	if err != nil {
		log.Printf("unable to upgrade %T to websocket %s: ", ctx, err.Error())
		SendErrorf(r, http.StatusBadRequest, "unable to upgrade")
		return
	}
}

func (c *wsConnection) init() bool {
	m, err := c.me.NextMessage()
	if err != nil {
		if err == errInvalidMsg {
			c.sendConnectionError("invalid json")
		}

		c.close(websocket.CloseProtocolError, "decoding error")
		return false
	}

	switch m.t {
	case initMessageType:
		if len(m.payload) > 0 {
			c.initPayload = make(InitPayload)
			err := json.Unmarshal(m.payload, &c.initPayload)
			if err != nil {
				return false
			}
		}

		if c.InitFunc != nil {
			ctx, err := c.InitFunc(c.ctx, c.initPayload)
			if err != nil {
				c.sendConnectionError(err.Error())
				c.close(websocket.CloseNormalClosure, "terminated")
				return false
			}
			c.ctx = ctx
		}

		c.write(&message{t: connectionAckMessageType})
		c.write(&message{t: keepAliveMessageType})
	case connectionCloseMessageType:
		c.close(websocket.CloseNormalClosure, "terminated")
		return false
	default:
		c.sendConnectionError("unexpected message %s", m.t)
		c.close(websocket.CloseProtocolError, "unexpected message")
		return false
	}

	return true
}

func (c *wsConnection) write(msg *message) {
	c.mu.Lock()
	// TODO: missing error handling here, err from previous implementation
	// was ignored
	_ = c.me.Send(msg)
	c.mu.Unlock()
}

func (c *wsConnection) run() {
	// We create a cancellation that will shutdown the keep-alive when we leave
	// this function.
	ctx, cancel := context.WithCancel(c.ctx)
	defer func() {
		cancel()
		c.close(websocket.CloseAbnormalClosure, "unexpected closure")
	}()

	// Create a timer that will fire every interval to keep the connection alive.
	if c.KeepAlivePingInterval != 0 {
		c.mu.Lock()
		c.keepAliveTicker = time.NewTicker(c.KeepAlivePingInterval)
		c.mu.Unlock()

		go c.keepAlive(ctx)
	}

	// Create a timer that will fire every interval a ping message that should
	// receive a pong (SetPongHandler in init() function)
	if c.PingPongInterval != 0 {
		c.mu.Lock()
		c.pingPongTicker = time.NewTicker(c.PingPongInterval)
		c.mu.Unlock()

		_ = c.conn.SetReadDeadline(time.Now().UTC().Add(2 * c.PingPongInterval))
		go c.ping(ctx)
	}

	// Close the connection when the context is cancelled.
	// Will optionally send a "close reason" that is retrieved from the context.
	go c.closeOnCancel(ctx)

	for {
		start := graphql.Now()
		m, err := c.me.NextMessage()
		if err != nil {
			// TODO: better error handling here
			return
		}

		switch m.t {
		case startMessageType:
			c.subscribe(start, &m)
		case stopMessageType:
			c.mu.Lock()
			closer := c.active[m.id]
			c.mu.Unlock()
			if closer != nil {
				closer()
			}
		case connectionCloseMessageType:
			c.close(websocket.CloseNormalClosure, "terminated")
			return
		case pingMesageType:
			c.write(&message{t: pongMessageType, payload: m.payload})
		case pongMessageType:
			_ = c.conn.SetReadDeadline(time.Now().UTC().Add(2 * c.PingPongInterval))
		default:
			c.sendConnectionError("unexpected message %s", m.t)
			c.close(websocket.CloseProtocolError, "unexpected message")
			return
		}
	}
}

func (c *wsConnection) keepAlive(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			c.keepAliveTicker.Stop()
			return
		case <-c.keepAliveTicker.C:
			c.write(&message{t: keepAliveMessageType})
		}
	}
}

func (c *wsConnection) ping(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			c.pingPongTicker.Stop()
			return
		case <-c.pingPongTicker.C:
			c.write(&message{t: pingMesageType, payload: json.RawMessage{}})
		}
	}
}

func (c *wsConnection) closeOnCancel(ctx context.Context) {
	<-ctx.Done()

	if r := closeReasonForContext(ctx); r != "" {
		c.sendConnectionError(r)
	}
	c.close(websocket.CloseNormalClosure, "terminated")
}

func (c *wsConnection) subscribe(start time.Time, msg *message) {
	ctx := graphql.StartOperationTrace(c.ctx)
	var params *graphql.RawParams
	if err := jsonDecode(bytes.NewReader(msg.payload), &params); err != nil {
		c.sendError(msg.id, &gqlerror.Error{Message: "invalid json"})
		c.complete(msg.id)
		return
	}

	params.ReadTime = graphql.TraceTiming{
		Start: start,
		End:   graphql.Now(),
	}

	rc, err := c.exec.CreateOperationContext(ctx, params)
	if err != nil {
		resp := c.exec.DispatchError(graphql.WithOperationContext(ctx, rc), err)
		switch errcode.GetErrorKind(err) {
		case errcode.KindProtocol:
			c.sendError(msg.id, resp.Errors...)
		default:
			c.sendResponse(msg.id, &graphql.Response{Errors: err})
		}

		c.complete(msg.id)
		return
	}

	ctx = graphql.WithOperationContext(ctx, rc)

	if c.initPayload != nil {
		ctx = withInitPayload(ctx, c.initPayload)
	}

	ctx, cancel := context.WithCancel(ctx)
	c.mu.Lock()
	c.active[msg.id] = cancel
	c.mu.Unlock()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				err := rc.Recover(ctx, r)
				var gqlerr *gqlerror.Error
				if !errors.As(err, &gqlerr) {
					gqlerr = &gqlerror.Error{}
					if err != nil {
						gqlerr.Message = err.Error()
					}
				}
				c.sendError(msg.id, gqlerr)
			}
			c.complete(msg.id)
			c.mu.Lock()
			delete(c.active, msg.id)
			c.mu.Unlock()
			cancel()
		}()

		responses, ctx := c.exec.DispatchOperation(ctx, rc)
		for {
			response := responses(ctx)
			if response == nil {
				break
			}

			c.sendResponse(msg.id, response)
		}
		c.complete(msg.id)

		c.mu.Lock()
		delete(c.active, msg.id)
		c.mu.Unlock()
		cancel()
	}()
}

func (c *wsConnection) sendResponse(id string, response *graphql.Response) {
	b, err := json.Marshal(response)
	if err != nil {
		panic(err)
	}
	c.write(&message{
		payload: b,
		id:      id,
		t:       dataMessageType,
	})
}

func (c *wsConnection) complete(id string) {
	c.write(&message{id: id, t: completeMessageType})
}

func (c *wsConnection) sendError(id string, errors ...*gqlerror.Error) {
	errs := make([]error, len(errors))
	for i, err := range errors {
		errs[i] = err
	}
	b, err := json.Marshal(errs)
	if err != nil {
		panic(err)
	}
	c.write(&message{t: errorMessageType, id: id, payload: b})
}

func (c *wsConnection) sendConnectionError(format string, args ...interface{}) {
	b, err := json.Marshal(&gqlerror.Error{Message: fmt.Sprintf(format, args...)})
	if err != nil {
		panic(err)
	}

	c.write(&message{t: connectionErrorMessageType, payload: b})
}

func (c *wsConnection) close(closeCode int, message string) {
	c.mu.Lock()
	_ = c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(closeCode, message))
	for _, closer := range c.active {
		closer()
	}
	c.mu.Unlock()
	_ = c.conn.Close()
}
