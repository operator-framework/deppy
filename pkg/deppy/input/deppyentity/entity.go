package deppyentity

import (
	"github.com/operator-framework/deppy/pkg/deppy"
)

type Entity struct {
	ID         deppy.Identifier  `json:"identifier"`
	Properties map[string]string `json:"properties"`
}

func (e *Entity) Identifier() deppy.Identifier {
	return e.ID
}

func NewEntity(id deppy.Identifier, properties map[string]string) *Entity {
	return &Entity{
		ID:         id,
		Properties: properties,
	}
}
