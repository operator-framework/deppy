package solver

import (
	"fmt"
	"io"
)

type SearchPosition interface {
	Identifiers() []Identifier
	Conflicts() []Constraint
}

type Tracer interface {
	Trace(p SearchPosition)
}

type DefaultTracer struct{}

func (DefaultTracer) Trace(_ SearchPosition) {
}

type LoggingTracer struct {
	Writer io.Writer
}

func (t LoggingTracer) Trace(p SearchPosition) {
	fmt.Fprintf(t.Writer, "---\nAssumptions:\n")
	for _, i := range p.Identifiers() {
		fmt.Fprintf(t.Writer, "- %s\n", i)
	}
	fmt.Fprintf(t.Writer, "Conflict:\n")
	for _, a := range p.Conflicts() {
		fmt.Fprintf(t.Writer, "- %s\n", a)
	}
}
