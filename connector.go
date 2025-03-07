package tubes

import (
	"net/http"
)

type ConnectHookFunc func(*Client)
type DisconnectHookFunc func(*Client)
type MessageHookFunc func(*Client, []byte)
type RequestHandlerFunc func(writer http.ResponseWriter, request *http.Request, properties map[string]interface{}) error

type Connector struct {
	requestHandler RequestHandlerFunc
	errorHandler   ErrorHandlerFunc
	clients        ClientStore
	hooks          *Hooks
}

type Hooks struct {
	OnConnect    ConnectHookFunc
	OnDisconnect DisconnectHookFunc
	OnMessage    MessageHookFunc
}

func NewConnector(requestHandler RequestHandlerFunc, errorHandler ErrorHandlerFunc) *Connector {
	connector := &Connector{
		requestHandler: requestHandler,
		hooks:          &Hooks{},
		errorHandler:   errorHandler,
	}
	connector.clients.init()
	return connector
}

// Join To be triggered if a client connects via ws
func (c *Connector) Join(sendMessage MessageSendFunc, properties map[string]interface{}) *Client {
	client := NewClient(sendMessage, properties)
	c.clients.Join(client)
	if c.hooks.OnConnect != nil {
		c.hooks.OnConnect(client)
	}
	return client
}

func (c *Connector) Message(clientId string, data []byte) {
	client := c.clients.Get(clientId)
	if c.hooks.OnMessage != nil {
		c.hooks.OnMessage(client, data)
	}
}

func (c *Connector) Leave(clientId string) {
	client := c.clients.Get(clientId)
	if c.hooks.OnDisconnect != nil {
		c.hooks.OnDisconnect(client)
	}
	c.clients.Remove(client.Id)
}

func (c *Connector) error(err *Error) {
	if c.errorHandler != nil {
		c.errorHandler(err)
	}
}

func (c *Connector) hook(hooks *Hooks) {
	c.hooks = hooks
}
