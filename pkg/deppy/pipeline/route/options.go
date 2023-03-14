package route

import "github.com/operator-framework/deppy/pkg/deppy/pipeline"

type Option func(router *Router)

func WithDebugChannel(debugChannel chan<- pipeline.Event) Option {
	return func(router *Router) {
		router.debugChannel = debugChannel
	}
}

func WithErrorChannel(errChannel chan<- pipeline.ErrorEvent) Option {
	return func(router *Router) {
		router.errChannel = errChannel
	}
}
