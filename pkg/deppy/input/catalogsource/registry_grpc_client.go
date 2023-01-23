package catalogsource

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	catalogsourceapi "github.com/operator-framework/operator-registry/pkg/api"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/operator-framework/deppy/pkg/deppy/input"
	"github.com/operator-framework/deppy/pkg/lib/grpc"
)

type RegistryClient interface {
	ListEntities(ctx context.Context, catsrc *v1alpha1.CatalogSource) ([]*input.Entity, error)
}

type registryGRPCClient struct {
	timeout time.Duration
	client  client.Client
}

func NewRegistryGRPCClient(grpcTimeout time.Duration, client client.Client) RegistryClient {
	if grpcTimeout == 0 {
		grpcTimeout = grpc.DefaultGRPCTimeout
	}
	return &registryGRPCClient{timeout: grpcTimeout, client: client}
}

func (r *registryGRPCClient) ListEntities(ctx context.Context, catalogSource *v1alpha1.CatalogSource) ([]*input.Entity, error) {
	// TODO: create GRPC connections separately
	conn, err := grpc.ConnectWithTimeout(ctx, catalogSource.Address(), r.timeout)
	if conn != nil {
		defer conn.Close()
	}
	if err != nil {
		return nil, err
	}

	catsrcClient := catalogsourceapi.NewRegistryClient(conn)
	stream, err := catsrcClient.ListBundles(ctx, &catalogsourceapi.ListBundlesRequest{})

	if err != nil {
		return nil, fmt.Errorf("ListBundles failed: %v", err)
	}

	var entities []*input.Entity
	catalogPackages := map[string]*catalogsourceapi.Package{}
	catalogSourceID := fmt.Sprintf("%s/%s", catalogSource.Namespace, catalogSource.Name)
	for {
		bundle, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return entities, fmt.Errorf("failed to read bundle stream: %v", err)
		}

		packageKey := fmt.Sprintf("%s/%s", catalogSourceID, bundle.PackageName)
		pkg, ok := catalogPackages[packageKey]
		if !ok {
			pkg, err = catsrcClient.GetPackage(ctx, &catalogsourceapi.GetPackageRequest{Name: bundle.PackageName})
			if err != nil {
				return entities, fmt.Errorf("failed to get package %s: %v", bundle.PackageName, err)
			}
			catalogPackages[packageKey] = pkg
		}

		entity, err := entityFromBundle(catalogSourceID, pkg, bundle)
		if err != nil {
			return entities, fmt.Errorf("failed to parse entity %s: %v", entity.Identifier(), err)
		}
		entities = append(entities, entity)
	}
	return entities, nil
}
