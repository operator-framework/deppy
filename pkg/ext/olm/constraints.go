package olm

import (
	"context"
	"strings"

	"github.com/operator-framework/deppy/pkg/deppy/input"

	"github.com/operator-framework/deppy/pkg/deppy"
)

// Question:
// These are not needed since while constructing the variables, its upto the users
// to sort, group by etc while providing it to the solver. Is this understanding right?
// Which means we will do the filtering based on constraints before constructing variables.
// Maybe we should create helpers for these separately keeping "bundle" as the domain.

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

func (r *requirePackage) GetVariables(ctx context.Context) ([]deppy.Variable, error) {
	return nil, nil
}

// RequirePackage creates a constraint generator to describe that a package is wanted for installation
func RequirePackage(packageName string, versionRange string, channel string) input.VariableSource {
	return &requirePackage{
		packageName:  packageName,
		versionRange: versionRange,
		channel:      channel,
	}
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
