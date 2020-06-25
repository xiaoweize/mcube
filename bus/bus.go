package bus

import (
	"github.com/infraboard/mcube/bus/event"
)

// Publisher 发送事件
type Publisher interface {
	// 发送事件
	Pub(topic string, e *event.Event) error
	Connect() error
	Disconnect() error
}

// Subscriber 订阅事件
type Subscriber interface {
	Sub(topic string, h EventHandler) error
	Connect() error
	Disconnect() error
}

// EventHandler is used to process messages via a subscription of a topic.
type EventHandler func(e *event.Event) error