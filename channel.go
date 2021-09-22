package ws

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

var wsupgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	Subprotocols:    []string{},
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type TransportationLayer func(ctx *Context) (*Context, error)

type Channel struct {
	Hub       *Hub
	GroupName string
	ID        string
	Conn      *websocket.Conn
	SendChan  chan []byte

	TransportationLayer TransportationLayer
}

func (c *Channel) Reader() {
	defer func() {
		log.Printf("Channel Reader Close in hub <%s>", c.Hub.Name)
		c.Hub.Unregister <- c
		if c.Conn != nil {
			c.Conn.Close()
		}
	}()
	log.Printf("Channel Reader Start in hub <%s>", c.Hub.Name)
	for {
		if c.Conn == nil {
			return
		}
		_, msg, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(
				err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure,
			) {
				log.Printf("error: %v", err)
			}
			return
		}
		//log.Println(
		//	"Websocket Package From: ",
		//	c.Conn.LocalAddr().String(),
		//	c.Conn.LocalAddr().Network(),
		//	c.Conn.RemoteAddr().String(),
		//	c.Conn.RemoteAddr().Network(),
		//)
		//log.Println("Source Bytes(Raw): ", msg)
		//log.Println("Source Bytes(Hex): ", hex.EncodeToString(msg))
		//log.Println("Source Bytes(String): ", string(msg))

		ctx := Context{
			ID:      c.ID,
			Cha:     c,
			Group:   c.GroupName,
			Message: msg,
		}
		c.Transport(&ctx)
	}
}

func (c *Channel) Transport(ctx *Context) *Context {
	if c.TransportationLayer == nil {
		return ctx
	}

	ctx, err := c.TransportationLayer(ctx)
	if err != nil {
		log.Println("Transportation Error: ", err.Error())
		return ctx
	}

	for i := range ctx.TransportationPayloads {
		transportation := ctx.TransportationPayloads[i]
		switch transportation.Class {
		case "broadcast":
			c.Hub.BroadcastToAll(transportation.Message)
		case "group":
			c.Hub.SendToGroup(transportation.TargetGroup, transportation.Message)
		case "channel":
			c.Hub.SendToChannel(transportation.TargetGroup, transportation.TargetID, transportation.Message)
		}
		//log.Println(c.Hub.ConnectionPool)
		log.Println("Data Transportation(", transportation.Class, "): ", ctx.ID)
	}

	return ctx
}

func (c *Channel) Writer() {
	defer func() {
		log.Printf("Channel Writer Close in hub <%s>", c.Hub.Name)
		c.Hub.Unregister <- c
		if c.Conn != nil {
			c.Conn.Close()
		}
		SafeClose(c.SendChan)
	}()
	log.Printf("Channel Writer Start in hub <%s>", c.Hub.Name)
	for message := range c.SendChan {
		if c.Conn == nil {
			return
		}
		w, err := c.Conn.NextWriter(websocket.TextMessage)
		if err != nil {
			return
		}

		w.Write(message)

		// Add queued chat messages to the current websocket message.
		n := len(c.SendChan)
		for i := 0; i < n; i++ {
			w.Write(newline)
			w.Write(<-c.SendChan)
		}
		log.Println("Writer Send(String): ", string(message))

		if err := w.Close(); err != nil {
			return
		}
	}
}

func NewChannel(id string, hub *Hub, groupName string, conn *websocket.Conn, transLayer TransportationLayer) *Channel {
	tun := Channel{
		Hub:                 hub,
		ID:                  id,
		GroupName:           groupName,
		Conn:                conn,
		SendChan:            make(chan []byte),
		TransportationLayer: transLayer,
	}
	return &tun
}
