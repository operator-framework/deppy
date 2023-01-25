package event_test

import (
	"encoding/json"
	"fmt"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/operator-framework/deppy/pkg/deppy/pipeline"
	"github.com/operator-framework/deppy/pkg/deppy/pipeline/event"
)

var _ pipeline.EventIDProvider = &fakeCustomEventIDProvider{}

type fakeCustomEventIDProvider struct {
	fn func() pipeline.EventID
}

func (f fakeCustomEventIDProvider) NextEventID() pipeline.EventID {
	return f.fn()
}

func TestEventFactory_NewDataEvent(t *testing.T) {
	factory := event.NewEventFactory[int]("node")
	dataEvent := factory.NewDataEvent(1)
	assert.Equal(t, 1, dataEvent.Data())
	assert.Equal(t, pipeline.EventSourceID("node"), dataEvent.Header().Creator())
	assert.Nil(t, dataEvent.Header().Metadata())
	assert.Empty(t, dataEvent.Header().Receiver())
	assert.False(t, dataEvent.Header().IsBroadcastEvent())
	assert.Equal(t, pipeline.EventSourceID("node"), dataEvent.Header().Sender())
	assert.NotNil(t, dataEvent.Header().CreationTime())
	assert.NotNil(t, dataEvent.Header().EventID())
}

func TestEventFactory_NewErrorEvent(t *testing.T) {
	factory := event.NewEventFactory[int]("node")
	errorEvent := factory.NewErrorEvent(fmt.Errorf("some error"))
	assert.Equal(t, fmt.Errorf("some error"), errorEvent.Error())
	assert.Equal(t, pipeline.EventSourceID("node"), errorEvent.Header().Creator())
	assert.Nil(t, errorEvent.Header().Metadata())
	assert.Empty(t, errorEvent.Header().Receiver())
	assert.False(t, errorEvent.Header().IsBroadcastEvent())
	assert.Equal(t, pipeline.EventSourceID("node"), errorEvent.Header().Sender())
	assert.NotNil(t, errorEvent.Header().CreationTime())
	assert.NotNil(t, errorEvent.Header().EventID())
}

func TestEventFactory_NewEventWithMetadata(t *testing.T) {
	eventMetadata := pipeline.EventMetadata{"meta": "data"}
	factory := event.NewEventFactory[int]("node", event.WithEventMetadata[int](eventMetadata))
	dataEvent := factory.NewDataEvent(1)
	errorEvent := factory.NewErrorEvent(fmt.Errorf("some error"))
	assert.Equal(t, eventMetadata, dataEvent.Header().Metadata())
	assert.Equal(t, eventMetadata, errorEvent.Header().Metadata())
}

func TestEventFactory_NewEventWithCustomEventIDProvider(t *testing.T) {
	staticEventID := pipeline.EventID("1")
	eventIDProvider := fakeCustomEventIDProvider{
		fn: func() pipeline.EventID {
			return staticEventID
		},
	}
	factory := event.NewEventFactory[int]("node", event.WithEventIDProvider[int](eventIDProvider))
	dataEvent := factory.NewDataEvent(1)
	errorEvent := factory.NewErrorEvent(fmt.Errorf("some error"))
	assert.Equal(t, staticEventID, dataEvent.Header().EventID())
	assert.Equal(t, staticEventID, errorEvent.Header().EventID())
}

func TestEventFactory_Broadcast(t *testing.T) {
	factory := event.NewEventFactory[int]("node")
	evt := factory.NewDataEvent(1)
	evt.Broadcast()
	assert.True(t, evt.Header().IsBroadcastEvent())
}

func TestEventFactory_Route(t *testing.T) {
	factory := event.NewEventFactory[int]("node")
	evt := factory.NewDataEvent(1)
	evt.Route("node2")
	assert.Equal(t, pipeline.EventSourceID("node"), evt.Header().Sender())
	assert.Equal(t, pipeline.EventSourceID("node2"), evt.Header().Receiver())
}

func TestEventFactory_JSONSerializable(t *testing.T) {
	staticEventID := pipeline.EventID("1")
	eventIDProvider := fakeCustomEventIDProvider{
		fn: func() pipeline.EventID {
			return staticEventID
		},
	}
	factory := event.NewEventFactory[int]("node", event.WithEventIDProvider[int](eventIDProvider))
	evt := factory.NewDataEvent(1)
	bytes, err := json.Marshal(evt)
	assert.Nil(t, err)
	regExp, err := regexp.Compile(`\{"header":\{"eventID":"1","creatorEventSourceID":"node","sender":"node","broadcast":false,"creationTime":"\d\d\d\d-\d\d-\d\dT\d\d:\d\d:\d\d\..*"},"data":1}`)
	assert.Nil(t, err)
	assert.True(t, regExp.Match(bytes), "Event JSON representation (%s) does not match expected regexp (%s)", string(bytes), regExp.String())
}
