package input

import (
	"github.com/operator-framework/deppy/pkg/solver"
)

type Entity struct {
	ID         solver.Identifier `json:"identifier"`
	Properties map[string]string `json:"properties"`
}

func (e *Entity) Identifier() solver.Identifier {
	return e.ID
}

func NewEntity(id solver.Identifier, properties map[string]string) *Entity {
	return &Entity{
		ID:         id,
		Properties: properties,
	}
}
