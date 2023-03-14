package route

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/deppy/pkg/deppy/pipeline"
	"github.com/operator-framework/deppy/pkg/deppy/pipeline/event"
)

var _ = Describe("Router", func() {
	defer GinkgoRecover()

	var (
		ctx context.Context
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
		var (
			router *Router
		)
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

	When("routing an event", func() {
		var (
			router *Router
		)
		BeforeEach(func() {
			router = NewEventRouter(ctx)
			Expect(router.debugChannel).To(BeNil())

			testSource := &testeventSource{"node"}
			router.AddRoute(testSource)
		})

		It("routes the event as expected", func() {
			evt := getNewTestEvent()
			evt.Broadcast()

			Expect(func() {
				router.route(evt)
			}).NotTo(Panic())

			router.ctx.Done()
		})

	})

})

func TestRouter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Router tests")
}

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
