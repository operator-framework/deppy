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
	Version   string   `json:"version,omitempty"`
}

type DefaultChannel struct {
	DefaultChannel string `json:"defaultchannel"`
}

type BundleSource struct {
	// possibly extend with support for other types of sources
	Path string `json:"bundlepath,omitempty"`
}

const TypeDefaultChannel = "olm.package.defaultchannel"
const TypeBundleSource = "olm.bundle.path"

func entityFromBundle(catsrcID string, pkg *catalogsourceapi.Package, bundle *catalogsourceapi.Bundle) (*input.Entity, error) {
	properties := map[string]string{}
	var errs []error

	// All properties are json encoded lists of values, regarless of property type.
	// They can all be handled by unmarshalling into an (interface{}) - deppy does not need to know how to handle different types of properties.
	propsList := map[string]map[string]struct{}{
		property.TypeGVK:             {},
		property.TypeGVKRequired:     {},
		property.TypePackage:         {},
		property.TypePackageRequired: {},
		property.TypeChannel:         {},
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
		Version:   bundle.Version,
	})
	if err != nil {
		errs = append(errs, err)
	} else {
		propsList[property.TypeChannel][string(upValue)] = struct{}{}
	}

	defaultValue, err := util.JSONMarshal(DefaultChannel{
		DefaultChannel: pkg.DefaultChannelName,
	})
	if err != nil {
		errs = append(errs, err)
	} else {
		propsList[TypeDefaultChannel] = map[string]struct{}{string(defaultValue): {}}
	}

	sourceValue, err := util.JSONMarshal(BundleSource{
		Path: bundle.BundlePath,
	})
	if err != nil {
		errs = append(errs, err)
	} else {
		propsList[TypeBundleSource] = map[string]struct{}{string(sourceValue): {}}
	}

	for _, p := range bundle.Properties {
		if _, ok := propsList[p.Type]; !ok {
			propsList[p.Type] = map[string]struct{}{}
		}
		if p.Type == property.TypePackage {
			// override the inferred package if explicitly specified in bundle properties
			propsList[p.Type] = map[string]struct{}{p.Value: {}}
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

	entityID := deppy.Identifier(fmt.Sprintf("%s/%s/%s/%s", catsrcID, bundle.PackageName, bundle.ChannelName, bundle.Version))
	return input.NewEntity(entityID, properties), nil
}
