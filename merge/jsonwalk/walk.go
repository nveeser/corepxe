package jsonwalk

import (
	"encoding/json"
	"iter"
	"maps"
	"strconv"
)

func WalkJSON(d []byte, visit Visitor) (Result, error) {
	root := make(map[string]any)
	if err := json.Unmarshal(d, &root); err != nil {
		return Exit, err
	}
	return walkObj(Node{}, root, visit), nil
}

func Walk(obj any, visit Visitor) Result {
	return walkObj(Node{}, obj, visit)
}

func walkObj(n Node, value any, visit Visitor) Result {
	switch v := value.(type) {
	case map[string]any:
		r := visit.Object(n, v)
		if r != Continue {
			return r
		}
		next := Node{
			Path:   n.Path,
			Object: v,
		}
		return walkIter(next, maps.All(v), visit)

	case []any:
		r := visit.Sequence(n, v)
		if r != Continue {
			return r
		}
		return walkIter(n, keyedSlice(v), visit)

	default:
		r := visit.Scalar(n, value)
		if r == Exit {
			return r
		}
	}
	return Continue
}

func walkIter(n Node, seq iter.Seq2[string, any], visit Visitor) Result {
	for k, v := range seq {
		nn := Node{
			Path:   n.Path.Child(k),
			Object: n.Object,
			Key:    k,
		}
		r := walkObj(nn, v, visit)
		if r == Exit {
			return r
		}
	}
	return Continue
}

func keyedSlice(s []any) iter.Seq2[string, any] {
	return func(yield func(string, any) bool) {
		for i, sv := range s {
			if !yield(strconv.Itoa(i), sv) {
				return
			}
		}
	}
}
