package entitysource

import "fmt"

type EntityID string

type EntityPropertyNotFoundError string

func (p EntityPropertyNotFoundError) Error() string {
	return fmt.Sprintf("Property '(%s)' Not Found", string(p))
}

type Entity struct {
	id         EntityID
	properties map[string]string
}

func NewEntity(id EntityID, properties map[string]string) *Entity {
	return &Entity{
		id:         id,
		properties: properties,
	}
}

func (e *Entity) ID() EntityID {
	return e.id
}

func (e *Entity) GetProperty(key string) (string, error) {
	value, ok := e.properties[key]
	if !ok {
		return "", EntityPropertyNotFoundError(key)
	}
	return value, nil
}
