package main

import (
	"context"
	"fmt"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/deppy/internal/constraints"
	"github.com/operator-framework/deppy/internal/olm"
	"github.com/operator-framework/deppy/internal/olm/source"
	"github.com/operator-framework/deppy/internal/solver"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
}

func main() {
	/*
		REQUIREMENTS:
		 - fresh kind cluster
		 - operator-sdk olm install
		 - update /etc/hosts to point operatorhubio-catalog.olm.svc to localhost
		 - kubectl port-forward -n olm pod/operatorhubio-catalog-xxxxx 50051:50051 (check your pod's exact name)
	*/
	kubeClient, err := client.New(ctrl.GetConfigOrDie(), client.Options{
		Scheme: scheme,
	})
	if err != nil {
		fmt.Println(err)
		return
	}

	// create deppy source and grab all the bundles
	deppySource := source.NewCatalogSourceDeppySource(kubeClient, "olm", "operatorhubio-catalog")
	err = deppySource.Sync(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}

	// create a constraint builder (or variable builder...)
	constraintBuilder := constraints.NewConstraintBuilder(deppySource, olm.EntityToVariable,
		constraints.WithConstraintGenerators([]constraints.ConstraintGenerator{
			olm.RequirePackage("ack-sfn-controller", "> 1.0.0", ""),
			olm.PackageUniqueness(),
			olm.GVKUniqueness(),
		}))

	// imagine a nice interface to create the solver which takes the sources and constraint builder
	variables, err := constraintBuilder.Variables(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}
	operatorSolver, err := solver.New(solver.WithInput(variables))
	if err != nil {
		fmt.Println(err)
		return
	}

	selection, err := operatorSolver.Solve(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("selection")
	for _, item := range selection {
		fmt.Println(item.Identifier())
	}
}
