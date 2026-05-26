package hocon

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/lithammer/dedent"
)

// Document describes HOCON parse tree
type Document struct {
	root propertyList
}

func ParseDocument(s string) (Document, error) {
	parsed, err := Parse("root", []byte(s), Debug(false))
	if err != nil {
		return Document{}, err
	}
	return Document{parsed.(propertyList)}, nil
}

func (d Document) Evaluate() (Object, error) {
	// Construct intrmediate object tree:
	rootValue := d.root.intoValue()
	root := rootValue.(Object)
	// Reduce until no more reductions are possible:
	// 1. Either because everything was successfully resolved / merged / concatenated.
	// 2. Or because there are unresolvable values / undefined references / invalid concatenations.
	for {
		ctx := newContext(root)
		root.reduce(ctx)
		if *ctx.resolved == 0 {
			break
		}
	}
	// Convert intermediate values (errors / merge nodes) into `EvaluationError`s, if any.
	err := evaluationErrorAt([]string{}, root)
	return root, err
}

// Value interface represents a value of an entry described in a HOCON document
type Value interface {
	Type() Type
}

// Type of a HOCON value
type Type int

// Type constants
const (
	ObjectType Type = iota
	ArrayType
	StringType
	DurationType
	IntegerType
	FloatType
	BooleanType
	NullType
	// Internal
	intermediateType
)

// Object represents a complete object described in a HOCON document.
type Object map[string]Value

func (o Object) Type() Type { return ObjectType }

// Object represents a complete object described in a HOCON document.
type Array []Value

func (a Array) Type() Type { return ArrayType }

// String represents a string value
type String string

func (s String) Type() Type { return StringType }

// Duration represents a span of time
type Duration time.Duration

func (s Duration) Type() Type { return DurationType }

type Int int64

func (i Int) Type() Type { return IntegerType }

type Float float64

func (i Float) Type() Type { return FloatType }

type Bool bool

func (i Bool) Type() Type { return BooleanType }

var (
	ErrInvalidPath   = errors.New("property or reference path is invalid")
	ErrUnresolvable  = errors.New("unresolvable reference")
	ErrUndefined     = errors.New("reference points to undefined value")
	ErrMixedPartials = errors.New("concatenation of mixed-type partials")
)

type EvaluationError struct {
	Path  string
	Cause error
	Value Value
}

func (e EvaluationError) Error() string {
	return fmt.Sprintf("config evaluation failed at: %s: %v", e.Path, e.Cause)
}

func (e EvaluationError) Unwrap() error { return e.Cause }

func evaluationErrorAt(path []string, v Value) error {
	var errs []error
	switch v := v.(type) {
	case Object:
		for k, ov := range v {
			if err := evaluationErrorAt(append(path, k), ov); err != nil {
				errs = append(errs, err)
			}
		}
		return errors.Join(errs...)
	case Array:
		var errs []error
		for i, av := range v {
			if err := evaluationErrorAt(append(path, strconv.Itoa(i+1)), av); err != nil {
				errs = append(errs, err)
			}
		}
		return errors.Join(errs...)
	case errorValue:
		return EvaluationError{Path: strings.Join(path, "."), Cause: v.err, Value: v.inner}
	case valueRef, concatOf, mergeOf, missingValue:
		return EvaluationError{Path: strings.Join(path, "."), Cause: ErrUnresolvable, Value: v}
	default:
		return nil
	}
}

// Parser internals

type parseValue any
type parseNode interface {
	intoValue() Value
}

type partials []partial

type partial struct {
	fragment parseNode
}

type propertyList []property
type nodeList []parseValue

type property struct {
	path string
	v    parseValue
}

type valueRef struct {
	path     string
	optional bool
}

type stringLit struct {
	raw  string
	form stringForm
}

type duration string
type percent string

type bytesize struct {
	n    int
	unit string
}

type stringForm int

const (
	unqouted stringForm = iota
	quoted
	triplequoted
)

type fragmentType int

const (
	fragmentObject fragmentType = iota
	fragmentArray
	fragmentString
	// either empty or substitution-only
	fragmentIndeterminate
)

func (p partial) fragType() fragmentType {
	switch p.fragment.(type) {
	case propertyList:
		return fragmentObject
	case nodeList:
		return fragmentArray
	case stringLit:
		return fragmentString
	}
	return fragmentIndeterminate
}

func (ref valueRef) to() []string {
	return strings.Split(ref.path, ".")
}

func (ref valueRef) refersTo(path []string) bool {
	return slices.Equal(ref.to(), path)
}

func (slit stringLit) string() string {
	s := slit.raw
	if slit.form == triplequoted {
		end := len(s)
		if end > 0 && s[end-1] == '~' {
			end -= 1
		}
		if strings.HasPrefix(s, "~\n") {
			return dedent.Dedent(s[2:end])
		}
		if strings.HasPrefix(s, "~\r\n") {
			return dedent.Dedent(s[2:end])
		}
		return s[:end]
	}
	return s
}

// Value representation

func (pl propertyList) intoValue() Value {
	v := Object{}
	for _, p := range pl {
		v.mergeWith(nestValue(strings.Split(p.path, "."), p.v))
	}
	return v
}

func nestValue(path []string, node parseValue) Object {
	out := Object{}
	if len(path) == 1 {
		out[path[0]] = intoValue(node)
	} else {
		out[path[0]] = nestValue(path[1:], node)
	}
	return out
}

func (ps partials) intoValue() Value {
	concat := concatOf{}
	for _, p := range ps {
		concat.inner = append(concat.inner, p.fragment.intoValue())
	}
	return concat.minimize()
}

func (vs nodeList) intoValue() Value {
	v := Array{}
	for _, node := range vs {
		v = append(v, intoValue(node))
	}
	return v
}

func (ref valueRef) intoValue() Value {
	return ref
}

func (sl stringLit) intoValue() Value {
	return String(sl.string())
}

func (p percent) intoValue() Value {
	digits := strings.Split(string(p), "%")
	parsed, _ := strconv.ParseInt(digits[0], 10, 64)
	return Float(float64(parsed) / 100)
}

func (d duration) intoValue() Value {
	parsed, _ := time.ParseDuration(string(d))
	return Duration(parsed)
}

func (bs bytesize) intoValue() Value {
	mult := 1024
	switch bs.unit {
	case "kb", "KB":
		return Int(bs.n * mult)
	case "mb", "MB":
		return Int(bs.n * mult * mult)
	case "gb", "GB":
		return Int(bs.n * mult * mult * mult)
	}
	return Int(bs.n)
}

func intoValue(node any) Value {
	if node == nil {
		return nil
	}
	switch cn := node.(type) {
	case parseNode:
		return cn.intoValue()
	case int64:
		return Int(cn)
	case float64:
		return Float(cn)
	case bool:
		return Bool(cn)
	}
	return nil
}

// Evaluation context

type context struct {
	root     Object
	rootPath []string
	level    int
	resolved *int
}

func newContext(root Object) context {
	resolved := 0
	return context{root, []string{}, -1, &resolved}
}

func (c context) path() []string {
	return c.rootPath[0 : c.level+1]
}

func (c context) drillInto(subpath string) context {
	out := c
	out.rootPath = append(out.rootPath, strings.Split(subpath, ".")...)
	out.level += 1
	return out
}

func (c context) resolve(ref valueRef) (lookupResult, Value) {
	r, v := c.root.lookup(ref.to())
	if (r == valueUndefined || r == valueUnresolvable) && ref.optional {
		r = valueFound
		v = missingValue{}
	}
	if r == valueUndefined && !ref.optional {
		r = valueFound
		v = errorValue{ErrUndefined, ref}
	}
	if r == valueUnresolvable && !ref.optional {
		r = valueFound
		v = errorValue{ErrUnresolvable, ref}
	}
	if r != valueIndeterminate {
		*c.resolved += 1
	}
	return r, v
}

// Trees

type concatOf struct {
	inner []Value
}

type mergeOf struct {
	left  Value
	right Value
}

type missingValue struct{}

type errorValue struct {
	err   error
	inner Value
}

func (v valueRef) Type() Type     { return intermediateType }
func (c concatOf) Type() Type     { return intermediateType }
func (m mergeOf) Type() Type      { return intermediateType }
func (m missingValue) Type() Type { return intermediateType }
func (v errorValue) Type() Type   { return intermediateType }

type lookupResult int

const (
	valueFound lookupResult = iota
	valueUndefined
	valueUnresolvable
	valueIndeterminate
)

func (o Object) lookup(path []string) (lookupResult, Value) {
	if v, ok := o[path[0]]; ok {
		switch v := v.(type) {
		case valueRef, concatOf, mergeOf:
			return valueIndeterminate, v
		case Object:
			if len(path) == 1 {
				return valueFound, v
			}
			return v.lookup(path[1:])
		default:
			if len(path) == 1 {
				return valueFound, v
			}
			return valueUnresolvable, v
		}
	}
	return valueUndefined, nil
}

func (o Object) mergeWith(o2 Object) {
	for k, n2 := range o2 {
		n1, found := o[k]
		if !found || n2 == nil {
			// Set if undefined / overwrite if `null`.
			o[k] = n2
		} else if n2.Type() == ObjectType || n2.Type() == intermediateType {
			// Create a merge node if object or intermediate.
			o[k] = mergeOf{left: n1, right: n2}
		} else {
			// Otherwise if array / string / scalar, overwrite.
			o[k] = n2
		}
	}
}

type concatableValue interface {
	tryConcat(node concatableValue) Value
}

func (o1 Object) tryConcat(node concatableValue) Value {
	if o2, ok := node.(Object); ok {
		o1.mergeWith(o2)
		return o1
	}
	return nil
}

func (a1 Array) tryConcat(node concatableValue) Value {
	if vs, ok := node.(Array); ok {
		return slices.Concat(a1, vs)
	}
	if o2, ok := node.(Object); ok {
		return rewriteIndices(a1, o2)
	}
	return nil
}

func (s1 String) tryConcat(node concatableValue) Value {
	if s2, ok := node.(String); ok {
		return String(s1 + s2)
	}
	return nil
}

func rewriteIndices(a Array, t Object) Array {
	for k, v := range t {
		pos, err := strconv.ParseInt(k, 10, 32)
		idx := int(pos - 1)
		if err != nil || idx < 0 || idx > len(a) {
			return nil
		}
		if idx < len(a) {
			a[idx] = v
		} else {
			a = append(a, v)
		}
	}
	return a
}

func (c concatOf) minimize() Value {
	n := len(c.inner)
	if n == 0 {
		return Object{}
	}
	if n == 1 {
		return c.inner[0]
	}
	i := 1
	v := c.inner[0]
	for i < n {
		next := tryConcat(v, c.inner[i])
		if err, ok := next.(errorValue); ok {
			return err
		}
		if next != nil {
			v = next
			i += 1
		} else {
			break
		}
	}
	if i == 1 {
		return c
	}
	if i == n {
		return v
	}
	return concatOf{slices.Concat([]Value{v}, c.inner[i:])}
}

func tryConcat(v1, v2 Value) Value {
	if c1, ok := v1.(concatableValue); ok {
		if c2, ok := v2.(concatableValue); ok {
			cv := c1.tryConcat(c2)
			if cv == nil {
				return errorValue{ErrMixedPartials, Array{v1, v2}}
			}
			return cv
		}
	}
	return nil
}

func mergeValue(ctx context, n1, n2 Value) Value {
	if c1, ok := n1.(concatOf); ok {
		return mergeConcat(ctx, c1, wrapConcat(n2))
	}
	if c2, ok := n2.(concatOf); ok {
		return mergeConcat(ctx, wrapConcat(n1), c2)
	}
	if t1, ok := n1.(Object); ok {
		if t2, ok := n2.(Object); ok {
			t1.mergeWith(t2)
			return t1
		}
	}
	if a1, ok := n1.(Array); ok {
		if t2, ok := n2.(Object); ok {
			am := rewriteIndices(a1, t2)
			if am != nil {
				return am
			}
		}
	}
	return n2
}

func wrapConcat(node Value) concatOf {
	if cn, ok := node.(concatOf); ok {
		return cn
	}
	return concatOf{[]Value{node}}
}

func mergeConcat(ctx context, c1, c2 concatOf) Value {
	ty1 := c1.innerType()
	ty2 := c2.innerType()
	if ty1 == fragmentObject && (ty2 == ty1 || ty2 == fragmentIndeterminate) {
		// Merge of concatenation of objects with concatenation of objects / substitutions:
		// Preserve `c1` as objects should be merged; drop any self-references from `c2` as they
		// are now pointless.
		merged := c1.inner
		for _, n2 := range c2.inner {
			if ref, ok := n2.(valueRef); ok && ref.refersTo(ctx.path()) {
				continue
			}
			merged = append(merged, n2)
		}
		cn := concatOf{merged}
		return cn.minimize()
	}
	if ty1 == fragmentArray && ty2 == fragmentObject {
		// Merge of concatenation of arrays with concatenation of objects:
		// Preserve both as it could be evaluated as array elements rewrite, see `rewriteIndices`.
		merged := c1.inner
		merged = append(merged, c2.inner...)
		cn := concatOf{merged}
		return cn.minimize()
	}
	if ty1 == ty2 || ty1 == fragmentIndeterminate || ty2 == fragmentIndeterminate {
		// Merge of concatenation of arrays-or-substitutions with arrays-or-substitutions, or same
		// combinations with strings:
		// Should overwrite `c1` unless there's self-ref in `c2`, in this case splice `c1` in place
		// of self-ref.
		merged := []Value{}
		for _, n2 := range c2.inner {
			if ref, ok := n2.(valueRef); ok && ref.refersTo(ctx.path()) {
				merged = append(merged, c1.inner...)
				continue
			}
			merged = append(merged, n2)
		}
		cn := concatOf{merged}
		return cn.minimize()
	}
	// Inconcatenable, arrays with strings / strings with arrays:
	// Overwrite `c1`.
	return c2.minimize()
}

func (c concatOf) innerType() fragmentType {
	ty := fragmentIndeterminate
	for _, v := range c.inner {
		switch v.(type) {
		case Object:
			return fragmentObject
		case Array:
			return fragmentArray
		case String:
			return fragmentString
		}
	}
	return ty
}

func (t Object) reduce(ctx context) {
	for k, v0 := range t {
		subctx := ctx.drillInto(k)
		v := reduceValue(subctx, v0)
		if _, missing := v.(missingValue); !missing {
			t[k] = v
		} else {
			delete(t, k)
		}
	}
}

func reduceValue(ctx context, v Value) Value {
	switch v := v.(type) {
	case Object:
		v.reduce(ctx)
		return v
	case Array:
		out := Array{}
		for _, v0 := range v {
			v1 := reduceValue(ctx, v0)
			if _, missing := v1.(missingValue); !missing {
				out = append(out, v1)
			}
		}
		return out
	case valueRef:
		r, refValue := ctx.resolve(v)
		if r == valueFound {
			return refValue
		}
		return v
	case concatOf:
		return v.reduce(ctx)
	case mergeOf:
		return v.reduce(ctx)
	}
	return v
}

func (m mergeOf) reduce(ctx context) Value {
	left := reduceValue(ctx, m.left)
	right := reduceValue(ctx, m.right)
	// If right side reduces to missing value, preserve left side:
	// This helps to respect semantics of `a = 42, a = ${?missing}`: should evaluate to `42`.
	if _, missing := right.(missingValue); missing {
		return left
	}
	if _, missing := left.(missingValue); missing {
		return right
	}
	*ctx.resolved += 1
	return mergeValue(ctx, left, right)
}

func (c concatOf) reduce(ctx context) Value {
	out := concatOf{}
	refs := 0
	resolves := 0
	for _, node := range c.inner {
		if ref, ok := node.(valueRef); ok {
			refs += 1
			r, refValue := ctx.resolve(ref)
			switch r {
			case valueFound:
				resolves += 1
				// If value is missing, exclude it from the reduced concat:
				if _, missing := refValue.(missingValue); !missing {
					out.inner = append(out.inner, refValue)
				}
			default:
				out.inner = append(out.inner, ref)
			}
		} else {
			out.inner = append(out.inner, node)
		}
	}
	// All references resolved to missing values, reduce whole concat to missing value:
	if refs > 0 && len(out.inner) == 0 {
		return missingValue{}
	}
	return out.minimize()
}
