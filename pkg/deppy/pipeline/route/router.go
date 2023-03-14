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
			eventSourceID:           eventSource.EventSourceID(),
			inputChannel:            make(chan pipeline.Event),
			outputChannel:           make(chan pipeline.Event),
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

func (r *Router) Route(event pipeline.Event, receiverEventSource pipeline.EventSource) {
	route, ok := r.routeTable[receiverEventSource.EventSourceID()]
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

// route an event based on the header information on the event
func (r *Router) route(event pipeline.Event) {
	// debug is a blocking call if the debug channel is set
	r.debug(event)

	if event.Header().IsBroadcastEvent() {
		conns := r.getAllRoutes()

		for _, conn := range conns {
			if conn.eventSourceID != event.Header().Sender() {
				func() {
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
		conn, ok := r.GetReceiver(event.Header().Receiver())
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
	}
}

// RouteTo returns the route for the particular event source from the routing table.
func (r *Router) GetReceiver(eventSourceID pipeline.EventSourceID) (*Route, bool) {
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
