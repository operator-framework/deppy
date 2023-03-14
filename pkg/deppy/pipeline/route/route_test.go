package route

import (
	"sync"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/deppy/pkg/deppy/pipeline"
)

var _ = Describe("Route", func() {
	defer GinkgoRecover()

	var (
		testroute *Route
	)

	BeforeEach(func() {
		testroute = &Route{
			eventSourceID:           pipeline.EventSourceID("testEvent"),
			inputChannel:            make(chan pipeline.Event),
			outputChannel:           make(chan pipeline.Event),
			connectionDoneListeners: make(map[chan<- struct{}]struct{}),
			lock:                    sync.RWMutex{},
		}
	})

	When("closing of Channel", func() {
		It("closing a non-nil channel", func() {
			testroute.inputChannelClosed = false
			testroute.CloseInputChannel()
			Expect(testroute.inputChannelClosed).To(BeTrue())
			Expect(isClosed(testroute.inputChannel)).To(BeTrue())
		})

		It("closing an already closed channel", func() {
			By("closing the channel before ")
			close(testroute.inputChannel)
			testroute.inputChannelClosed = true

			By("making sure it doesn't panic")
			Expect(func() {
				testroute.CloseInputChannel()
			}).NotTo(Panic())
			Expect(testroute.inputChannelClosed).To(BeTrue())
		})

		It("closing a nil channel", func() {
			By("setting input channel parameters")
			testroute.inputChannelClosed = false
			testroute.inputChannel = nil

			By("making sure it doesn't panic")
			Expect(func() {
				testroute.CloseInputChannel()
			}).NotTo(Panic())
		})
	})

	When("fetching input and output channel", func() {
		It("should fetch input and output channels correctly", func() {
			Expect(testroute.InputChannel()).NotTo(BeNil())
			Expect(testroute.OutputChannel()).NotTo(BeNil())
		})
	})

	When("notifyConnectionDoneListers", func() {
		// TODO: finish up tests for this

	})

})

func isClosed(ch <-chan pipeline.Event) bool {
	select {
	case <-ch:
		return true
	default:
	}
	return false
}

func TestRoute(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Route tests")
}
