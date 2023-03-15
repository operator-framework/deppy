package route

import (
	"context"
	"sync"

	"github.com/operator-framework/deppy/pkg/deppy/pipeline"
)

type Router struct {
	// contains entries that map the route table and the route.
	routeTable map[pipeline.EventSourceID]*Route
	// debugChannel to send debug events
	debugChannel chan<- pipeline.Event
	// errChannel to send error events
	errChannel chan<- pipeline.ErrorEvent
	lock       sync.RWMutex
	ctx        context.Context
}

func NewEventRouter(ctx context.Context, opts ...Option) *Router {
	router := &Router{
		routeTable: map[pipeline.EventSourceID]*Route{},
		lock:       sync.RWMutex{},
		ctx:        ctx,
	}

	for _, applyOption := range opts {
		applyOption(router)
	}

	// close debug and error channel when context expires
	if router.debugChannel != nil {
		go func(router *Router) {
			<-router.ctx.Done()
			close(router.debugChannel)
		}(router)
	}

	if router.errChannel != nil {
		go func(router *Router) {
			<-router.ctx.Done()
			close(router.errChannel)
		}(router)
	}
	return router
}

func (r *Router) AddRoute(eventSource pipeline.EventSource) bool {
	r.lock.Lock()
	defer r.lock.Unlock()

	if _, ok := r.routeTable[eventSource.EventSourceID()]; !ok {
		conn := Route{
			eventSourceID: eventSource.EventSourceID(),
			inputChannel:  make(chan pipeline.Event, eventSource.IngressCapacity()),
			// output channel should not be limited.
			outputChannel:           make(chan pipeline.Event, eventSource.IngressCapacity()),
			connectionDoneListeners: map[chan<- struct{}]struct{}{},
			inputChannelClosed:      false,
		}
		r.routeTable[eventSource.EventSourceID()] = &conn
		return true
	}
	return false
}

// DeleteRoute deletes a particular route from the routing table.
func (r *Router) DeleteRoute(eventSourceID pipeline.EventSourceID) {
	r.lock.Lock()
	defer r.lock.Unlock()
	delete(r.routeTable, eventSourceID)
}

// send a debug event through the debug channel
func (r *Router) debug(event pipeline.Event) {
	if r.debugChannel != nil {
		select {
		case r.debugChannel <- event:
			return
		case <-r.ctx.Done():
			return
		}
	}
}

// send a debug event through the error channel
func (r *Router) error(event pipeline.ErrorEvent) {
	if r.errChannel != nil {
		select {
		case r.errChannel <- event:
			return
		case <-r.ctx.Done():
			return
		}
	}
}

// Route takes in the sending event source, fetches the connection details from the
// routing channel. Takes the content from its output channel and sends over the event
// to the right input channel as specified in the event header.
func (r *Router) Route(senderEventSource pipeline.EventSource) {
	route, ok := r.routeTable[senderEventSource.EventSourceID()]
	if !ok {
		// TODO: send an error event in the error channel
		return
	}

	go func(rte *Route) {
		defer func() {
			// Delete the event sourceID from the router after routing.
			r.DeleteRoute(rte.eventSourceID)
			// close the connection
			rte.ConnectionDone(r.ctx)
		}()

		for {
			select {
			case <-r.ctx.Done():
				return
			case event, hasNext := <-rte.outputChannel:
				if !hasNext {
					return
				}
				r.route(event)
			}
		}
	}(route)
}

// route an event based on the header information on the event.
// If the event is a broadcast event, we fetch all the routes from
// the routing table and send the event to all.
// If the receiver is specified in the header of the event, the route
// for the specified event is fetched and the event is directed to its
// input channel.
func (r *Router) route(event pipeline.Event) {
	// debug is a blocking call if the debug channel is set
	r.debug(event)

	if event.Header().IsBroadcastEvent() {
		conns := r.getAllRoutes()

		for _, conn := range conns {
			if conn.eventSourceID != event.Header().Sender() {
				func() {
					if conn.inputChannel == nil {
					}
					select {
					case conn.inputChannel <- event:
						return
					case <-r.ctx.Done():
						return
					}
				}()
			}
		}
	} else if event.Header().Receiver() != "" {
		conn, ok := r.GetRoute(event.Header().Receiver())
		if !ok {
			// TODO: create an error event and send it through the error channel.
		} else {
			select {
			case conn.inputChannel <- event:
				return
			case <-r.ctx.Done():
				return
			}
		}
	} else {
		// TODO: send an event through error channel.
		// errChannel should be non-blocking.
	}
}

// RouteTo returns the route for the particular event source from the routing table.
func (r *Router) GetRoute(eventSourceID pipeline.EventSourceID) (*Route, bool) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	conn, ok := r.routeTable[eventSourceID]
	return conn, ok
}

func (r *Router) getAllRoutes() []*Route {
	r.lock.Lock()
	defer r.lock.Unlock()

	var out []*Route
	for _, conn := range r.routeTable {
		out = append(out, conn)
	}
	return out
}

// Sendevent sends an event to the eventsource's output channel.
// This is used by the node to send an event after processing. For this to be used,
// the event source ID should be in the routing table.
func (r *Router) SendOutputEvent(event pipeline.Event, sendingSource pipeline.EventSourceID) {
	conn, ok := r.GetRoute(sendingSource)
	if !ok {
		// send an error through errChannel
		return
	}

	r.lock.Lock()
	defer r.lock.Unlock()
	if conn != nil && conn.outputChannel != nil {
		func() {
			select {
			case conn.outputChannel <- event:
				return
			case <-r.ctx.Done():
				return
			}
		}()
	}

}
