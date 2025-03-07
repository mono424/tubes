package tubes

import (
	"encoding/json"
	"net/http"
)

const (
	MessageTypeSubscribe      = "subscribe"
	MessageTypeUnsubscribe    = "unsubscribe"
	MessageTypeChannelMessage = "message"
)

type Message struct {
	Type    string          `json:"type"`
	Channel string          `json:"channel"`
	Payload json.RawMessage `json:"payload"`
}

type TubeSystem struct {
	connector *Connector
	channels  ChannelStore
}

// New Creates a new TubeSystem instance
func New(connector *Connector) *TubeSystem {
	r := TubeSystem{}

	r.connector = connector
	r.channels.init(connector.error)
	r.connector.hook(&Hooks{
		OnConnect:    r.connectHandler,
		OnDisconnect: r.disconnectHandler,
		OnMessage:    r.messageHandler,
	})

	return &r
}

// RegisterChannel registers a new channel
func (r *TubeSystem) RegisterChannel(channelName string, handlers ChannelHandlers) *Channel {
	return r.channels.Register(channelName, handlers)
}

// HandleRequest handles a new websocket request, adds the properties to the new client
func (r *TubeSystem) HandleRequest(writer http.ResponseWriter, request *http.Request, properties map[string]interface{}) error {
	return r.connector.requestHandler(writer, request, properties)
}

func (r *TubeSystem) IsConnected(clientId string) bool {
	return r.connector.clients.Exists(clientId)
}

// IsSubscribed checks whether a client is subscribed to all channels associated with a certain channelPath or not
func (r *TubeSystem) IsSubscribed(channelPath string, clientId string) bool {
	if matches := r.channels.Get(channelPath); len(matches) > 0 {
		for _, match := range matches {
			if !match.Channel.IsSubscribed(clientId, channelPath) {
				return false
			}
		}
		return true
	}
	return false
}

// GetChannel returns a registered channel for an exact channel path.
func (r *TubeSystem) GetChannel(channelPath string) (bool, *Channel) {
	return r.channels.GetByExactPath(channelPath)
}

func (r *TubeSystem) Send(channelPath string, clientId string, payload []byte) *Error {
	matches := r.channels.Get(channelPath)
	if len(matches) == 0 {
		return NewError(nil, ErrorUnknownChannel, "channel does not exist", nil)
	}
	if !r.IsSubscribed(channelPath, clientId) {
		return NewError(nil, ErrorClientNotSubscribed, "user not subscribed to channel", nil)
	}

	var errors []*Error
	for _, match := range matches {
		context, userSubscribed := match.Channel.FindContext(clientId, channelPath)
		if !userSubscribed {
			errors = append(errors, NewError(nil, ErrorClientNotSubscribed, "user not subscribed to channel", nil))
		}
		if err := context.Send(payload); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return NewMultiError(nil, "Failed to send at least over one channel", nil, errors)
	}
	return nil
}

// connectHandler handles a new melody connection
func (r *TubeSystem) connectHandler(client *Client) {}

// disconnectHandler handles a client disconnect
func (r *TubeSystem) disconnectHandler(c *Client) {
	r.channels.UnsubscribeAll(c.Id)
}

// messageHandler handles a new client message
func (r *TubeSystem) messageHandler(c *Client, msg []byte) {
	var req Message
	err := json.Unmarshal(msg, &req)
	if err != nil {
		r.connector.error(NewError(nil, ErrorInvalidMessage, "invalid message received", err))
		return
	}

	switch req.Type {
	case MessageTypeSubscribe:
		r.channels.Subscribe(c, req.Channel)
	case MessageTypeUnsubscribe:
		r.channels.Unsubscribe(c.Id, req.Channel)
	case MessageTypeChannelMessage:
		r.channels.OnMessage(c, &req)
	default:
		r.connector.error(NewError(nil, ErrorUnknownType, "unknown tubeSystem request type: '"+req.Type+"'", nil))
	}
}
