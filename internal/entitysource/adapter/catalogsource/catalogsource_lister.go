package catalogsource

import (
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"context"
)

type Lister struct {
	client client.Client
	ctx    context.Context
}

type NamespaceLister struct {
	namespace string
	lister    *Lister
}

func NewCatalogSourceLister(ctx context.Context) (*Lister, error) {
	scheme := runtime.NewScheme()
	if err := v1alpha1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	c, err := client.New(ctrl.GetConfigOrDie(), client.Options{
		Scheme: scheme,
	})
	if err != nil {
		return nil, err
	}

	return &Lister{
		client: c,
		ctx:    ctx,
	}, nil
}

func (c *Lister) Namespace(namespace string) *NamespaceLister {
	return &NamespaceLister{
		namespace: namespace,
		lister:    c,
	}
}

func (c *NamespaceLister) List() (*v1alpha1.CatalogSourceList, error) {
	list := v1alpha1.CatalogSourceList{}
	if err := c.lister.client.List(c.lister.ctx, &list, &client.ListOptions{Namespace: c.namespace}); err != nil {
		return nil, err
	}
	return &list, nil
}

func (c *NamespaceLister) Get(name string) (*v1alpha1.CatalogSource, error) {
	catsrc := v1alpha1.CatalogSource{}
	if err := c.lister.client.Get(c.lister.ctx, client.ObjectKey{Namespace: c.namespace, Name: name}, &catsrc); err != nil {
		return nil, err
	}
	return &catsrc, nil
}
