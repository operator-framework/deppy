package olm

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/blang/semver/v4"
	"github.com/tidwall/gjson"

	"github.com/operator-framework/deppy/pkg/deppy/input"

	"github.com/operator-framework/deppy/pkg/deppy/constraint"

	"github.com/operator-framework/deppy/pkg/deppy"
)

const (
	PropertyOLMGVK            = "olm.gvk"
	PropertyOLMPackageName    = "olm.packageName"
	PropertyOLMVersion        = "olm.version"
	PropertyOLMChannel        = "olm.channel"
	PropertyOLMDefaultChannel = "olm.defaultChannel"

	propertyNotFound = ""
)

var _ input.VariableSource = &requirePackage{}

type requirePackage struct {
	packageName  string
	versionRange string
	channel      string
}

func (r *requirePackage) GetVariables(ctx context.Context, entitySource input.EntitySource) ([]deppy.Variable, error) {
	resultSet, err := entitySource.Filter(ctx, input.And(
		withPackageName(r.packageName),
		withinVersion(r.versionRange),
		withChannel(r.channel)))
	if err != nil || len(resultSet) == 0 {
		return nil, err
	}
	ids := resultSet.Sort(byChannelAndVersion).CollectIds()
	subject := subject("require", r.packageName, r.versionRange, r.channel)
	return []deppy.Variable{
		input.NewSimpleVariable(subject, constraint.Mandatory(), constraint.Dependency(ids...)),
	}, nil
}

// RequirePackage creates a constraint generator to describe that a package is wanted for installation
func RequirePackage(packageName string, versionRange string, channel string) input.VariableSource {
	return &requirePackage{
		packageName:  packageName,
		versionRange: versionRange,
		channel:      channel,
	}
}

var _ input.VariableSource = &uniqueness{}

type subjectFormatFn func(key string) deppy.Identifier

type uniqueness struct {
	subject   subjectFormatFn
	groupByFn input.GroupByFunction
}

func (u *uniqueness) GetVariables(ctx context.Context, entitySource input.EntitySource) ([]deppy.Variable, error) {
	resultSet, err := entitySource.GroupBy(ctx, u.groupByFn)
	if err != nil || len(resultSet) == 0 {
		return nil, err
	}
	variables := make([]deppy.Variable, 0, len(resultSet))
	for key, entities := range resultSet {
		ids := entities.Sort(byChannelAndVersion).CollectIds()
		variables = append(variables, input.NewSimpleVariable(u.subject(key), constraint.AtMost(1, ids...)))
	}
	return variables, nil
}

// GVKUniqueness generates constraints describing that only a single bundle / gvk can be selected
func GVKUniqueness() input.VariableSource {
	return &uniqueness{
		subject:   uniquenessSubjectFormat,
		groupByFn: gvkGroupFunction,
	}
}

// PackageUniqueness generates constraints describing that only a single bundle / package can be selected
func PackageUniqueness() input.VariableSource {
	return &uniqueness{
		subject:   uniquenessSubjectFormat,
		groupByFn: packageGroupFunction,
	}
}

func uniquenessSubjectFormat(key string) deppy.Identifier {
	return deppy.IdentifierFromString(fmt.Sprintf("%s uniqueness", key))
}

var _ input.VariableSource = &packageDependency{}

type packageDependency struct {
	subject      deppy.Identifier
	packageName  string
	versionRange string
}

func (p *packageDependency) GetVariables(ctx context.Context, entitySource input.EntitySource) ([]deppy.Variable, error) {
	entities, err := entitySource.Filter(ctx, input.And(withPackageName(p.packageName), withinVersion(p.versionRange)))
	if err != nil || len(entities) == 0 {
		return nil, err
	}
	ids := entities.Sort(byChannelAndVersion).CollectIds()
	return []deppy.Variable{input.NewSimpleVariable(p.subject, constraint.Dependency(ids...))}, nil
}

// PackageDependency generates constraints to describe a package's dependency on another package
func PackageDependency(subject deppy.Identifier, packageName string, versionRange string) input.VariableSource {
	return &packageDependency{
		subject:      subject,
		packageName:  packageName,
		versionRange: versionRange,
	}
}

var _ input.VariableSource = &gvkDependency{}

type gvkDependency struct {
	subject deppy.Identifier
	group   string
	version string
	kind    string
}

func (g *gvkDependency) GetVariables(ctx context.Context, entitySource input.EntitySource) ([]deppy.Variable, error) {
	entities, err := entitySource.Filter(ctx, input.And(withExportsGVK(g.group, g.version, g.kind)))
	if err != nil || len(entities) == 0 {
		return nil, err
	}
	ids := entities.Sort(byChannelAndVersion).CollectIds()
	return []deppy.Variable{input.NewSimpleVariable(g.subject, constraint.Dependency(ids...))}, nil
}

// GVKDependency generates constraints to describe a package's dependency on a gvk
func GVKDependency(subject deppy.Identifier, group string, version string, kind string) input.VariableSource {
	return &gvkDependency{
		subject: subject,
		group:   group,
		version: version,
		kind:    kind,
	}
}

func withPackageName(packageName string) input.Predicate {
	return func(entity *input.Entity) bool {
		if pkgName, ok := entity.Properties[PropertyOLMPackageName]; ok {
			return pkgName == packageName
		}
		return false
	}
}

func withinVersion(semverRange string) input.Predicate {
	return func(entity *input.Entity) bool {
		if v, ok := entity.Properties[PropertyOLMVersion]; ok {
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

func withChannel(channel string) input.Predicate {
	return func(entity *input.Entity) bool {
		if channel == "" {
			return true
		}
		if c, ok := entity.Properties[PropertyOLMChannel]; ok {
			return c == channel
		}
		return false
	}
}

func withExportsGVK(group string, version string, kind string) input.Predicate {
	return func(entity *input.Entity) bool {
		if g, ok := entity.Properties[PropertyOLMGVK]; ok {
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
func byChannelAndVersion(e1 *input.Entity, e2 *input.Entity) bool {
	idOrder := e1.Identifier() < e2.Identifier()

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

	// if neither has a version property, sort in Identifier order
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

func gvkGroupFunction(entity *input.Entity) []string {
	if gvks, ok := entity.Properties[PropertyOLMGVK]; ok {
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

func packageGroupFunction(entity *input.Entity) []string {
	if packageName, ok := entity.Properties[PropertyOLMPackageName]; ok {
		return []string{packageName}
	}
	return nil
}

func subject(str ...string) deppy.Identifier {
	return deppy.Identifier(regexp.MustCompile(`\\s`).ReplaceAllString(strings.Join(str, "-"), ""))
}

func getPropertyOrNotFound(entity *input.Entity, propertyName string) string {
	value, ok := entity.Properties[propertyName]
	if !ok {
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
