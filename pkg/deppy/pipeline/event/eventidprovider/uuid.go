package eventidprovider

import (
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/operator-framework/deppy/pkg/deppy/pipeline"
)

var _ pipeline.EventIDProvider = &UUIDEventIDProvider{}

type UUIDProviderFn func() (uuid.UUID, error)

type UUIDEventIDProvider struct {
	nextUUIDFn UUIDProviderFn
}

func NewUUIDEventIDProvider() *UUIDEventIDProvider {
	return &UUIDEventIDProvider{
		nextUUIDFn: func() (uuid.UUID, error) { return uuid.NewRandom() },
	}
}

func NewCustomUUIDEventIDProvider(nextUUIDFn UUIDProviderFn) *UUIDEventIDProvider {
	return &UUIDEventIDProvider{
		nextUUIDFn: nextUUIDFn,
	}
}

func (p *UUIDEventIDProvider) NextEventID() pipeline.EventID {
	eid, err := p.nextUUIDFn()
	if err != nil {
		id := err.Error() + time.Now().String()
		id = hex.EncodeToString([]byte(id))
		return pipeline.EventID(fmt.Sprintf("%s (with error: %s)", id, err))
	}
	return pipeline.EventID(hex.EncodeToString([]byte(eid.String())))
}
