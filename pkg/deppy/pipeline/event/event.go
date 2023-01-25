package event

import (
	"encoding/json"
	"sync"

	"github.com/operator-framework/deppy/pkg/deppy/pipeline"
)

var _ pipeline.Event = &event{}

type event struct {
	EventHeader *eventHeader `json:"header"`
	lock        sync.RWMutex
}

func newEvent(header *eventHeader) *event {
	return &event{
		EventHeader: header,
		lock:        sync.RWMutex{},
	}
}

func (e *event) Header() pipeline.EventHeader {
	return e.EventHeader
}

func (e *event) Broadcast() {
	e.lock.Lock()
	defer e.lock.Unlock()
	e.EventHeader.broadcast()
}

func (e *event) Route(dest pipeline.EventSourceID) {
	e.lock.Lock()
	defer e.lock.Unlock()
	e.EventHeader.route(dest)
}

func (e *event) String() string {
	e.lock.Lock()
	defer e.lock.Unlock()
	bytes, err := json.Marshal(e)
	if err != nil {
		return string(e.EventHeader.EventID())
	}
	return string(bytes)
}

var _ pipeline.DataEvent[interface{}] = &dataEvent[interface{}]{}

type dataEvent[D interface{}] struct {
	*event
	EventData D `json:"data"`
}

func (p *dataEvent[I]) Header() pipeline.EventHeader {
	return p.EventHeader
}

func (p *dataEvent[I]) Data() I {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.EventData
}

func (p *dataEvent[I]) String() string {
	p.lock.Lock()
	defer p.lock.Unlock()
	bytes, err := json.Marshal(p)
	if err != nil {
		return string(p.EventHeader.EventID())
	}
	return string(bytes)
}

func (p *dataEvent[I]) Copy() pipeline.DataEvent[I] {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return &dataEvent[I]{
		event: &event{
			EventHeader: p.EventHeader.copy(),
			lock:        sync.RWMutex{},
		},
		EventData: p.EventData,
	}
}

var _ pipeline.ErrorEvent = &errorEvent{}

type errorEvent struct {
	*event
	EventError error `json:"error"`
}

func (e *errorEvent) Header() pipeline.EventHeader {
	e.lock.RLock()
	defer e.lock.RUnlock()
	return e.EventHeader
}

func (e *errorEvent) Error() error {
	e.lock.RLock()
	defer e.lock.RUnlock()
	return e.EventError
}

func (e *errorEvent) String() string {
	e.lock.Lock()
	defer e.lock.Unlock()
	bytes, err := json.Marshal(e)
	if err != nil {
		return string(e.EventHeader.EventID())
	}
	return string(bytes)
}

func (e *errorEvent) Copy() pipeline.ErrorEvent {
	e.lock.RLock()
	defer e.lock.RUnlock()
	return &errorEvent{
		event: &event{
			EventHeader: e.EventHeader.copy(),
			lock:        sync.RWMutex{},
		},
		EventError: e.EventError,
	}
}
