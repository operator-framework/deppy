package event

import (
	"sync"
	"time"

	"github.com/operator-framework/deppy/pkg/deppy/pipeline"
	"github.com/operator-framework/deppy/pkg/deppy/pipeline/event/eventidprovider"
)

var _ pipeline.EventFactory[interface{}] = &eventFactory[interface{}]{}

type FactoryOption[I interface{}] func(factory *eventFactory[I])

func WithEventIDProvider[I interface{}](eventIDProvider pipeline.EventIDProvider) FactoryOption[I] {
	return func(eventFactory *eventFactory[I]) {
		eventFactory.eventIDProvider = eventIDProvider
	}
}

func WithEventMetadata[I interface{}](eventMetadata pipeline.EventMetadata) FactoryOption[I] {
	return func(eventFactory *eventFactory[I]) {
		eventFactory.eventMetadata = eventMetadata
	}
}

type eventFactory[I interface{}] struct {
	eventSourceID   pipeline.EventSourceID
	eventMetadata   pipeline.EventMetadata
	eventIDProvider pipeline.EventIDProvider
}

func NewEventFactory[I interface{}](eventSourceID pipeline.EventSourceID, options ...FactoryOption[I]) pipeline.EventFactory[I] {
	factory := &eventFactory[I]{
		eventSourceID: eventSourceID,
		// eventIDProvider: eventidprovider.NewUUIDEventIDProvider(),
		eventIDProvider: eventidprovider.MonotonicallyIncreasingEventIDProvider(),
	}

	for _, applyOption := range options {
		applyOption(factory)
	}

	return factory
}

func (p *eventFactory[I]) NewErrorEvent(err error) pipeline.ErrorEvent {
	return &errorEvent{
		event:      newEvent(p.newEventHeader()),
		EventError: err,
	}
}

func (p *eventFactory[I]) NewDataEvent(data I) pipeline.DataEvent[I] {
	return &dataEvent[I]{
		event:     newEvent(p.newEventHeader()),
		EventData: data,
	}
}

func (p *eventFactory[I]) newEventHeader() *eventHeader {
	return &eventHeader{
		HeaderEventMetadata:    p.eventMetadata,
		HeaderCreator:          p.eventSourceID,
		HeaderSender:           p.eventSourceID,
		HeaderCreationTime:     time.Now(),
		HeaderEventID:          p.eventIDProvider.NextEventID(),
		HeaderIsBroadcastEvent: false,
		lock:                   sync.RWMutex{},
	}
}
