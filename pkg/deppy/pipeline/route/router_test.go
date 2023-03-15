package route

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/deppy/pkg/deppy/pipeline"
	"github.com/operator-framework/deppy/pkg/deppy/pipeline/event"
)

var _ = Describe("Router", func() {
	defer GinkgoRecover()

	var (
		ctx    context.Context
		router *Router
	)

	BeforeEach(func() {
		ctx = context.TODO()
	})

	When("test creating of new router", func() {
		It("creating a new router", func() {
			router := NewEventRouter(ctx)
			Expect(router.routeTable).NotTo(BeNil())
			Expect(router.ctx).NotTo(BeNil())
			Expect(router.debugChannel).To(BeNil())
			Expect(router.errChannel).To(BeNil())
		})

		It("create a new router with options", func() {
			debugChannel := make(chan<- pipeline.Event)
			errChannel := make(chan<- pipeline.ErrorEvent)

			router := NewEventRouter(ctx, WithDebugChannel(debugChannel), WithErrorChannel(errChannel))
			Expect(router.routeTable).NotTo(BeNil())
			Expect(router.ctx).NotTo(BeNil())
			Expect(router.debugChannel).NotTo(BeNil())
			Expect(router.errChannel).NotTo(BeNil())
		})
	})

	When("adding a route", func() {
		BeforeEach(func() {
			router = NewEventRouter(ctx)
		})

		It("Adding a new route", func() {
			testSource := &testeventSource{"node"}
			res := router.AddRoute(testSource)
			Expect(res).To(BeTrue())
			Expect(len(router.routeTable)).To(BeEquivalentTo(1))

			conn := router.routeTable[testSource.EventSourceID()]
			Expect(conn).NotTo(BeNil())
			Expect(conn.eventSourceID).To(BeEquivalentTo(testSource.EventSourceID()))
			Expect(conn.inputChannel).NotTo(BeNil())
			Expect(conn.outputChannel).NotTo(BeNil())
			Expect(conn.connectionDoneListeners).NotTo(BeNil())
		})

		It("Adding an existing route again", func() {
			testSource1 := &testeventSource{"node"}
			testSource2 := &testeventSource{"node"}

			res := router.AddRoute(testSource1)
			Expect(res).To(BeTrue())

			res = router.AddRoute(testSource2)
			Expect(res).To(BeFalse())

			Expect(len(router.routeTable)).To(BeEquivalentTo(1))
		})

		It("Adding different routes", func() {
			testSource1 := &testeventSource{"node1"}
			testSource2 := &testeventSource{"node2"}

			res := router.AddRoute(testSource1)
			Expect(res).To(BeTrue())

			res = router.AddRoute(testSource2)
			Expect(res).To(BeTrue())

			Expect(len(router.routeTable)).To(BeEquivalentTo(2))
		})
	})

	When("fetching the right route", func() {
		BeforeEach(func() {
			router = NewEventRouter(ctx)
			Expect(router.debugChannel).To(BeNil())

			testSource1 := &testeventSource{"node1"}
			router.AddRoute(testSource1)

			testSource2 := &testeventSource{"node2"}
			router.AddRoute(testSource2)

			Expect(len(router.routeTable)).To(BeEquivalentTo(2))
		})

		It("fetches the route to an existing event source", func() {
			route, exists := router.GetRoute(pipeline.EventSourceID("node1"))
			Expect(route).NotTo(BeNil())
			Expect(exists).To(BeTrue())
		})

		It("fetches nil route to a non-existing event source", func() {
			route, exists := router.GetRoute(pipeline.EventSourceID("notexistentNode"))
			Expect(route).To(BeNil())
			Expect(exists).To(BeFalse())
		})

		It("fetches all routes in the routing table", func() {
			allRoutes := router.getAllRoutes()
			Expect(len(allRoutes)).To(BeEquivalentTo(2))
		})

	})

	When("routing an event to a specific input channel", func() {
		BeforeEach(func() {
			router = NewEventRouter(ctx)
			// TODO: add tests with a non-nil debug channel
			Expect(router.debugChannel).To(BeNil())

			testSource1 := &testeventSource{"node1"}
			router.AddRoute(testSource1)

			testSource2 := &testeventSource{"node2"}
			router.AddRoute(testSource2)

			Expect(len(router.routeTable)).To(BeEquivalentTo(2))
		})

		AfterEach(func() {
			router.ctx.Done()
		})

		It("routes a broadcast event as expected", func() {
			By("making the event to be a broadcast event")
			evt := getNewTestEvent()
			evt.Broadcast()

			By("ensuring that event header is correctly set for broadcast")
			Expect(evt.Header().IsBroadcastEvent()).To(BeTrue())

			Expect(func() {
				router.route(evt)
			}).NotTo(Panic())

			By("verifying if each input channel in the routing table has received the event")

			noData := true
			for _, conn := range router.routeTable {
				select {
				case data, exists := <-conn.inputChannel:
					Expect(exists).To(BeTrue())
					Expect(data).NotTo(BeNil())
					Expect(data.String()).To(ContainSubstring("testEvent"))
				default:
					// It shouldn't be reaching this. Hence checking with a boolean
					noData = false
				}
				Expect(noData).To(BeTrue())
			}
		})

		It("routes an event to a specific receiver", func() {
			By("creating a new event and setting its receiver")
			evt := getNewTestEvent()
			evt.Route("node2")

			By("ensuring that the header of the event is set as expected")
			Expect(evt.Header().Receiver()).To(BeEquivalentTo("node2"))

			Expect(func() {
				router.route(evt)
			}).NotTo(Panic())

			By("verifying if the event has not been routed to node1")
			connForNode1, exists := router.GetRoute(pipeline.EventSourceID("node1"))
			Expect(exists).To(BeTrue())
			Expect(connForNode1).NotTo(BeNil())

			noData := false
			select {
			// Shouldn't be entering this case. Do we remove this?
			case receivedMsg, exists := <-connForNode1.inputChannel:
				Expect(receivedMsg).To(BeNil())
				Expect(exists).To(BeFalse())
			default:
				noData = true
			}
			Expect(noData).To(BeTrue())

			By("verifying if the event has been routed to node2")
			connForNode2, exists := router.GetRoute(pipeline.EventSourceID("node2"))
			Expect(exists).To(BeTrue())
			Expect(connForNode2).NotTo(BeNil())

			noData = true
			select {
			case receivedMsg, exists := <-connForNode2.inputChannel:
				Expect(receivedMsg.String()).To(ContainSubstring("testEvent"))
				Expect(exists).To(BeTrue())
			default:
				// Shouldn't be entering default.
				noData = false
			}
			Expect(noData).To(BeTrue())
		})
	})

	When("send output event", func() {
		BeforeEach(func() {
			router = NewEventRouter(ctx)
			testSource1 := &testeventSource{"node1"}
			router.AddRoute(testSource1)

			Expect(len(router.routeTable)).To(BeEquivalentTo(1))
		})

		It("correctly routes the events", func() {
			// Create an event with its receiver set to node2
			By("creating a new event and setting its receiver")
			evt := getNewTestEvent()
			evt.Route("node2")

			By("push this event into node 1's output channel")
			Expect(func() {
				router.SendOutputEvent(evt, pipeline.EventSourceID("node1"))
			}).NotTo(Panic())

			By("verifying if the event has been sent successfully")
			conn, ok := router.GetRoute("node1")
			Expect(ok).To(BeTrue())
			Expect(conn).NotTo(BeNil())

			noData := false
			select {
			case receivedMsg, exists := <-conn.outputChannel:
				Expect(receivedMsg.String()).To(ContainSubstring("testEvent"))
				Expect(exists).To(BeTrue())
			default:
				// Shouldn't be entering default.
				noData = true
			}
			Expect(noData).To(BeFalse())

		})

		AfterEach(func() {
			router.ctx.Done()
		})
	})
})

func getNewTestEvent() pipeline.Event {
	factory := event.NewEventFactory[string]("testnode")
	return factory.NewDataEvent("testEvent")
}

type testeventSource struct {
	eventsourceID string
}

var _ pipeline.EventSource = &testeventSource{}

func (t *testeventSource) EventSourceID() pipeline.EventSourceID {
	return pipeline.EventSourceID(t.eventsourceID)
}

func (t *testeventSource) IngressCapacity() int {
	return 10
}
