package deppy

type SearchPosition interface {
	Variables() []Variable
	Conflicts() []AppliedConstraint
}

type Tracer interface {
	Trace(p SearchPosition)
}
