package entitysource

import "context"

var _ EntityContentGetter = &NoContentSource{}

type NoContentSource struct{}

func (n *NoContentSource) GetContent(_ context.Context, _ EntityID) (interface{}, error) {
	return nil, nil
}
