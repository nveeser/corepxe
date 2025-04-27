package jsonwalk

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

type Printer struct{}

func (p Printer) Object(n Node, v map[string]any) Result {
	fmt.Printf("%s:object [%v]\n", indent(n), slices.Collect(maps.Keys(v)))
	return Continue
}

func (p Printer) Sequence(n Node, v []any) Result {
	fmt.Printf("%s:sequence(size=%d)\n", indent(n), len(v))
	return Continue
}

func (p Printer) Scalar(n Node, v any) Result {
	vs := fmt.Sprintf("%v", v)
	if len(vs) > 100 {
		vs = vs[:100]
	}
	fmt.Printf("%s:scalar %s\n", indent(n), vs)
	return Continue
}

func indent(n Node) string {
	return fmt.Sprintf("%s[%s]", strings.Repeat(" ", n.Path.Len()), n.Path)
}
