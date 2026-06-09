package hocon

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/lithammer/dedent"
)

// Document describes HOCON parse tree
type Document struct {
	root           propertyList
	includeBaseDir string
	includeStack   []string
}

func ParseDocument(s string, includeDir ...string) (Document, error) {
	baseDir := ""
	if len(includeDir) > 0 {
		baseDir = includeDir[0]
	}
	parsed, err := Parse("<input>", []byte(s))
	if err != nil {
		return Document{}, err
	}
	doc := Document{
		root:           parsed.(propertyList),
		includeBaseDir: baseDir,
		includeStack:   []string{},
	}
	return doc, nil
}

func ParseDocumentFile(filename string) (Document, error) {
	absPath, err := filepath.Abs(filename)
	if err != nil {
		return Document{}, err
	}
	absPath = filepath.Clean(absPath)
	parsed, err := ParseFile(absPath)
	if err != nil {
		return Document{}, err
	}
	doc := Document{
		root:           parsed.(propertyList),
		includeBaseDir: filepath.Dir(absPath),
		includeStack:   []string{absPath},
	}
	return doc, nil
}

func (d Document) Evaluate() (Object, error) {
	// Construct intrmediate object tree:
	rootValue := d.root.intoValue()
	// Reduce until no more reductions are possible:
	// 1. Either because everything was successfully resolved / merged / concatenated.
	// 2. Or because there are unresolvable values / undefined references / invalid concatenations.
	for {
		ctx := newContext(rootValue, d.includeBaseDir, d.includeStack)
		rootValue = reduceValue(ctx, rootValue)
		if *ctx.resolved == 0 {
			break
		}
	}
	// Convert intermediate values (errors / merge nodes) into `EvaluationError`s, if any.
	err := evaluationErrorAt([]string{}, rootValue)
	root, _ := rootValue.(Object)
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
	IntegerType
	FloatType
	BooleanType
	NullType
	// Internal
	intermediateType
)

type StringContentType int

const (
	StringGeneric StringContentType = iota
	StringDuration
	StringBytesize
	StringPercent
)

// Object represents a complete object described in a HOCON document.
type Object map[string]Value

func (o Object) Type() Type { return ObjectType }

func (o Object) Lookup(path string) (Value, bool) {
	p := asPath(path)
	if len(p) == 0 {
		return o, true
	}
	result, v := o.lookup(p)
	if result != valueFound {
		return nil, false
	}
	return v, true
}

func (o Object) DeepCopy() Object {
	return o.deepCopy().(Object)
}

// Object represents a complete object described in a HOCON document.
type Array []Value

func (a Array) DeepCopy() Array {
	return a.deepCopy().(Array)
}
func (a Array) Type() Type { return ArrayType }

type StringValue interface {
	ContentType() StringContentType
	String() string
}

// String represents a string value
type String string
type DurationString string
type BytesizeString string
type PercentString string

func (s String) Type() Type         { return StringType }
func (s DurationString) Type() Type { return StringType }
func (s BytesizeString) Type() Type { return StringType }
func (s PercentString) Type() Type  { return StringType }

func (s String) ContentType() StringContentType         { return StringGeneric }
func (s DurationString) ContentType() StringContentType { return StringDuration }
func (s BytesizeString) ContentType() StringContentType { return StringBytesize }
func (s PercentString) ContentType() StringContentType  { return StringPercent }

func (s String) String() string         { return string(s) }
func (s DurationString) String() string { return string(s) }
func (s BytesizeString) String() string { return string(s) }
func (s PercentString) String() string  { return string(s) }

func (ps PercentString) AsFloat() float64 {
	digits := strings.Split(string(ps), "%")
	parsed, _ := strconv.ParseInt(digits[0], 10, 64)
	return float64(parsed) / 100
}

func (ds DurationString) AsDuration() time.Duration {
	s := string(ds)
	if strings.HasSuffix(s, "d") || strings.HasSuffix(s, "D") {
		days, _ := strconv.ParseInt(s[:len(s)-1], 10, 64)
		return time.Duration(days) * 24 * time.Hour
	}
	duration, _ := time.ParseDuration(strings.ToLower(s))
	return duration
}

func (bs BytesizeString) AsInteger() int64 {
	s := strings.ToLower(string(bs))
	l := len(s)
	mult := 1
	suffix := 0
	switch {
	case l == 0:
		return 0
	case s[l-1] != 'b':
		break
	case l == 1:
		return 1
	case s[l-2] == 'k':
		suffix = 2
		mult = 1024
	case s[l-2] == 'm':
		suffix = 2
		mult = 1024 * 1024
	case s[l-2] == 'g':
		suffix = 2
		mult = 1024 * 1024 * 1024
	default:
		suffix = 1
	}
	num := int64(1)
	if l-suffix > 0 {
		num, _ = strconv.ParseInt(s[:l-suffix], 10, 64)
	}
	return num * int64(mult)
}

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
	ErrBadArrayIndex = errors.New("out of bounds array index update")
	ErrIncludeCycle  = errors.New("include cycle")
	ErrIncludeFailed = errors.New("could not process include")
)

type IncludeError struct {
	Path string
	Err  error
}

func (e IncludeError) Error() string {
	return fmt.Sprintf("%v %q: %v", ErrIncludeFailed, e.Path, e.Err)
}

func (e IncludeError) Unwrap() error {
	return errors.Join(ErrIncludeFailed, e.Err)
}

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
	case valueRef, concatOf, mergeOf, includeOf, missingValue:
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

type propertyList []propertyListEntry

type nodeList []parseValue

// Either property or includeNode:
type propertyListEntry any

type property struct {
	path string
	v    parseValue
}

type includeOf struct {
	path     string
	required bool
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
type bytesize string

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

type path []string

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

func (ref valueRef) to() path {
	return asPath(ref.path)
}

func (ref valueRef) refersTo(path path) bool {
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

func asPath(pathString string) path {
	return slices.DeleteFunc(
		strings.Split(pathString, "."),
		func(comp string) bool { return comp == "" },
	)
}

func appendPath(prefix path, suffix path) path {
	out := slices.Clone(prefix)
	return append(out, suffix...)
}

// Value representation

func (pl propertyList) intoValue() Value {
	current := Object{}
	concat := concatOf{}
	for _, entry := range pl {
		switch entry := entry.(type) {
		case property:
			current.mergeWith(nestValue(asPath(entry.path), entry.v))
		case includeOf:
			if len(current) > 0 {
				concat.inner = append(concat.inner, current)
				current = Object{}
			}
			concat.inner = append(concat.inner, entry)
		}
	}
	if len(concat.inner) == 0 {
		return current
	}
	if len(current) > 0 {
		concat.inner = append(concat.inner, current)
	}
	return concat
}

func nestValue(path path, node parseValue) Object {
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
	return PercentString(p)
}

func (d duration) intoValue() Value {
	return DurationString(d)
}

func (bs bytesize) intoValue() Value {
	return BytesizeString(bs)
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

// deepValue marks a Value that needs to be deeply copied during substitution.
type deepValue interface {
	deepCopy() Value
}

func (o Object) deepCopy() Value {
	out := Object{}
	for k, v := range o {
		out[k] = deepCopy(v)
	}
	return out
}

func (a Array) deepCopy() Value {
	out := make(Array, 0, len(a))
	for _, v := range a {
		out = append(out, deepCopy(v))
	}
	return out
}

func deepCopy(v Value) Value {
	if v == nil {
		return nil
	}
	if dv, ok := v.(deepValue); ok {
		return dv.deepCopy()
	}
	return v
}

// Evaluation context

type context struct {
	root     Value
	rootPath path
	level    int
	resolved *int
	baseDir  string
	stack    []string
}

func newContext(root Value, baseDir string, stack []string) context {
	resolved := 0
	return context{root: root, rootPath: path{}, level: -1, resolved: &resolved, baseDir: baseDir, stack: stack}
}

func (c context) path() path {
	return c.rootPath[0 : c.level+1]
}

func (c context) drillInto(subpath string) context {
	out := c
	out.rootPath = append(out.rootPath, asPath(subpath)...)
	out.level += 1
	return out
}

func (c context) withIncludeFile(filename string) context {
	out := c
	out.baseDir = filepath.Dir(filename)
	out.stack = append(slices.Clone(c.stack), filename)
	return out
}

func (c context) resolve(ref valueRef) (lookupResult, Value) {
	// If root is not an object (i.e. a concatOf), postpone resolution:
	root, ok := c.root.(Object)
	if !ok {
		return valueIndeterminate, c.root
	}
	r, v := root.lookup(ref.to())
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
func (i includeOf) Type() Type    { return intermediateType }
func (m missingValue) Type() Type { return intermediateType }
func (v errorValue) Type() Type   { return intermediateType }

func (v valueRef) MarshalJSON() ([]byte, error)     { return unmarshallableValue(v) }
func (c concatOf) MarshalJSON() ([]byte, error)     { return unmarshallableValue(c) }
func (m mergeOf) MarshalJSON() ([]byte, error)      { return unmarshallableValue(m) }
func (i includeOf) MarshalJSON() ([]byte, error)    { return unmarshallableValue(i) }
func (m missingValue) MarshalJSON() ([]byte, error) { return unmarshallableValue(m) }
func (v errorValue) MarshalJSON() ([]byte, error)   { return unmarshallableValue(v) }

func unmarshallableValue(v Value) ([]byte, error) {
	return nil, fmt.Errorf("intermediate HOCON value %T", v)
}

func (v errorValue) deepCopy() Value {
	return errorValue{err: v.err, inner: deepCopy(v.inner)}
}

type lookupResult int

const (
	valueFound lookupResult = iota
	valueUndefined
	valueUnresolvable
	valueIndeterminate
)

func (o Object) lookup(path path) (lookupResult, Value) {
	if v, ok := o[path[0]]; ok {
		switch v := v.(type) {
		case valueRef, concatOf, mergeOf, includeOf:
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
	return nil
}

func (s1 String) tryConcat(node concatableValue) Value {
	if s2, ok := node.(String); ok {
		return s1 + s2
	}
	return nil
}

func rewriteIndices(a Array, t Object) Value {
	for k, v := range t {
		pos, err := strconv.ParseInt(k, 10, 32)
		idx := int(pos - 1)
		if err != nil || idx < 0 || idx > len(a) {
			return errorValue{ErrBadArrayIndex, t}
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
			return rewriteIndices(a1, t2)
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
	c2vs := make([]Value, 0, len(c2.inner))
	for _, n2 := range c2.inner {
		if ref, ok := n2.(valueRef); ok && ref.refersTo(ctx.path()) {
			*ctx.resolved += 1
			c2vs = append(c2vs, c1.inner...)
			continue
		}
		c2vs = append(c2vs, n2)
	}
	c2n := concatOf{c2vs}
	v1 := c1.minimize()
	v2 := c2n.minimize()
	return mergeOf{v1, v2}
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
			return deepCopy(refValue)
		}
		return v
	case concatOf:
		return v.reduce(ctx)
	case mergeOf:
		return v.reduce(ctx)
	case includeOf:
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

func (i includeOf) reduce(ctx context) Value {
	*ctx.resolved += 1
	target := i.path
	if !filepath.IsAbs(target) {
		target = filepath.Join(ctx.baseDir, target)
	}
	target = filepath.Clean(target)
	if slices.Contains(ctx.stack, target) {
		return errorValue{ErrIncludeCycle, i}
	}
	parsed, err := ParseFile(target)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return errorValue{IncludeError{Path: target, Err: err}, i}
		}
		if i.required {
			return errorValue{IncludeError{Path: target, Err: err}, i}
		}
		return Object{}
	}
	includedRoot := parsed.(propertyList).intoValue()
	includedValue := rebaseValueReferences(includedRoot, ctx.path())
	return reduceValue(ctx.withIncludeFile(target), includedValue)
}

func (c concatOf) reduce(ctx context) Value {
	out := concatOf{}
	for _, node := range c.inner {
		switch node := node.(type) {
		case valueRef:
			r, refValue := ctx.resolve(node)
			switch r {
			case valueFound:
				// If value is missing, exclude it from the reduced concat:
				if _, missing := refValue.(missingValue); !missing {
					out.inner = append(out.inner, deepCopy(refValue))
				}
			default:
				out.inner = append(out.inner, node)
			}
		default:
			reduced := reduceValue(ctx, node)
			if err, ok := reduced.(errorValue); ok {
				return err
			}
			out.inner = append(out.inner, reduced)
		}
	}
	// All references and/or includes resolved to missing values, reduce whole concat to missing value:
	if len(out.inner) == 0 {
		return missingValue{}
	}
	return out.minimize()
}

func rebaseValueReferences(v Value, prefix path) Value {
	if len(prefix) == 0 {
		return v
	}
	switch v := v.(type) {
	case valueRef:
		rebased := appendPath(prefix, asPath(v.path))
		v.path = strings.Join(rebased, ".")
		return v
	case Object:
		out := Object{}
		for k, v := range v {
			out[k] = rebaseValueReferences(v, prefix)
		}
		return out
	case Array:
		out := make(Array, 0, len(v))
		for _, node := range v {
			out = append(out, rebaseValueReferences(node, prefix))
		}
		return out
	case concatOf:
		out := concatOf{}
		for _, node := range v.inner {
			out.inner = append(out.inner, rebaseValueReferences(node, prefix))
		}
		return out
	case mergeOf:
		return mergeOf{
			left:  rebaseValueReferences(v.left, prefix),
			right: rebaseValueReferences(v.right, prefix),
		}
	default:
		return v
	}
}
