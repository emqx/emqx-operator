package hocon

import (
	"errors"
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
				"f":          Duration(time.Minute * 30),
				"g":          Float(0.99),
				"h":          Int(524288),
				"part_empty": Object{},
				"part": Object{
					"s": String("multi\nline\nstring"),
				},
			}},
		))
	})

	t.Run("overrides", func(t *testing.T) {
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

	t.Run("selfrefs", func(t *testing.T) {
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

	t.Run("selfref undefined", func(t *testing.T) {
		g := NewWithT(t)
		doc, err := ParseDocument(`a = ${a}`)
		g.Expect(err).To(Succeed())
		_, err = doc.Evaluate()
		g.Expect(err).To(HaveOccurred())
		g.Expect(errors.Is(err, ErrUnresolvable)).To(BeTrue())
		g.Expect(err.Error()).To(ContainSubstring("config evaluation failed at: a"))
		g.Expect(err.Error()).To(ContainSubstring("unresolvable"))
	})
}

func TestSubstitution(t *testing.T) {
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
}
