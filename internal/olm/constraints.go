package olm

import (
	"context"
	"fmt"

	"github.com/blang/semver/v4"
	"github.com/operator-framework/deppy/internal/constraints"
	"github.com/operator-framework/deppy/internal/solver"
	"github.com/operator-framework/deppy/internal/source"
	"github.com/tidwall/gjson"
)

const (
	propertyOLMVersion         = "olm.version"
	propertyOLMPackageName     = "olm.packageName"
	propertyOLMChannel         = "olm.channel"
	propertyOLMDefaultChannel  = "olm.defaultChannel"
	propertyOLMGVK             = "olm.gvk"
	propertyOLMPackageRequired = "olm.package.required"
	propertyOLMGVKRequired     = "olm.gvk.required"
)

var _ solver.Variable = &OLMVariable{}

type OLMVariable struct {
	id          solver.Identifier
	constraints []solver.Constraint
}

func (o *OLMVariable) Identifier() solver.Identifier {
	return o.id
}

func (o *OLMVariable) Constraints() []solver.Constraint {
	return o.constraints
}

func EntityToVariable(ctx context.Context, entity *source.Entity, querier source.EntityQuerier) (solver.Variable, error) {
	solverVar := &OLMVariable{
		id: solver.Identifier(entity.ID()),
	}
	constrs := make([]solver.Constraint, 0)

	// collect package dependencies
	if pkgRequirementsJson, err := entity.GetProperty(propertyOLMPackageRequired); err == nil {
		pkgRequirements := gjson.Parse(pkgRequirementsJson).Array()
		for _, pkgReqs := range pkgRequirements {
			packageName := pkgReqs.Get("packageName").String()
			versionRange := pkgReqs.Get("versionRange").String()
			resultSet, err := querier.Filter(ctx, source.And(withPackageName(packageName), withinVersion(versionRange)))
			if err != nil {
				return nil, err
			}
			resultSet = resultSet.Sort(byChannelAndVersion)
			constrs = append(constrs, solver.Dependency(toSolverIdentifier(resultSet.CollectIds())...))
		}
	}

	// collect gvk dependencies
	if gvkRequirementsJson, err := entity.GetProperty(propertyOLMGVKRequired); err == nil {
		gvkRequirements := gjson.Parse(gvkRequirementsJson).Array()
		for _, gvkReqs := range gvkRequirements {
			group := gvkReqs.Get("group").String()
			version := gvkReqs.Get("version").String()
			kind := gvkReqs.Get("kind").String()

			resultSet, err := querier.Filter(ctx, source.And(withExportsGVK(group, version, kind)))
			if err != nil {
				return nil, err
			}
			resultSet = resultSet.Sort(byChannelAndVersion)
			constrs = append(constrs, solver.Dependency(toSolverIdentifier(resultSet.CollectIds())...))
		}
	}

	solverVar.constraints = constrs
	return solverVar, nil
}

func RequirePackage(packageName string, versionRange string, channel string) constraints.ConstraintGenerator {
	return func(ctx context.Context, querier source.EntityQuerier) ([]solver.Variable, error) {
		vars := make([]solver.Variable, 0)
		resultSet, err := querier.Filter(ctx, withPackageName(packageName)) // source.And(withPackageName(packageName), withinVersion(versionRange), withChannel(channel)))
		if err != nil {
			return nil, err
		}
		entities := resultSet.Sort(byChannelAndVersion)
		vars = append(vars, &OLMVariable{
			id: solver.Identifier(fmt.Sprintf("package %s required at %s", packageName, versionRange)),
			constraints: []solver.Constraint{
				solver.Mandatory(),
				solver.Dependency(toSolverIdentifier(entities.CollectIds())...),
			},
		})

		return vars, nil
	}
}

func GVKUniqueness() constraints.ConstraintGenerator {
	return func(ctx context.Context, querier source.EntityQuerier) ([]solver.Variable, error) {
		resultMap, err := querier.GroupBy(ctx, func(e1 *source.Entity) []string {
			gvks, err := e1.GetProperty(propertyOLMGVK)
			if err != nil {
				return nil
			}
			gvkArray := gjson.Parse(gvks).Array()
			out := make([]string, 0)
			for _, val := range gvkArray {
				out = append(out, fmt.Sprintf("%s/%s/%s", val.Get("group"), val.Get("version"), val.Get("kind")))
			}
			return out
		})
		if err != nil {
			return nil, err
		}
		resultMap = resultMap.Sort(byVersionIncreasing)

		vars := make([]solver.Variable, 0, len(resultMap))
		for key, entities := range resultMap {
			v := &OLMVariable{
				id: solver.Identifier(fmt.Sprintf("%s uniqueness", key)),
				constraints: []solver.Constraint{
					solver.AtMost(1, toSolverIdentifier(entities.CollectIds())...),
				},
			}
			vars = append(vars, v)
		}

		return vars, nil
	}
}

func PackageUniqueness() constraints.ConstraintGenerator {
	return func(ctx context.Context, querier source.EntityQuerier) ([]solver.Variable, error) {
		resultMap, err := querier.GroupBy(ctx, func(e1 *source.Entity) []string {
			packageName, err := e1.GetProperty(propertyOLMPackageName)
			if err != nil {
				return nil
			}
			return []string{packageName}
		})
		if err != nil {
			return nil, err
		}
		resultMap = resultMap.Sort(byVersionIncreasing)

		vars := make([]solver.Variable, 0, len(resultMap))
		for key, entities := range resultMap {
			v := &OLMVariable{
				id: solver.Identifier(fmt.Sprintf("%s uniqueness", key)),
				constraints: []solver.Constraint{
					solver.AtMost(1, toSolverIdentifier(entities.CollectIds())...),
				},
			}
			vars = append(vars, v)
		}

		return vars, nil
	}
}

func withPackageName(packageName string) source.Predicate {
	return func(entity *source.Entity) bool {
		if pkgName, err := entity.GetProperty(propertyOLMPackageName); err == nil {
			return pkgName == packageName
		}
		return false
	}
}

func withinVersion(semverRange string) source.Predicate {
	return func(entity *source.Entity) bool {
		if v, err := entity.GetProperty(propertyOLMVersion); err == nil {
			vrange := semver.MustParseRange(semverRange)
			version := semver.MustParse(v)
			return vrange(version)
		}
		return false
	}
}

func withChannel(channel string) source.Predicate {
	return func(entity *source.Entity) bool {
		if channel == "" {
			return true
		}
		if c, err := entity.GetProperty(propertyOLMChannel); err == nil {
			return c == channel
		}
		return false
	}
}

func withExportsGVK(group string, version string, kind string) source.Predicate {
	return func(entity *source.Entity) bool {
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

func byChannelAndVersion(e1 *source.Entity, e2 *source.Entity) bool {
	e1PackageName, err := e1.GetProperty(propertyOLMPackageName)
	if err != nil {
		return false
	}

	e2PackageName, err := e2.GetProperty(propertyOLMPackageName)
	if err != nil {
		return false
	}

	if e1PackageName != e2PackageName {
		return e1PackageName < e2PackageName
	}

	e1Channel, err := e1.GetProperty(propertyOLMChannel)
	if err != nil {
		return false
	}
	e2Channel, err := e2.GetProperty(propertyOLMChannel)
	if err != nil {
		return false
	}

	if e1Channel != e2Channel {
		e1DefaultChannel, err := e1.GetProperty(propertyOLMDefaultChannel)
		if err != nil {
			return false
		}
		e2DefaultChannel, err := e2.GetProperty(propertyOLMDefaultChannel)
		if err != nil {
			return false
		}

		if e1Channel == e1DefaultChannel {
			return true
		}
		if e2Channel == e2DefaultChannel {
			return false
		}
		return e1Channel < e2Channel
	}

	v1, err := e1.GetProperty(propertyOLMVersion)
	if err != nil {
		return false
	}
	v2, err := e2.GetProperty(propertyOLMVersion)
	if err != nil {
		return false
	}

	return semver.MustParse(v1).GT(semver.MustParse(v2))
}

func byVersionIncreasing(e1 *source.Entity, e2 *source.Entity) bool {
	v1, err := e1.GetProperty(propertyOLMVersion)
	if err != nil {
		return false
	}

	v2, err := e1.GetProperty(propertyOLMVersion)
	if err != nil {
		return false
	}
	return semver.MustParse(v1).LT(semver.MustParse(v2))
}

func toSolverIdentifier(ids []source.EntityID) []solver.Identifier {
	out := make([]solver.Identifier, len(ids))
	for i, _ := range ids {
		out[i] = solver.Identifier(ids[i])
	}
	return out
}
