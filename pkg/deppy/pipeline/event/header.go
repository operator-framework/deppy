package event

import (
	"sync"
	"time"

	"github.com/operator-framework/deppy/pkg/deppy/pipeline"
)

var _ pipeline.EventHeader = &eventHeader{}

type eventHeader struct {
	HeaderEventID          pipeline.EventID       `json:"eventID"`
	HeaderCreator          pipeline.EventSourceID `json:"creatorEventSourceID"`
	HeaderSender           pipeline.EventSourceID `json:"sender"`
	HeaderReceiver         pipeline.EventSourceID `json:"receiver,omitempty"`
	HeaderIsBroadcastEvent bool                   `json:"broadcast"`
	HeaderCreationTime     time.Time              `json:"creationTime"`
	HeaderEventMetadata    pipeline.EventMetadata `json:"metadata,omitempty"`
	lock                   sync.RWMutex
}

func (p *eventHeader) route(receiver pipeline.EventSourceID) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.HeaderIsBroadcastEvent = false
	p.HeaderReceiver = receiver
}

func (p *eventHeader) broadcast() {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.HeaderReceiver = ""
	p.HeaderIsBroadcastEvent = true
}

func (p *eventHeader) copy() *eventHeader {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return &eventHeader{
		HeaderEventID:          p.HeaderEventID,
		HeaderSender:           p.HeaderSender,
		HeaderCreator:          p.HeaderCreator,
		HeaderReceiver:         p.HeaderReceiver,
		HeaderIsBroadcastEvent: p.HeaderIsBroadcastEvent,
		HeaderEventMetadata:    p.HeaderEventMetadata,
		HeaderCreationTime:     p.HeaderCreationTime,
	}
}

func (p *eventHeader) EventID() pipeline.EventID {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.HeaderEventID
}

func (p *eventHeader) Creator() pipeline.EventSourceID {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.HeaderCreator
}

func (p *eventHeader) Sender() pipeline.EventSourceID {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.HeaderSender
}

func (p *eventHeader) Receiver() pipeline.EventSourceID {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.HeaderReceiver
}

func (p *eventHeader) IsBroadcastEvent() bool {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.HeaderIsBroadcastEvent
}

func (p *eventHeader) CreationTime() time.Time {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.HeaderCreationTime
}

func (p *eventHeader) Metadata() pipeline.EventMetadata {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.HeaderEventMetadata
}
