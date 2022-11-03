package catalogsource

import (
	"github.com/operator-framework/deppy/internal/entitysource"
	"github.com/operator-framework/deppy/internal/lib/grpc"
	"github.com/operator-framework/deppy/internal/lib/util"

	"github.com/operator-framework/operator-registry/alpha/property"
	catalogsourceapi "github.com/operator-framework/operator-registry/pkg/api"
	errors2 "k8s.io/apimachinery/pkg/util/errors"

	"context"
	"errors"
	"fmt"
	"io"
	"sort"
)

type UpgradeEdge struct {
	property.Channel
	Replaces  string   `json:"replaces,omitempty"`
	Skips     []string `json:"skips,omitempty"`
	SkipRange string   `json:"skipRange,omitempty"`
}

type BundleEntityID struct {
	CSVName string `json:"name,omitempty"`
	Package string `json:"package,omitempty"`
	Version string `json:"version,omitempty"`
	Source  string `json:"source,omitempty"`
	Path    string `json:"path,omitempty"`
}

func (e BundleEntityID) EntityID() (entitysource.EntityID, error) {
	id, err := util.JSONMarshal(e)
	return entitysource.EntityID(id), err
}

func BundleID(id entitysource.EntityID) (BundleEntityID, error) {
	var bid BundleEntityID
	err := util.JSONUnmarshal([]byte(id), &bid)
	return bid, err
}

// ListEntities implementation
func (s *DeppyAdapter) listEntities(ctx context.Context) ([]*entitysource.Entity, error) {
	// TODO: sync catsrc state separately
	if s.catsrc.Namespace != "" && s.catsrc.Name != "" {
		catsrc, err := s.catsrcLister.Namespace(s.catsrc.Namespace).Get(s.catsrc.Name)
		if err != nil {
			return nil, err
		}
		s.catsrc = catsrc
	}

	conn, err := grpc.ConnectWithTimeout(ctx, s.catsrc.Address(), s.catsrcTimeout)
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

	var entities []*entitysource.Entity
	for {
		bundle, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return entities, fmt.Errorf("failed to read bundle stream: %v", err)
		}

		entity, err := entityFromBundle(s.catsrc.Name, bundle)
		if err != nil {
			return entities, fmt.Errorf("failed to parse entity %s on bundle stream: %v", entity.ID(), err)
		}
		entities = append(entities, entity)
	}

	entities, err = deduplicate(entities)
	if err != nil {
		return nil, fmt.Errorf("failed to deduplicate properties for entites: %v", err)
	}
	return entities, nil
}

func entityFromBundle(catsrcName string, bundle *catalogsourceapi.Bundle) (*entitysource.Entity, error) {
	id, err := idFromBundle(catsrcName, bundle)
	if err != nil {
		return nil, err
	}
	props, err := propertiesFromBundle(catsrcName, bundle)
	if err != nil {
		return nil, err
	}
	return entitysource.NewEntity(id, props), nil
}

func idFromBundle(catsrcID string, bundle *catalogsourceapi.Bundle) (entitysource.EntityID, error) {
	return BundleEntityID{
		CSVName: bundle.CsvName, // can possibly be dropped in future
		Version: bundle.Version,
		Package: bundle.PackageName,
		Path:    bundle.BundlePath,
		Source:  catsrcID,
	}.EntityID()
}

func propertiesFromBundle(catsrcName string, bundle *catalogsourceapi.Bundle) (map[string]string, error) {
	properties := map[string]string{}
	errors := []error{}

	propsList := map[string]map[string]struct{}{
		property.TypeGVK:         {},
		property.TypeGVKRequired: {},
		property.TypePackage:     {},
		property.TypeChannel:     {},
	}

	for _, prvAPI := range bundle.ProvidedApis {
		apiValue, err := util.JSONMarshal(prvAPI)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		propsList[property.TypeGVK][string(apiValue)] = struct{}{}
	}

	for _, reqAPI := range bundle.RequiredApis {
		apiValue, err := util.JSONMarshal(reqAPI)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		propsList[property.TypeGVKRequired][string(apiValue)] = struct{}{}
	}

	for _, reqAPI := range bundle.Dependencies {
		propsList[property.TypeGVKRequired][reqAPI.Value] = struct{}{}
	}

	pkgValue, err := util.JSONMarshal(property.Package{
		PackageName: bundle.PackageName,
		Version:     bundle.Version,
	})
	if err != nil {
		errors = append(errors, err)
	} else {
		propsList[property.TypePackage][string(pkgValue)] = struct{}{}
	}

	upValue, err := util.JSONMarshal(UpgradeEdge{
		Channel: property.Channel{
			ChannelName: bundle.ChannelName,
		},
		Replaces:  bundle.Replaces,
		Skips:     bundle.Skips,
		SkipRange: bundle.SkipRange,
	})
	if err != nil {
		errors = append(errors, err)
	} else {
		propsList[property.TypeChannel][string(upValue)] = struct{}{}
	}

	for _, p := range bundle.Properties {
		if _, ok := propsList[p.Type]; !ok {
			propsList[p.Type] = map[string]struct{}{}
		}
		propsList[p.Type][p.Value] = struct{}{}
	}

	for pType, pValues := range propsList {
		prop := []interface{}{}
		for pValue := range pValues {
			var v interface{}
			err := util.JSONUnmarshal([]byte(pValue), &v)
			if err != nil {
				errors = append(errors, err)
				continue
			}
			prop = append(prop, v)
		}
		if len(prop) == 0 {
			continue
		}
		if len(prop) > 1 {
			sort.Slice(prop, func(i, j int) bool {
				// enforce some ordering for deterministic properties. Possibly a neater way to do this.
				return fmt.Sprintf("%v", prop[i]) < fmt.Sprintf("%v", prop[j])
			})
		}
		pValue, err := util.JSONMarshal(prop)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		properties[pType] = string(pValue)
	}
	if len(errors) > 0 {
		return properties, fmt.Errorf("failed to parse properties for bundle %s/%s in %s: %v", bundle.GetPackageName(), bundle.GetVersion(), catsrcName, errors2.NewAggregate(errors))
	}
	return properties, nil
}

// Deduplicate entities, aggregate multivalue properties and ensure no conflict for single value properties
func deduplicate(entities []*entitysource.Entity) ([]*entitysource.Entity, error) {
	entityMap := map[entitysource.EntityID]*entitysource.Entity{}
	resultSet := []*entitysource.Entity{}
	errs := map[entitysource.EntityID][]error{}
	for _, e := range entities {
		if _, ok := entityMap[e.ID()]; !ok {
			resultSet = append(resultSet, e)
			entityMap[e.ID()] = e
			continue
		}

		entityErrors := []error{}
		e2 := entityMap[e.ID()]
		for pType, pValue := range e.Properties {
			v2 := []interface{}{}
			err := util.JSONUnmarshal([]byte(e2.Properties[pType]), &v2)
			if err != nil {
				entityErrors = append(entityErrors, fmt.Errorf("unable to merge property values for %s: failed to unmarshal property value %s: %v", pType, e2.Properties[pType], err))
				continue
			}

			v := []interface{}{}
			err = util.JSONUnmarshal([]byte(e.Properties[pType]), &v)
			if err != nil {
				entityErrors = append(entityErrors, fmt.Errorf("unable to merge property values for %s: failed to unmarshal property value %s: %v", pType, e2.Properties[pType], err))
				continue
			}
			switch pType {
			// The operator-registry ListBundles API lists different entries for each channel the referenced bundle is present on.
			// FBC allows a bundle to have different upgrade paths on each channel.
			case property.TypeChannel:
				mergedProp, err := util.JSONMarshal(append(v2, v...))
				if err != nil {
					entityErrors = append(entityErrors, fmt.Errorf("unable to merge property values for %s: failed to unmarshal merged property value: %v", pType, err))
					continue
				}
				e2.Properties[pType] = string(mergedProp)
			// Most properties are single value by default.
			default:
				pValue2, err := e2.GetProperty(pType)
				if err != nil {
					e2.Properties[pType] = pValue
					continue
				}
				if pValue2 != pValue {
					entityErrors = append(entityErrors, fmt.Errorf("property %s has conflicting values : %s , %s ", pType, pValue2, pValue))
					continue
				}
			}
		}
		if len(entityErrors) > 0 {
			if _, ok := errs[e.ID()]; !ok {
				errs[e.ID()] = entityErrors
				continue
			}
			errs[e.ID()] = append(errs[e.ID()], entityErrors...)
		}
	}
	if len(errs) > 0 {
		var errList []error
		for id, entityErrs := range errs {
			errList = append(errList, fmt.Errorf("%s: %v", id, errors2.NewAggregate(entityErrs)))
		}
		return nil, errors2.NewAggregate(errList)
	}
	return resultSet, nil
}
