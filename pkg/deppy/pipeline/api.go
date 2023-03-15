package pipeline

import (
	"time"
)

type EventSourceID string

type EventType string
type EventID string
type EventMetadata map[string]string

type EventHeader interface {
	EventID() EventID
	Creator() EventSourceID
	Sender() EventSourceID
	Receiver() EventSourceID
	IsBroadcastEvent() bool
	CreationTime() time.Time
	Metadata() EventMetadata
}

type EventSource interface {
	EventSourceID() EventSourceID
	IngressCapacity() int
}

type EventIDProvider interface {
	NextEventID() EventID
}

type Event interface {
	Header() EventHeader
	Broadcast()
	Route(dest EventSourceID)
	String() string
}

type DataEvent[D interface{}] interface {
	Event
	Data() D
	Copy() DataEvent[D]
}

type ErrorEvent interface {
	Event
	Error() error
	Copy() ErrorEvent
}

type EventFactory[I interface{}] interface {
	NewErrorEvent(err error) ErrorEvent
	NewDataEvent(data I) DataEvent[I]
}
