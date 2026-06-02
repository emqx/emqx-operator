package hocon

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func TestParse(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		g := NewWithT(t)
		doc, err := ParseDocument("")
		g.Expect(err).To(Succeed())
		root, err := doc.Evaluate()
		g.Expect(err).To(Succeed())
		g.Expect(root).To(BeComparableTo(Object{}))
	})

	t.Run("mixed", func(t *testing.T) {
		g := NewWithT(t)
		doc, err := ParseDocument(`
		object { a = 42, b=ident0
                 c = [1.0 -2.0 +3.0 true]
                 d = "quoted\nstring"
                 e:null
                 , f : 30m
                 , g : 99%,
                 h = 512KB } {
                 part_empty{} // leave empty
                 part :{
                   # Should preserve newlines
                   s = """~
                     multi
                     line
                     string~"""
                 }
               }`)
		g.Expect(err).To(Succeed())
		root, err := doc.Evaluate()
		g.Expect(err).To(Succeed())
		g.Expect(root).To(BeComparableTo(
			Object{"object": Object{
				"a":          Int(42),
				"b":          String("ident0"),
				"c":          Array{Float(1), Float(-2), Float(3), Bool(true)},
				"d":          String("quoted\nstring"),
				"e":          nil,
				"f":          DurationString("30m"),
				"g":          PercentString("99%"),
				"h":          BytesizeString("512KB"),
				"part_empty": Object{},
				"part": Object{
					"s": String("multi\nline\nstring"),
				},
			}},
		))
	})

	t.Run("partials merge correctly", func(t *testing.T) {
		g := NewWithT(t)
		doc, err := ParseDocument(`
		object { c = [1 2 3 true]
		     } { c.5 = appended, c.2 = rewritten
             } { x = { y.o1: "base" }
             } { x.y.o2 = "single-field" }
        object.x.y = { o1: "rewrite", o3: "merged-in" }`)
		g.Expect(err).To(Succeed())
		root, err := doc.Evaluate()
		g.Expect(err).To(Succeed())
		g.Expect(root).To(BeComparableTo(
			Object{"object": Object{
				"c": Array{Int(1), String("rewritten"), Int(3), Bool(true), String("appended")},
				"x": Object{
					"y": Object{
						"o1": String("rewrite"),
						"o2": String("single-field"),
						"o3": String("merged-in"),
					},
				},
			}},
		))
	})

	t.Run("mixed immediate partials are parse errors", func(t *testing.T) {
		g := NewWithT(t)
		_, err := ParseDocument(`a = [] "x"`)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring(ErrMixedPartials.Error()))
	})
}

func TestMarshalJSON(t *testing.T) {
	t.Run("evaluated values marshal with default json encoding", func(t *testing.T) {
		g := NewWithT(t)
		out, err := json.Marshal(Object{
			"array": Array{String("value"), Int(42), Float(1.5), Bool(true), nil},
			"typed": Object{
				"duration": DurationString("1d"),
				"bytesize": BytesizeString("512KB"),
				"percent":  PercentString("99%"),
			},
		})
		g.Expect(err).To(Succeed())
		g.Expect(out).To(MatchJSON(`{
			"array":["value",42,1.5,true,null],
			"typed":{"duration":"1d","bytesize":"512KB","percent":"99%"}
		}`))
	})

	t.Run("intermediate values fail to marshal", func(t *testing.T) {
		g := NewWithT(t)
		_, err := json.Marshal(Object{"a": valueRef{path: "b"}})
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("intermediate HOCON value"))
	})
}

func TestObjectLookup(t *testing.T) {
	g := NewWithT(t)
	root := Object{
		"a": Object{
			"b": Int(42),
			"c": nil,
		},
		"scalar": String("value"),
	}

	g.Expect(root.Lookup("a.b")).To(Equal(Int(42)))
	g.Expect(root.Lookup("a.c")).To(BeNil())
	g.Expect(root.Lookup("missing")).To(BeNil())
	g.Expect(root.Lookup("scalar.child")).To(BeNil())
	g.Expect(root.Lookup("")).To(BeNil())
}

func TestSelfref(t *testing.T) {
	t.Run("selfrefs evaluate correctly", func(t *testing.T) {
		g := NewWithT(t)
		doc, err := ParseDocument(`object { c = [1 2 3 true] }
			object.a = [
			  {x: 0.0} ${object.y}
			  {x: 1.0, y: -17.1}
			  {x: 2.0, y: -17.9}
			  {x: 3.0, y: -22.1}
			]
	    	object.x = "hello"
	    	object.x = ${object.x} "world"
	    	object.x = ${object.x} "wide web"
	    	object.xx = "'" ${object.x} "'"
			object.y { f1: 42, f2: ${object.c}, f3: ${object.y.f1} }`)
		g.Expect(err).To(Succeed())
		root, err := doc.Evaluate()
		g.Expect(err).To(Succeed())
		g.Expect(root).To(BeComparableTo(
			Object{"object": Object{
				"c": Array{Int(1), Int(2), Int(3), Bool(true)},
				"a": Array{
					Object{"x": Float(0), "f1": Int(42), "f2": Array{Int(1), Int(2), Int(3), Bool(true)}, "f3": Int(42)},
					Object{"x": Float(1), "y": Float(-17.1)},
					Object{"x": Float(2), "y": Float(-17.9)},
					Object{"x": Float(3), "y": Float(-22.1)},
				},
				// "x":  String("hello world wide web"),
				"x": String("helloworldwide web"),
				// "xx": String("'hello world wide web'"),
				"xx": String("'helloworldwide web'"),
				"y":  Object{"f1": Int(42), "f2": Array{Int(1), Int(2), Int(3), Bool(true)}, "f3": Int(42)},
			}},
		))
	})

	t.Run("selfref with no existing value is unresolvable", func(t *testing.T) {
		g := NewWithT(t)
		doc, err := ParseDocument(`a = ${a}`)
		g.Expect(err).To(Succeed())
		_, err = doc.Evaluate()
		g.Expect(err).To(HaveOccurred())
		g.Expect(errors.Is(err, ErrUnresolvable)).To(BeTrue())
		g.Expect(err.Error()).To(ContainSubstring("config evaluation failed at: a"))
		g.Expect(err.Error()).To(ContainSubstring("unresolvable"))
	})

	t.Run("selfref substitution", func(t *testing.T) {
		g := NewWithT(t)
		doc, err := ParseDocument(`
			a = "1"
			a = ${a} "2"`)
		g.Expect(err).To(Succeed())
		root, err := doc.Evaluate()
		g.Expect(err).To(Succeed())
		g.Expect(root).To(BeComparableTo(Object{"a": String("12")}))
	})

	t.Run("optional selfref substitution", func(t *testing.T) {
		g := NewWithT(t)
		doc, err := ParseDocument(`
			a = "1"
			a = ${?a} "2"`)
		g.Expect(err).To(Succeed())
		root, err := doc.Evaluate()
		g.Expect(err).To(Succeed())
		g.Expect(root).To(BeComparableTo(Object{"a": String("12")}))
	})

	t.Run("object merge skips trailing self reference", func(t *testing.T) {
		g := NewWithT(t)
		doc, err := ParseDocument(`
			a = { x = 1 }
			a = { y = 2 } ${a}`)
		g.Expect(err).To(Succeed())
		root, err := doc.Evaluate()
		g.Expect(err).To(Succeed())
		g.Expect(root).To(BeComparableTo(Object{
			"a": Object{
				"x": Int(1),
				"y": Int(2),
			},
		}))
	})
}

func TestSubstitution(t *testing.T) {
	t.Run("missing optional substitution is ignored", func(t *testing.T) {
		g := NewWithT(t)
		doc, err := ParseDocument(`obj {
			a = 42
			b = ${?missing} 
			c = ${?missing1} ${?missing2} ${?missing3} }`)
		g.Expect(err).To(Succeed())
		root, err := doc.Evaluate()
		g.Expect(err).To(Succeed())
		g.Expect(root).To(BeComparableTo(Object{"obj": Object{"a": Int(42)}}))
	})

	t.Run("missing optional substitution preserves previous value", func(t *testing.T) {
		g := NewWithT(t)
		doc, err := ParseDocument(`
			a = 1
			a = ${?missing}`)
		g.Expect(err).To(Succeed())
		root, err := doc.Evaluate()
		g.Expect(err).To(Succeed())
		g.Expect(root).To(BeComparableTo(Object{"a": Int(1)}))
	})

	t.Run("missing optional substitution is overridable", func(t *testing.T) {
		g := NewWithT(t)
		doc, err := ParseDocument(`
			a = ${?missing}
			a = 2`)
		g.Expect(err).To(Succeed())
		root, err := doc.Evaluate()
		g.Expect(err).To(Succeed())
		g.Expect(root).To(BeComparableTo(Object{"a": Int(2)}))
	})

	t.Run("missing optional substitution preserves explicit null", func(t *testing.T) {
		g := NewWithT(t)
		doc, err := ParseDocument(`
			a = null
			a = ${?missing}`)
		g.Expect(err).To(Succeed())
		root, err := doc.Evaluate()
		g.Expect(err).To(Succeed())
		g.Expect(root).To(BeComparableTo(Object{"a": nil}))
	})

	t.Run("missing required substitution is evaluation error", func(t *testing.T) {
		g := NewWithT(t)
		doc, err := ParseDocument(`a = ${missing}`)
		g.Expect(err).To(Succeed())
		_, err = doc.Evaluate()
		g.Expect(err).To(HaveOccurred())
		g.Expect(errors.Is(err, ErrUndefined)).To(BeTrue())
		g.Expect(err.Error()).To(ContainSubstring("config evaluation failed at: a"))
		g.Expect(err.Error()).To(ContainSubstring("reference points to undefined value"))
	})

	t.Run("missing required substitution in array includes element path", func(t *testing.T) {
		g := NewWithT(t)
		doc, err := ParseDocument(`a = [${missing}]`)
		g.Expect(err).To(Succeed())
		_, err = doc.Evaluate()
		g.Expect(err).To(HaveOccurred())
		g.Expect(errors.Is(err, ErrUndefined)).To(BeTrue())
		g.Expect(err.Error()).To(ContainSubstring(`config evaluation failed at: a.1`))
	})

	t.Run("mixed partials are evaluation errors", func(t *testing.T) {
		g := NewWithT(t)
		doc, err := ParseDocument(`base = [1]
								   a = ${base} "x"`)
		g.Expect(err).To(Succeed())
		_, err = doc.Evaluate()
		g.Expect(err).To(HaveOccurred())
		g.Expect(errors.Is(err, ErrMixedPartials)).To(BeTrue())
		g.Expect(strings.Count(err.Error(), "config evaluation failed")).To(Equal(1))
	})

	t.Run("mixed partials after substitution is unresolvable", func(t *testing.T) {
		g := NewWithT(t)
		doc, err := ParseDocument(`base = {x = 1}
								   a = ${base} [2]`)
		g.Expect(err).To(Succeed())
		_, err = doc.Evaluate()
		g.Expect(err).To(HaveOccurred())
		g.Expect(errors.Is(err, ErrMixedPartials)).To(BeTrue())
		g.Expect(err.Error()).To(ContainSubstring("config evaluation failed at: a"))
	})

	t.Run("substitution path through non-object is unresolvable", func(t *testing.T) {
		g := NewWithT(t)
		doc, err := ParseDocument(`a = 1
								   b = ${a.x}`)
		g.Expect(err).To(Succeed())
		_, err = doc.Evaluate()
		g.Expect(err).To(HaveOccurred())
		g.Expect(errors.Is(err, ErrUnresolvable)).To(BeTrue())
		g.Expect(err.Error()).To(ContainSubstring("config evaluation failed at: b"))
	})

	t.Run("out of bounds array index update is forbidden", func(t *testing.T) {
		g := NewWithT(t)
		doc, err := ParseDocument(`a = [1, 2, {d: 42}], a.42.x = foo`)
		g.Expect(err).To(Succeed())
		_, err = doc.Evaluate()
		g.Expect(err).To(HaveOccurred())
		g.Expect(errors.Is(err, ErrBadArrayIndex)).To(BeTrue())
		g.Expect(err.Error()).To(ContainSubstring(`config evaluation failed at: a`))
		g.Expect(err.Error()).To(ContainSubstring(`out of bounds`))
	})

	t.Run("array concat accepts object index update", func(t *testing.T) {
		g := NewWithT(t)
		doc, err := ParseDocument(`
			a = [1, 2] ${?missing}
			a { "2" = rewritten, "3" = appended }`)
		g.Expect(err).To(Succeed())
		root, err := doc.Evaluate()
		g.Expect(err).To(Succeed())
		g.Expect(root).To(BeComparableTo(Object{
			"a": Array{Int(1), String("rewritten"), String("appended")},
		}))
	})

	t.Run("mixed partials after substitution", func(t *testing.T) {
		g := NewWithT(t)
		doc, err := ParseDocument(`a = {x = 1}
								   arr = [42]
								   a.y = ["{"] ${arr} ["}"]
								   obj = { foo: bar }
								   a.z = {} ${obj} {}
								   a = { y: ${a.y} ${a.z} }`)
		g.Expect(err).To(Succeed())
		_, err = doc.Evaluate()
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring(`config evaluation failed at: a.y`))
		g.Expect(err.Error()).To(ContainSubstring(`mixed-type partials`))
	})

	t.Run("incompatible values after substitution overwrite", func(t *testing.T) {
		g := NewWithT(t)
		doc, err := ParseDocument(`a = {x = 1}
								   arr = [42]
								   a.y = ["{"] ${arr} ["}"]
								   obj = { foo: bar }
								   a.z = {} ${obj} {}
								   a = { y: ${a.y} }
								   a = { y: ${a.z} }`)
		g.Expect(err).To(Succeed())
		root, err := doc.Evaluate()
		g.Expect(root).To(BeComparableTo(Object{
			"arr": Array{Int(42)},
			"obj": Object{"foo": String("bar")},
			"a": Object{
				"x": Int(1),
				"y": Object{"foo": String("bar")},
				"z": Object{"foo": String("bar")},
			},
		}))
		g.Expect(err).To(Succeed())
	})

}

func TestStrings(t *testing.T) {
	t.Run("string values expose raw string", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(String("plain").String()).To(Equal("plain"))
		g.Expect(DurationString("1d").String()).To(Equal("1d"))
		g.Expect(BytesizeString("512KB").String()).To(Equal("512KB"))
		g.Expect(PercentString("99%").String()).To(Equal("99%"))
	})

	t.Run("bytesize strings", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(BytesizeString(`42`).AsInteger()).To(Equal(int64(42)))
		g.Expect(BytesizeString(`42b`).AsInteger()).To(Equal(int64(42)))
		g.Expect(BytesizeString(`1GB`).AsInteger()).To(Equal(int64(1 * 1024 * 1024 * 1024)))
		g.Expect(BytesizeString(`-10Mb`).AsInteger()).To(Equal(int64(-10 * 1024 * 1024)))
		g.Expect(BytesizeString(`42kb`).AsInteger()).To(Equal(int64(42 * 1024)))
		// parser fallbacks
		g.Expect(BytesizeString(`MB`).AsInteger()).To(Equal(int64(1024 * 1024)))
		g.Expect(BytesizeString(`b`).AsInteger()).To(Equal(int64(1)))
		g.Expect(BytesizeString(``).AsInteger()).To(Equal(int64(0)))
	})

	t.Run("duration strings", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(DurationString(`1d`).AsDuration()).To(Equal(24 * time.Hour))
		g.Expect(DurationString(`2D`).AsDuration()).To(Equal(48 * time.Hour))
		g.Expect(DurationString(`120s`).AsDuration()).To(Equal(120 * time.Second))
	})

	t.Run("percent strings", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(PercentString(`0%`).AsFloat()).To(Equal(0.0))
		g.Expect(PercentString(`99%`).AsFloat()).To(Equal(0.99))
	})
}

func TestInclude(t *testing.T) {
	t.Run("string document includes from provided directory", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()
		g.Expect(os.WriteFile(filepath.Join(dir, "child.hocon"), []byte(`b = 2`), 0o600)).To(Succeed())
		doc, err := ParseDocument(`
			include "child.hocon"
			a = 1`,
			dir,
		)
		g.Expect(err).To(Succeed())
		root, err := doc.Evaluate()
		g.Expect(err).To(Succeed())
		g.Expect(root).To(BeComparableTo(Object{"a": Int(1), "b": Int(2)}))
	})

	t.Run("optional missing include is ignored", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()
		doc, err := ParseDocument(`
			include "missing.conf"
			a = 1`,
			dir,
		)
		g.Expect(err).To(Succeed())
		root, err := doc.Evaluate()
		g.Expect(err).To(Succeed())
		g.Expect(root).To(BeComparableTo(Object{"a": Int(1)}))
	})

	t.Run("optional missing include keeps surrounding object", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()
		doc, err := ParseDocument(`a { include "missing.conf" }`, dir)
		g.Expect(err).To(Succeed())
		root, err := doc.Evaluate()
		g.Expect(err).To(Succeed())
		g.Expect(root).To(BeComparableTo(Object{"a": Object{}}))
	})

	t.Run("required missing include fails", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()
		doc, err := ParseDocument(`
			a = 1
			include required("missing.conf")
			b = 2`,
			dir,
		)
		g.Expect(err).To(Succeed())
		_, err = doc.Evaluate()
		g.Expect(errors.Is(err, ErrIncludeFailed)).To(BeTrue())
		g.Expect(errors.Is(err, os.ErrNotExist)).To(BeTrue())
	})

	t.Run("include cycle fails", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()
		path := filepath.Join(dir, "cycle.hocon")
		g.Expect(os.WriteFile(path, []byte(`include "cycle.hocon"`), 0o600)).To(Succeed())
		doc, err := ParseDocumentFile(path)
		g.Expect(err).To(Succeed())
		_, err = doc.Evaluate()
		g.Expect(errors.Is(err, ErrIncludeCycle)).To(BeTrue())
	})

	t.Run("included parse error uses included file name", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()
		childPath := filepath.Join(dir, "child.hocon")
		g.Expect(os.WriteFile(childPath, []byte(`[1,2,3]`), 0o600)).To(Succeed())

		doc, err := ParseDocument(`include required("child.hocon")`, dir)
		g.Expect(err).To(Succeed())
		_, err = doc.Evaluate()
		g.Expect(err).To(HaveOccurred())
		g.Expect(errors.Is(err, ErrIncludeFailed)).To(BeTrue())
		g.Expect(err.Error()).To(ContainSubstring("child.hocon"))

		doc, err = ParseDocument(`include "child.hocon"`, dir)
		g.Expect(err).To(Succeed())
		_, err = doc.Evaluate()
		g.Expect(err).To(HaveOccurred())
		g.Expect(errors.Is(err, ErrIncludeFailed)).To(BeTrue())
		g.Expect(err.Error()).To(ContainSubstring("child.hocon"))
	})

	t.Run("nested include rebases substitutions", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()
		rootPath := filepath.Join(dir, "root.hocon")
		childPath := filepath.Join(dir, "child.hocon")
		g.Expect(os.WriteFile(
			childPath,
			[]byte(`x = 10
					y = ${x}
					n { x = 20, y = ${x}, z = ${n.x} }
					arr: [ ${x}, ${y} ]`),
			0o600,
		)).To(Succeed())
		g.Expect(os.WriteFile(
			rootPath,
			[]byte(`a { include "child.hocon" }, a.x = 42`),
			0o600,
		)).To(Succeed())
		doc, err := ParseDocumentFile(rootPath)
		g.Expect(err).To(Succeed())
		root, err := doc.Evaluate()
		g.Expect(err).To(Succeed())
		g.Expect(root).To(BeComparableTo(Object{
			"a": Object{
				"x": Int(42),
				"y": Int(42),
				"n": Object{
					"x": Int(20),
					"y": Int(42),
					"z": Int(20),
				},
				"arr": Array{Int(42), Int(42)},
			},
		}))
	})

	t.Run("root include substitutions see final root object", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()
		rootPath := filepath.Join(dir, "root.hocon")
		childPath := filepath.Join(dir, "child.hocon")
		g.Expect(os.WriteFile(childPath, []byte(`x = 10, y = ${x}`), 0o600)).To(Succeed())
		g.Expect(os.WriteFile(
			rootPath,
			[]byte(`include "child.hocon"
					x = 42`),
			0o600,
		)).To(Succeed())
		doc, err := ParseDocumentFile(rootPath)
		g.Expect(err).To(Succeed())
		root, err := doc.Evaluate()
		g.Expect(err).To(Succeed())
		g.Expect(root).To(BeComparableTo(Object{
			"x": Int(42),
			"y": Int(42),
		}))
	})

	t.Run("nested include resolves relative to included file", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()
		childDir := filepath.Join(dir, "child")
		g.Expect(os.Mkdir(childDir, 0o700)).To(Succeed())
		rootPath := filepath.Join(dir, "root.hocon")
		childPath := filepath.Join(childDir, "child.hocon")
		grandchildPath := filepath.Join(childDir, "grandchild.hocon")
		g.Expect(os.WriteFile(
			rootPath,
			[]byte(`a { include "child/child.hocon" }`),
			0o600,
		)).To(Succeed())
		g.Expect(os.WriteFile(
			childPath,
			[]byte(`include "grandchild.hocon"
					y = ${x}`),
			0o600,
		)).To(Succeed())
		g.Expect(os.WriteFile(
			grandchildPath,
			[]byte(`x = 7`),
			0o600,
		)).To(Succeed())
		doc, err := ParseDocumentFile(rootPath)
		g.Expect(err).To(Succeed())
		root, err := doc.Evaluate()
		g.Expect(err).To(Succeed())
		g.Expect(root).To(BeComparableTo(Object{
			"a": Object{
				"x": Int(7),
				"y": Int(7),
			},
		}))
	})
}
