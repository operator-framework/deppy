package constraints

type Identifier string

func (id Identifier) String() string {
	return string(id)
}

// Constraint implementations limit the circumstances under which a
// particular Variable can appear in a solution.
type Constraint struct {
	ConstraintType string
	Properties     map[string]interface{}
	Anchor         bool
	Order          []Identifier
}

// Variable values are the basic unit of problems and solutions
// understood by this package.
type IVariable interface {
	// Identifier returns the Identifier that uniquely identifies
	// this Variable among all other Variables in a given
	// problem.
	Identifier() Identifier
	// Constraints returns the set of constraints that apply to
	// this Variable.
	Constraints() []Constraint
}

// AppliedConstraint values compose a single Constraint with the
// Variable it applies to.
type AppliedConstraint struct {
	Variable   IVariable
	Constraint Constraint
}

// String implements fmt.Stringer and returns a human-readable message
// representing the receiver.
func (a AppliedConstraint) String() string {
	return a.Variable.Identifier().String()
}

var _ IVariable = &Variable{}

// Variable is a simple implementation of sat.Variable
type Variable struct {
	id          Identifier
	constraints []Constraint
}

func (v *Variable) Identifier() Identifier {
	return v.id
}

func (v *Variable) Constraints() []Constraint {
	return v.constraints
}

func (v *Variable) AddConstraint(constraint ...Constraint) {
	v.constraints = append(v.constraints, constraint...)
}

func NewVariable(id Identifier, constraints ...Constraint) *Variable {
	return &Variable{
		id:          id,
		constraints: constraints,
	}
}
