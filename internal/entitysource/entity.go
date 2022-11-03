package entitysource

import "fmt"

type EntityID string

type EntityPropertyNotFoundError string

func (p EntityPropertyNotFoundError) Error() string {
	return fmt.Sprintf("Property '(%s)' Not Found", string(p))
}

type Entity struct {
	Eid        EntityID          `json:"id,omitempty"`
	Properties map[string]string `json:"properties,omitempty"`
}

func NewEntity(id EntityID, properties map[string]string) *Entity {
	return &Entity{
		Eid:        id,
		Properties: properties,
	}
}

func (e *Entity) ID() EntityID {
	return e.Eid
}

func (e *Entity) GetProperty(key string) (string, error) {
	value, ok := e.Properties[key]
	if !ok {
		return "", EntityPropertyNotFoundError(key)
	}
	return value, nil
}
