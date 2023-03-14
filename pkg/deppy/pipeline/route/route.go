package route

import (
	"context"
	"sync"

	"github.com/operator-framework/deppy/pkg/deppy/pipeline"
)

// Route is a part of Routing table that tracks the input and output
// channel of the event source
type Route struct {
	// Event Source Id
	eventSourceID pipeline.EventSourceID
	// source of the event
	inputChannel chan pipeline.Event
	// Destination channel
	outputChannel chan pipeline.Event
	// whether the input channel has been closed, can happen when:
	// 1. the event source has closed its output channel (signalling end-of-output/processing)
	// 2. closed by an auto-close because all input sources into this event source have closed their output channels (end-of-input)
	inputChannelClosed bool
	// both input and output channels are closed
	connectionDoneListeners map[chan<- struct{}]struct{}

	lock sync.RWMutex
}

func (r *Route) CloseInputChannel() {
	r.lock.Lock()
	defer r.lock.Unlock()
	if !r.inputChannelClosed && r.inputChannel != nil {
		close(r.inputChannel)
		r.inputChannelClosed = true
	}
}

func (c *Route) InputChannel() <-chan pipeline.Event {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.inputChannel
}

func (c *Route) OutputChannel() chan<- pipeline.Event {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.outputChannel
}

func (c *Route) notifyConnectionDoneListeners(ctx context.Context) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	for doneCh := range c.connectionDoneListeners {
		go func(doneCh chan<- struct{}) {
			select {
			case <-ctx.Done():
				return
			case doneCh <- struct{}{}:
				return
			}
		}(doneCh)
	}
}

func (c *Route) ConnectionDone(ctx context.Context) {
	c.CloseInputChannel()
	c.notifyConnectionDoneListeners(ctx)
}
