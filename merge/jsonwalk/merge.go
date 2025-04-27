package jsonwalk

import (
	"fmt"
)

var Debug bool = false

func debugf(format string, args ...any) {
	if Debug {
		fmt.Printf(format, args...)
	}
}

type MergeOption func(visitor *mergeVisitor)

func IgnorePath(path string) MergeOption {
	return func(v *mergeVisitor) {
		v.opts.put(ParsePath(path), strategyIgnore)
	}
}

func Replace(path string) MergeOption {
	return func(v *mergeVisitor) {
		v.opts.put(ParsePath(path), strategyReplace)
	}
}

type object = map[string]any

func Merge(dst, src map[string]any, opts ...MergeOption) error {
	v := &mergeVisitor{
		dst:       dst,
		dstByPath: make(map[string]object),
	}
	v.dstByPath[""] = dst
	for _, opt := range opts {
		opt(v)
	}
	Walk(src, v)
	return v.error
}

type mergeVisitor struct {
	dst  object
	opts optionTrie
	// objects by path
	dstByPath map[string]object
	error     error
}

func (m *mergeVisitor) Object(n Node, v object) Result {
	path := n.Path
	if n.Path == nil {
		debugf("%s:object SKIP ROOT\n", indent(n))
		return Continue
	}
	// no parent
	// parent wrong type
	// parent missing key
	// parent replace
	//
	parent, dstObj, err := parentValue[object](m.dstByPath, path)
	if err != nil {
		m.error = err
		return Exit
	}
	switch {
	case dstObj == nil:
		debugf("%s:object replace\n", indent(n))
		dstObj = v
	case m.opts.strategy(n.Path) == strategyIgnore:
		debugf("%s:object ignore\n", indent(n))
		dstObj = v
	}
	m.dstByPath[path.String()] = dstObj
	parent[path.Key()] = dstObj
	return Continue
}

func (m *mergeVisitor) Sequence(n Node, v []any) Result {
	path := n.Path
	strat := m.opts.strategy(path)

	parent, dstSeq, err := parentValue[[]any](m.dstByPath, path)
	if err != nil {
		m.error = err
		return Exit
	}
	switch {
	case strat == strategyIgnore:
		debugf("%s:sequence ignore\n", indent(n))
		return Skip
	case strat == strategyReplace:
		debugf("%s:sequence replace\n", indent(n))
		dstSeq = v
	case dstSeq == nil:
		debugf("%s:sequence add\n", indent(n))
		dstSeq = v
	default:
		debugf("%s:slice append\n", indent(n))
		dstSeq = append(dstSeq, v...)
	}
	parent[path.Key()] = dstSeq
	return Skip
}

func (m *mergeVisitor) Scalar(n Node, v any) Result {
	path := n.Path

	parent, dstAny, err := parentValue[any](m.dstByPath, path)
	if err != nil {
		m.error = err
		return Exit
	}
	switch {
	case m.opts.strategy(path) == strategyIgnore:
		debugf("%s:scalar ignore\n", indent(n))
	case dstAny == nil:
		debugf("%s:scalar set\n", indent(n))
		dstAny = v
	default:
		debugf("%s:scalar replace\n", indent(n))
		dstAny = v
	}
	parent[path.Key()] = dstAny
	return Continue
}

func parentValue[T any](dstCache map[string]object, path Path) (object, T, error) {
	var zero T
	parent, ok := dstCache[path.Parent().String()]
	if !ok {
		return nil, zero, fmt.Errorf("no parent object at %s", path.Parent())
	}
	dstAny, ok := parent[path.Key()]
	if !ok {
		return parent, zero, nil
	}
	dstValue, ok := dstAny.(T)
	if !ok {
		return parent, zero, fmt.Errorf("invalid type at %s: was %T expected %T", path, dstValue, zero)
	}
	return parent, dstValue, nil
}
