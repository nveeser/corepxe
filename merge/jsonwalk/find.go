package jsonwalk

import (
	"iter"
	"slices"
	"strings"
)

func Find(obj any, path string) iter.Seq[any] {
	return func(yield func(any) bool) {
		v := &findVisitor{
			path:  ParsePath(path),
			yield: yield,
		}
		Walk(obj, v)
	}
}

func FindOne(obj any, path string) (any, bool) {
	if strings.Contains(path, "*") {
		panic("path may contain multiple values: " + path)
	}
	found := slices.Collect(Find(obj, path))
	switch len(found) {
	case 1:
		return found[0], true
	case 0:
		return nil, false
	default:
		panic("path contains multiple values: " + path)
	}
}

type findVisitor struct {
	path  Path
	yield func(any) bool
}

func (f *findVisitor) Object(n Node, v map[string]any) Result { return f.check(n, v) }
func (f *findVisitor) Sequence(n Node, v []any) Result        { return f.check(n, v) }
func (f *findVisitor) Scalar(n Node, v any) Result            { return f.check(n, v) }

func (f *findVisitor) check(n Node, v any) Result {
	if !n.Path.Match(f.path) {
		return Skip
	}
	if n.Path.Len() == f.path.Len() {
		if !f.yield(v) {
			return Exit
		}
	}
	return Continue
}
