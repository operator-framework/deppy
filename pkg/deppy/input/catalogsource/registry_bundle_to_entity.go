package catalogsource

import (
	"fmt"
	"sort"

	"github.com/operator-framework/operator-registry/alpha/property"
	catalogsourceapi "github.com/operator-framework/operator-registry/pkg/api"
	"k8s.io/apimachinery/pkg/util/errors"

	"github.com/operator-framework/deppy/pkg/deppy"
	"github.com/operator-framework/deppy/pkg/deppy/input"
	"github.com/operator-framework/deppy/pkg/lib/util"
)

type UpgradeEdge struct {
	property.Channel
	Replaces  string   `json:"replaces,omitempty"`
	Skips     []string `json:"skips,omitempty"`
	SkipRange string   `json:"skipRange,omitempty"`
}

func entityFromBundle(catsrcID string, bundle *catalogsourceapi.Bundle) (*input.Entity, error) {
	properties := map[string]string{}
	var errs []error

	propsList := map[string]map[string]struct{}{
		property.TypeGVK:         {},
		property.TypeGVKRequired: {},
		property.TypePackage:     {},
		property.TypeChannel:     {},
	}

	for _, prvAPI := range bundle.ProvidedApis {
		apiValue, err := util.JSONMarshal(prvAPI)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		propsList[property.TypeGVK][string(apiValue)] = struct{}{}
	}

	for _, reqAPI := range bundle.RequiredApis {
		apiValue, err := util.JSONMarshal(reqAPI)
		if err != nil {
			errs = append(errs, err)
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
		errs = append(errs, err)
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
		errs = append(errs, err)
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
		var prop []interface{}
		for pValue := range pValues {
			var v interface{}
			err := util.JSONUnmarshal([]byte(pValue), &v)
			if err != nil {
				errs = append(errs, err)
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
			errs = append(errs, err)
			continue
		}
		properties[pType] = string(pValue)
	}
	if len(errs) > 0 {
		return nil, fmt.Errorf("failed to parse properties for bundle %s/%s in %s: %v", bundle.GetPackageName(), bundle.GetVersion(), catsrcID, errors.NewAggregate(errs))
	}

	entityID := deppy.Identifier(fmt.Sprintf("%s/%s/%s", catsrcID, bundle.PackageName, bundle.Version))
	return input.NewEntity(entityID, properties), nil
}

// Deduplicate entities by their ID, aggregate multi-value properties and ensure no conflict for single value properties
func deduplicate(entities []*input.Entity) ([]*input.Entity, error) {
	entityMap := map[deppy.Identifier]*input.Entity{}
	var resultSet []*input.Entity
	errs := map[deppy.Identifier][]error{}
	for _, e := range entities {
		if _, ok := entityMap[e.Identifier()]; !ok {
			resultSet = append(resultSet, e)
			entityMap[e.Identifier()] = e
			continue
		}

		var entityErrors []error
		e2 := entityMap[e.Identifier()]
		for pType, pValue := range e.Properties {
			var v2 []interface{}
			err := util.JSONUnmarshal([]byte(e2.Properties[pType]), &v2)
			if err != nil {
				entityErrors = append(entityErrors, fmt.Errorf("unable to merge property values for %s: failed to unmarshal property value %s: %v", pType, e2.Properties[pType], err))
				continue
			}

			var v []interface{}
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
				pValue2, ok := e2.Properties[pType]
				if !ok {
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
			if _, ok := errs[e.Identifier()]; !ok {
				errs[e.Identifier()] = entityErrors
				continue
			}
			errs[e.Identifier()] = append(errs[e.Identifier()], entityErrors...)
		}
	}
	if len(errs) > 0 {
		var errList []error
		for id, entityErrs := range errs {
			errList = append(errList, fmt.Errorf("%s: %v", id, errors.NewAggregate(entityErrs)))
		}
		return nil, errors.NewAggregate(errList)
	}
	return resultSet, nil
}
