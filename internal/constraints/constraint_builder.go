package constraints

import (
	"context"

	"github.com/operator-framework/deppy/internal/solver"
	"github.com/operator-framework/deppy/internal/source"
)

type Option func(builder *ConstraintBuilder)

func WithConstraintGenerators(generators []ConstraintGenerator) Option {
	return func(builder *ConstraintBuilder) {
		builder.constraintGenerators = generators
	}
}

type ConstraintGenerator func(ctx context.Context, querier source.EntityQuerier) ([]solver.Variable, error)
type EntityToVariableConverter func(ctx context.Context, entity *source.Entity, querier source.EntityQuerier) (solver.Variable, error)

type ConstraintBuilder struct {
	constraintGenerators    []ConstraintGenerator
	entityQuerier           source.EntityQuerier
	entityVariableConverter EntityToVariableConverter
}

func NewConstraintBuilder(entityQuerier source.EntityQuerier, converter EntityToVariableConverter, options ...Option) *ConstraintBuilder {
	builder := &ConstraintBuilder{
		entityQuerier:           entityQuerier,
		entityVariableConverter: converter,
	}

	for _, option := range options {
		option(builder)
	}

	return builder
}

func (b *ConstraintBuilder) AddConstraintGenerator(generator ConstraintGenerator) {
	b.constraintGenerators = append(b.constraintGenerators, generator)
}

func (b *ConstraintBuilder) Variables(ctx context.Context) ([]solver.Variable, error) {
	variables := make([]solver.Variable, 0)
	for _, constraintGenerator := range b.constraintGenerators {
		vars, err := constraintGenerator(ctx, b.entityQuerier)
		if err != nil {
			return nil, err
		}
		variables = append(variables, vars...)
	}

	err := b.entityQuerier.Iterate(ctx, func(entity *source.Entity) error {
		entityVar, err := b.entityVariableConverter(ctx, entity, b.entityQuerier)
		if err != nil {
			return err
		}
		variables = append(variables, entityVar)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return variables, nil
}
