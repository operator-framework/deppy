package catalogsource

import (
	"github.com/operator-framework/deppy/internal/entitysource/adapter/api"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"context"
	"fmt"
	"time"
)

const defaultCatsrcTimeout = 5 * time.Minute

type DeppyAdapter struct {
	api.UnimplementedDeppySourceAdapterServer
	catsrc        *v1alpha1.CatalogSource
	catsrcLister  *Lister
	catsrcTimeout time.Duration
	logger        *logrus.Entry
}

func NewCatalogSourceDeppyAdapter(opts ...AdapterOptions) (*DeppyAdapter, error) {
	lister, err := NewCatalogSourceLister(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to create CatalogSource Lister: %v", err)
	}
	c := &DeppyAdapter{
		UnimplementedDeppySourceAdapterServer: api.UnimplementedDeppySourceAdapterServer{},
		catsrcTimeout:                         defaultCatsrcTimeout,
		logger:                                logrus.NewEntry(logrus.New()),
		catsrcLister:                          lister,
	}
	for _, o := range opts {
		o(c)
	}
	if c.catsrc == nil {
		return nil, fmt.Errorf("CatalogSource-DeppyAdapter requires non-nil catsrc")
	}
	return c, nil
}

type AdapterOptions func(*DeppyAdapter)

func WithLogger(l *logrus.Entry) AdapterOptions {
	return func(c *DeppyAdapter) {
		c.logger = l
	}
}

func WithTimeout(d time.Duration) AdapterOptions {
	return func(c *DeppyAdapter) {
		c.catsrcTimeout = d
	}
}

func WithSourceAddress(name, a string) AdapterOptions {
	return func(c *DeppyAdapter) {
		c.catsrc = &v1alpha1.CatalogSource{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
			Spec: v1alpha1.CatalogSourceSpec{
				Address: a,
			},
		}
	}
}

func WithNamespacedSource(name, namespace string) AdapterOptions {
	return func(c *DeppyAdapter) {
		c.catsrc = &v1alpha1.CatalogSource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		}
	}
}

func (s *DeppyAdapter) ListEntities(_ *api.ListEntitiesRequest, stream api.DeppySourceAdapter_ListEntitiesServer) error {
	s.logger.Infof("ListEntities: %v, catsrc %s/%s (%s)", time.Now(), s.catsrc.Namespace, s.catsrc.Name, s.catsrc.Spec.Address)
	// TODO: watch catsrc for changes
	if s.catsrc.Namespace != "" && s.catsrc.Name != "" {
		catsrc, err := s.catsrcLister.Namespace(s.catsrc.Namespace).Get(s.catsrc.Name)
		if err != nil {
			s.logger.Errorf("ListEntities: Failed to list: %v", err)
			return err
		}
		s.catsrc = catsrc
	}

	entities, err := s.listEntities(stream.Context())
	if err != nil {
		return err
	}
	for _, e := range entities {
		if err := stream.Send(&api.Entity{Id: string(e.Eid), Properties: e.Properties}); err != nil {
			return err
		}
	}

	return nil
}
