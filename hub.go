package ws

import (
	"log"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/nucktwillieren/gor-lazy/cache"
)

type Hub struct {
	Name           string
	ExpectedMethod string

	CMutex sync.Mutex

	ConnectionPool map[string]map[string]*Channel //Group, ID, Channel

	Unregister chan *Channel

	Cache cache.Cache
	/*BroadcastChan  chan []byte
	GroupSendChan  chan map[string][]byte
	SendChan       chan map[string]map[string][]byte*/
}

type Context struct {
	Conn *websocket.Conn
	Cha  *Channel

	SecWebSocketProtocol string
	Group                string
	ID                   string
	Message              []byte

	TransportationPayloads []*TransportationPayload
}

func (ctx *Context) AddTranspotationPayload(class string, group string, target string, message []byte) {
	ctx.TransportationPayloads = append(ctx.TransportationPayloads, &TransportationPayload{
		Class:       class,
		TargetGroup: group,
		TargetID:    target,
		Message:     message,
	})
}

func (ctx *Context) AddSingleTargetPayload(group, target string, message []byte) {
	ctx.AddTranspotationPayload("channel", group, target, message)
}

func (ctx *Context) AddGroupTargetPayload(group string, message []byte) {
	ctx.AddTranspotationPayload("group", group, "", message)
}

func (ctx *Context) AddBroadcastPayload(message []byte) {
	ctx.AddTranspotationPayload("broadcast", "", "", message)
}

type TransportationPayload struct {
	Class       string
	TargetGroup string
	TargetID    string
	Message     []byte
}

func Upgrading(w http.ResponseWriter, r *http.Request, ctx *Context) *Context {
	log.Println("Upgrading Protocol")
	conn, err := wsupgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Failed to set websocket upgrade: ", err)
		return ctx
	}
	ctx.Conn = conn
	return ctx
}

func CreateConnection(w http.ResponseWriter, r *http.Request, hub *Hub, transLayer TransportationLayer) *Context {
	secWebSocketProtocol := r.Header.Get("Sec-WebSocket-Protocol")
	ctx := &Context{
		SecWebSocketProtocol: secWebSocketProtocol,
	}

	ctx = Upgrading(w, r, ctx)

	hub.Join(ctx.Conn, ctx, transLayer)

	return ctx
}

func (h *Hub) DoesIDExist(t string, id string) bool {
	for key := range h.ConnectionPool[t] {
		if id == key {
			return true
		}
	}
	return false
}

func (h *Hub) GenUID(group string) string {
	h.CMutex.Lock()
	id := uuid.New().String()
	for h.DoesIDExist(group, id) {
		id = uuid.New().String()
	}
	h.CMutex.Unlock()
	return id
}

func (h *Hub) AddToPool(t string, id string, channel *Channel) {
	if h.ConnectionPool[t] == nil {
		h.ConnectionPool[t] = make(map[string]*Channel)
	}
	h.ConnectionPool[t][id] = channel
}

func (h *Hub) Join(conn *websocket.Conn, ctx *Context, transLayer TransportationLayer) {
	ctx.Cha = NewChannel("", h, "", conn, transLayer)
	ctx.Cha.ID = ctx.ID
	ctx.Cha.GroupName = ctx.Group
	h.AddToPool(ctx.Group, ctx.ID, ctx.Cha)

	if ctx.Cha == nil {
		return
	}

	go ctx.Cha.Reader()
	go ctx.Cha.Writer()
}

func (h *Hub) BroadcastToAll(msg []byte) {
	for k := range h.ConnectionPool {
		for v := range h.ConnectionPool[k] {
			if client := h.ConnectionPool[k][v]; client != nil {
				h.Send(client, msg)
			}
		}
	}
}

func (h *Hub) SendToGroup(groupName string, msg []byte) {
	for k := range h.ConnectionPool[groupName] {
		if client := h.ConnectionPool[groupName][k]; client != nil {
			h.Send(client, msg)
		}
	}
}

func (h *Hub) SendToChannel(groupName string, ID string, msg []byte) {
	if client := h.ConnectionPool[groupName][ID]; client != nil {
		h.Send(client, msg)
	}
}

func (h *Hub) Send(client *Channel, msg []byte) (closed bool) {
	defer func() {
		if recover() != nil {
			closed = true
		}
	}()

	client.SendChan <- msg

	return false
}

func (h *Hub) Start() {
	for channel := range h.Unregister {
		//if _, ok := h.ConnectionPool[channel.GroupName][channel.ID]; ok {
		SafeClose(channel.SendChan)
		delete(h.ConnectionPool[channel.GroupName], channel.ID)
		//}
	}
}

func SafeClose(ch chan []byte) (justClosed bool) {
	defer func() {
		if recover() != nil {
			// The return result can be altered
			// in a defer function call.
			justClosed = false
		}
	}()

	// assume ch != nil here.
	close(ch)   // panic if ch is closed
	return true // <=> justClosed = true; return
}

func NewHub(name string) *Hub {
	log.Println("Create New Hub, name= ", name)
	hub := &Hub{
		Name:           name,
		Unregister:     make(chan *Channel),
		ConnectionPool: make(map[string]map[string]*Channel),
	}
	go hub.Start()
	return hub
}
