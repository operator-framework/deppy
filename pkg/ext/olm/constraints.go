package olm

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/blang/semver/v4"
	"github.com/tidwall/gjson"

	"github.com/operator-framework/deppy/pkg/entitysource"
	"github.com/operator-framework/deppy/pkg/sat"
	"github.com/operator-framework/deppy/pkg/variablesource"
)

const (
	PropertyOLMGVK            = "olm.gvk"
	PropertyOLMPackageName    = "olm.packageName"
	PropertyOLMVersion        = "olm.version"
	PropertyOLMChannel        = "olm.channel"
	PropertyOLMDefaultChannel = "olm.defaultChannel"

	propertyNotFound = ""
)

var _ variablesource.VariableSource = &requirePackage{}

type requirePackage struct {
	packageName  string
	versionRange string
	channel      string
}

func (r *requirePackage) GetVariables(ctx context.Context, querier entitysource.EntityQuerier) ([]sat.Variable, error) {
	resultSet, err := querier.Filter(ctx, entitysource.And(
		withPackageName(r.packageName),
		withinVersion(r.versionRange),
		withChannel(r.channel)))
	if err != nil || len(resultSet) == 0 {
		return nil, err
	}
	ids := resultSet.Sort(byChannelAndVersion).CollectIds()
	subject := subject("require", r.packageName, r.versionRange, r.channel)
	return []sat.Variable{
		variablesource.NewVariable(subject, sat.Mandatory(), sat.Dependency(toSatIdentifier(ids...)...)),
	}, nil
}

// RequirePackage creates a constraint generator to describe that a package is wanted for installation
func RequirePackage(packageName string, versionRange string, channel string) variablesource.VariableSource {
	return &requirePackage{
		packageName:  packageName,
		versionRange: versionRange,
		channel:      channel,
	}
}

var _ variablesource.VariableSource = &uniqueness{}

type subjectFormatFn func(key string) sat.Identifier

type uniqueness struct {
	subject   subjectFormatFn
	groupByFn entitysource.GroupByFunction
}

func (u *uniqueness) GetVariables(ctx context.Context, querier entitysource.EntityQuerier) ([]sat.Variable, error) {
	resultSet, err := querier.GroupBy(ctx, u.groupByFn)
	if err != nil || len(resultSet) == 0 {
		return nil, err
	}
	variables := make([]sat.Variable, 0, len(resultSet))
	for key, entities := range resultSet {
		ids := toSatIdentifier(entities.Sort(byChannelAndVersion).CollectIds()...)
		variables = append(variables, variablesource.NewVariable(u.subject(key), sat.AtMost(1, ids...)))
	}
	return variables, nil
}

// GVKUniqueness generates constraints describing that only a single bundle / gvk can be selected
func GVKUniqueness() variablesource.VariableSource {
	return &uniqueness{
		subject:   uniquenessSubjectFormat,
		groupByFn: gvkGroupFunction,
	}
}

// PackageUniqueness generates constraints describing that only a single bundle / package can be selected
func PackageUniqueness() variablesource.VariableSource {
	return &uniqueness{
		subject:   uniquenessSubjectFormat,
		groupByFn: packageGroupFunction,
	}
}

func uniquenessSubjectFormat(key string) sat.Identifier {
	return sat.IdentifierFromString(fmt.Sprintf("%s uniqueness", key))
}

var _ variablesource.VariableSource = &packageDependency{}

type packageDependency struct {
	subject      sat.Identifier
	packageName  string
	versionRange string
}

func (p *packageDependency) GetVariables(ctx context.Context, querier entitysource.EntityQuerier) ([]sat.Variable, error) {
	entities, err := querier.Filter(ctx, entitysource.And(withPackageName(p.packageName), withinVersion(p.versionRange)))
	if err != nil || len(entities) == 0 {
		return nil, err
	}
	ids := toSatIdentifier(entities.Sort(byChannelAndVersion).CollectIds()...)
	return []sat.Variable{variablesource.NewVariable(p.subject, sat.Dependency(ids...))}, nil
}

// PackageDependency generates constraints to describe a package's dependency on another package
func PackageDependency(subject sat.Identifier, packageName string, versionRange string) variablesource.VariableSource {
	return &packageDependency{
		subject:      subject,
		packageName:  packageName,
		versionRange: versionRange,
	}
}

var _ variablesource.VariableSource = &gvkDependency{}

type gvkDependency struct {
	subject sat.Identifier
	group   string
	version string
	kind    string
}

func (g *gvkDependency) GetVariables(ctx context.Context, querier entitysource.EntityQuerier) ([]sat.Variable, error) {
	entities, err := querier.Filter(ctx, entitysource.And(withExportsGVK(g.group, g.version, g.kind)))
	if err != nil || len(entities) == 0 {
		return nil, err
	}
	ids := toSatIdentifier(entities.Sort(byChannelAndVersion).CollectIds()...)
	return []sat.Variable{variablesource.NewVariable(g.subject, sat.Dependency(ids...))}, nil
}

// GVKDependency generates constraints to describe a package's dependency on a gvk
func GVKDependency(subject sat.Identifier, group string, version string, kind string) variablesource.VariableSource {
	return &gvkDependency{
		subject: subject,
		group:   group,
		version: version,
		kind:    kind,
	}
}

func withPackageName(packageName string) entitysource.Predicate {
	return func(entity *entitysource.Entity) bool {
		if pkgName, err := entity.GetProperty(PropertyOLMPackageName); err == nil {
			return pkgName == packageName
		}
		return false
	}
}

func withinVersion(semverRange string) entitysource.Predicate {
	return func(entity *entitysource.Entity) bool {
		if v, err := entity.GetProperty(PropertyOLMVersion); err == nil {
			vrange, err := semver.ParseRange(semverRange)
			if err != nil {
				return false
			}
			version, err := semver.Parse(v)
			if err != nil {
				return false
			}
			return vrange(version)
		}
		return false
	}
}

func withChannel(channel string) entitysource.Predicate {
	return func(entity *entitysource.Entity) bool {
		if channel == "" {
			return true
		}
		if c, err := entity.GetProperty(PropertyOLMChannel); err == nil {
			return c == channel
		}
		return false
	}
}

func withExportsGVK(group string, version string, kind string) entitysource.Predicate {
	return func(entity *entitysource.Entity) bool {
		if g, err := entity.GetProperty(PropertyOLMGVK); err == nil {
			for _, gvk := range gjson.Parse(g).Array() {
				if gjson.Get(gvk.String(), "group").String() == group && gjson.Get(gvk.String(), "version").String() == version && gjson.Get(gvk.String(), "kind").String() == kind {
					return true
				}
			}
		}
		return false
	}
}

// byChannelAndVersion is an entity sort function that orders the entities in
// package, channel (default channel at the head), and inverse version (higher versions on top)
// if a property does not exist for one of the entities, the one missing the property is pushed down
// if both entities are missing the same property they are ordered by id
func byChannelAndVersion(e1 *entitysource.Entity, e2 *entitysource.Entity) bool {
	idOrder := e1.ID() < e2.ID()

	// first sort package lexical order
	pkgOrder := compareProperty(getPropertyOrNotFound(e1, PropertyOLMPackageName), getPropertyOrNotFound(e2, PropertyOLMPackageName))
	if pkgOrder != 0 {
		return pkgOrder < 0
	}

	// then sort by channel order with default channel at the start and all other channels in lexical order
	e1DefaultChannel := getPropertyOrNotFound(e1, PropertyOLMDefaultChannel)
	e2DefaultChannel := getPropertyOrNotFound(e2, PropertyOLMDefaultChannel)

	e1Channel := getPropertyOrNotFound(e1, PropertyOLMChannel)
	e2Channel := getPropertyOrNotFound(e2, PropertyOLMChannel)
	channelOrder := compareProperty(e1Channel, e2Channel)

	// if both entities are from different channels
	if channelOrder != 0 {
		e1InDefaultChannel := e1Channel == e1DefaultChannel && e1Channel != propertyNotFound
		e2InDefaultChannel := e2Channel == e2DefaultChannel && e2Channel != propertyNotFound

		// if one of them is in the default channel, promote it
		// if both of them are in their default channels, stay with lexical channel order
		if e1InDefaultChannel || e2InDefaultChannel && !(e1InDefaultChannel && e2InDefaultChannel) {
			return e1InDefaultChannel
		}

		// otherwise sort by channel lexical order
		return channelOrder < 0
	}

	// if package and channel are the same, compare by version
	e1Version := getPropertyOrNotFound(e1, PropertyOLMVersion)
	e2Version := getPropertyOrNotFound(e2, PropertyOLMVersion)

	// if neither has a version property, sort in ID order
	if e1Version == propertyNotFound && e2Version == propertyNotFound {
		return idOrder
	}

	// if one of the version is not found, not found is higher than found
	if e1Version == propertyNotFound || e2Version == propertyNotFound {
		return e1Version > e2Version
	}

	// if one or both of the versions cannot be parsed, return id order
	v1, err := semver.Parse(e1Version)
	if err != nil {
		return idOrder
	}
	v2, err := semver.Parse(e2Version)
	if err != nil {
		return idOrder
	}

	// finally, order version from highest to lowest (favor the latest release)
	return v1.GT(v2)
}

func gvkGroupFunction(entity *entitysource.Entity) []string {
	if gvks, err := entity.GetProperty(PropertyOLMGVK); err == nil {
		gvkArray := gjson.Parse(gvks).Array()
		keys := make([]string, 0, len(gvkArray))
		for _, val := range gvkArray {
			var group = val.Get("group")
			var version = val.Get("version")
			var kind = val.Get("kind")
			if group.String() != "" && version.String() != "" && kind.String() != "" {
				gvk := fmt.Sprintf("%s/%s/%s", group, version, kind)
				keys = append(keys, gvk)
			}
		}
		return keys
	}
	return nil
}

func packageGroupFunction(entity *entitysource.Entity) []string {
	if packageName, err := entity.GetProperty(PropertyOLMPackageName); err == nil {
		return []string{packageName}
	}
	return nil
}

func subject(str ...string) sat.Identifier {
	return sat.Identifier(regexp.MustCompile(`\\s`).ReplaceAllString(strings.Join(str, "-"), ""))
}

func toSatIdentifier(ids ...entitysource.EntityID) []sat.Identifier {
	satIds := make([]sat.Identifier, len(ids))
	for i := range ids {
		satIds[i] = sat.Identifier(ids[i])
	}
	return satIds
}

func getPropertyOrNotFound(entity *entitysource.Entity, propertyName string) string {
	value, err := entity.GetProperty(propertyName)
	if err != nil {
		return propertyNotFound
	}
	return value
}

// compareProperty compares two entity property values. It works as strings.Compare
// except when one of the properties is not found (""). It computes not found properties as higher than found.
// If p1 is not found, it returns 1, if p2 is not found it returns -1
func compareProperty(p1, p2 string) int {
	if p1 == p2 {
		return 0
	}
	if p1 == propertyNotFound {
		return 1
	}
	if p2 == propertyNotFound {
		return -1
	}
	return strings.Compare(p1, p2)
}
