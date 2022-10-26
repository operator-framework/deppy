package entitysource

import "context"

var _ EntityContentGetter = &noContentSource{}

type noContentSource struct{}

func NoContentSource() EntityContentGetter {
	return &noContentSource{}
}

func (n *noContentSource) GetContent(_ context.Context, _ EntityID) (interface{}, error) {
	return nil, nil
}
