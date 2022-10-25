package olm

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/blang/semver/v4"
	"github.com/tidwall/gjson"

	"github.com/operator-framework/deppy/internal/constraints"
	"github.com/operator-framework/deppy/internal/entitysource"
	"github.com/operator-framework/deppy/internal/sat"
)

const (
	propertyOLMGVK            = "olm.gvk"
	propertyOLMPackageName    = "olm.packageName"
	propertyOLMVersion        = "olm.version"
	propertyOLMChannel        = "olm.channel"
	propertyOLMDefaultChannel = "olm.defaultChannel"
)

var _ constraints.ConstraintGenerator = &requirePackage{}

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
	if err != nil {
		return nil, err
	}
	ids := resultSet.Sort(byChannelAndVersion).CollectIds()
	subject := subject("require", r.packageName, r.versionRange, r.channel)
	return []sat.Variable{
		constraints.NewVariable(subject, sat.Mandatory(), sat.Dependency(toSatIdentifier(ids...)...)),
	}, nil
}

// RequirePackage creates a constraint generator to describe that a package is wanted for installation
func RequirePackage(packageName string, versionRange string, channel string) constraints.ConstraintGenerator {
	return &requirePackage{
		packageName:  packageName,
		versionRange: versionRange,
		channel:      channel,
	}
}

var _ constraints.ConstraintGenerator = &uniqueness{}

type subjectFormatFn func(key string) sat.Identifier

type uniqueness struct {
	subject   subjectFormatFn
	groupByFn entitysource.GroupByFunction
}

func (u *uniqueness) GetVariables(ctx context.Context, querier entitysource.EntityQuerier) ([]sat.Variable, error) {
	resultSet, err := querier.GroupBy(ctx, u.groupByFn)
	if err != nil {
		return nil, err
	}
	variables := make([]sat.Variable, 0, len(resultSet))
	for key, entities := range resultSet {
		ids := toSatIdentifier(entities.Sort(byChannelAndVersion).CollectIds()...)
		variables = append(variables, constraints.NewVariable(u.subject(key), sat.AtMost(1, ids...)))
	}
	return variables, nil
}

// GVKUniqueness generates constraints describing that only a single bundle / gvk can be selected
func GVKUniqueness() constraints.ConstraintGenerator {
	return &uniqueness{
		subject:   uniquenessSubjectFormat,
		groupByFn: gvkGroupFunction,
	}
}

// PackageUniqueness generates constraints describing that only a single bundle / package can be selected
func PackageUniqueness() constraints.ConstraintGenerator {
	return &uniqueness{
		subject:   uniquenessSubjectFormat,
		groupByFn: packageGroupFunction,
	}
}

func uniquenessSubjectFormat(key string) sat.Identifier {
	return sat.IdentifierFromString(fmt.Sprintf("%s uniqueness", key))
}

var _ constraints.ConstraintGenerator = &packageDependency{}

type packageDependency struct {
	subject      sat.Identifier
	packageName  string
	versionRange string
}

func (p *packageDependency) GetVariables(ctx context.Context, querier entitysource.EntityQuerier) ([]sat.Variable, error) {
	entities, err := querier.Filter(ctx, entitysource.And(withPackageName(p.packageName), withinVersion(p.versionRange)))
	if err != nil {
		return nil, err
	}
	ids := toSatIdentifier(entities.Sort(byChannelAndVersion).CollectIds()...)
	return []sat.Variable{constraints.NewVariable(p.subject, sat.Dependency(ids...))}, nil
}

// PackageDependency generates constraints to describe a package's dependency on another package
func PackageDependency(subject sat.Identifier, packageName string, versionRange string) constraints.ConstraintGenerator {
	return &packageDependency{
		subject:      subject,
		packageName:  packageName,
		versionRange: versionRange,
	}
}

var _ constraints.ConstraintGenerator = &gvkDependency{}

type gvkDependency struct {
	subject sat.Identifier
	group   string
	version string
	kind    string
}

func (g *gvkDependency) GetVariables(ctx context.Context, querier entitysource.EntityQuerier) ([]sat.Variable, error) {
	entities, err := querier.Filter(ctx, entitysource.And(withExportsGVK(g.group, g.version, g.kind)))
	if err != nil {
		return nil, err
	}
	ids := toSatIdentifier(entities.Sort(byChannelAndVersion).CollectIds()...)
	return []sat.Variable{constraints.NewVariable(g.subject, sat.Dependency(ids...))}, nil
}

// GVKDependency generates constraints to describe a package's dependency on a gvk
func GVKDependency(subject sat.Identifier, group string, version string, kind string) constraints.ConstraintGenerator {
	return &gvkDependency{
		subject: subject,
		group:   group,
		version: version,
		kind:    kind,
	}
}

func withPackageName(packageName string) entitysource.Predicate {
	return func(entity *entitysource.Entity) bool {
		if pkgName, err := entity.GetProperty(propertyOLMPackageName); err == nil {
			return pkgName == packageName
		}
		return false
	}
}

func withinVersion(semverRange string) entitysource.Predicate {
	return func(entity *entitysource.Entity) bool {
		if v, err := entity.GetProperty(propertyOLMVersion); err == nil {
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
		if c, err := entity.GetProperty(propertyOLMChannel); err == nil {
			return c == channel
		}
		return false
	}
}

func withExportsGVK(group string, version string, kind string) entitysource.Predicate {
	return func(entity *entitysource.Entity) bool {
		if g, err := entity.GetProperty(propertyOLMGVK); err == nil {
			for _, gvk := range gjson.Parse(g).Array() {
				if gjson.Get(gvk.String(), "group").String() == group && gjson.Get(gvk.String(), "version").String() == version && gjson.Get(gvk.String(), "kind").String() == kind {
					return true
				}
			}
		}
		return false
	}
}

func byChannelAndVersion(e1 *entitysource.Entity, e2 *entitysource.Entity) bool {
	e1pkgName, err := e1.GetProperty(propertyOLMPackageName)
	if err != nil {
		return false
	}

	e2pkgName, err := e2.GetProperty(propertyOLMPackageName)
	if err != nil {
		return true
	}

	if e1pkgName != e2pkgName {
		return e1pkgName < e2pkgName
	}

	e1Channel, err := e1.GetProperty(propertyOLMChannel)
	if err != nil {
		return false
	}

	e2Channel, err := e2.GetProperty(propertyOLMChannel)
	if err != nil {
		return true
	}

	if e1Channel != e2Channel {
		e1DefaultChannel, err := e1.GetProperty(propertyOLMDefaultChannel)
		if err != nil {
			return e1Channel < e2Channel
		}
		e2DefaultChannel, err := e2.GetProperty(propertyOLMDefaultChannel)
		if err != nil {
			return e1Channel < e2Channel
		}
		if e1Channel == e1DefaultChannel {
			return true
		} else if e2Channel == e2DefaultChannel {
			return false
		}
		return e1Channel < e2Channel
	}

	e1Version, err := e1.GetProperty(propertyOLMVersion)
	if err != nil {
		return false
	}
	e2Version, err := e2.GetProperty(propertyOLMVersion)
	if err != nil {
		return true
	}

	v1, err := semver.Parse(e1Version)
	if err != nil {
		return false
	}
	v2, err := semver.Parse(e2Version)
	if err != nil {
		return false
	}

	// order version from highest to lowest
	return v1.GT(v2)
}

func gvkGroupFunction(entity *entitysource.Entity) []string {
	if gvks, err := entity.GetProperty(propertyOLMGVK); err == nil {
		gvkArray := gjson.Parse(gvks).Array()
		keys := make([]string, len(gvkArray))
		for i, val := range gvkArray {
			gvk := fmt.Sprintf("%s/%s/%s", val.Get("group"), val.Get("version"), val.Get("kind"))
			keys[i] = gvk
		}
		return keys
	}
	return nil
}

func packageGroupFunction(entity *entitysource.Entity) []string {
	if packageName, err := entity.GetProperty(propertyOLMPackageName); err == nil {
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
