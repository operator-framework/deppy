package olm

import (
	"context"

	"github.com/operator-framework/deppy/pkg/deppy/input"

	"github.com/operator-framework/deppy/pkg/deppy"
)

var _ input.VariableSource = &requirePackage{}

type requirePackage struct {
	packageName  string
	versionRange string
	channel      string
}

func (r *requirePackage) GetVariables(_ context.Context) ([]deppy.Variable, error) {
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
