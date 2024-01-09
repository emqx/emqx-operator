package v2beta1

import (
	"fmt"
	"testing"

	"github.com/rory-z/go-hocon"
	"github.com/stretchr/testify/assert"
)

func TestMergeDefaultConfig(t *testing.T) {
	t.Run("case1", func(t *testing.T) {
		config := ""
		got := mergeDefaultConfig(config)
		assert.Equal(t, "1883", got.GetString("listeners.tcp.default.bind"))
		assert.Equal(t, "8883", got.GetString("listeners.ssl.default.bind"))
		assert.Equal(t, "8083", got.GetString("listeners.ws.default.bind"))
		assert.Equal(t, "8084", got.GetString("listeners.wss.default.bind"))
	})

	t.Run("case2", func(t *testing.T) {
		config := ""
		config += fmt.Sprintln("listeners.tcp.default.bind = 31883")
		config += fmt.Sprintln("listeners.ssl.default.bind = 38883")
		config += fmt.Sprintln("listeners.ws.default.bind  = 38083")
		config += fmt.Sprintln("listeners.wss.default.bind = 38084")

		got := mergeDefaultConfig(config)
		assert.Equal(t, "31883", got.GetString("listeners.tcp.default.bind"))
		assert.Equal(t, "38883", got.GetString("listeners.ssl.default.bind"))
		assert.Equal(t, "38083", got.GetString("listeners.ws.default.bind"))
		assert.Equal(t, "38084", got.GetString("listeners.wss.default.bind"))
	})
}

func TestDeepEqualHoconValue(t *testing.T) {
	t.Run("case1", func(t *testing.T) {
		v1 := hocon.String("a")
		v2 := hocon.String("a")
		assert.True(t, deepEqualHoconValue(v1, v2))
	})

	t.Run("case2", func(t *testing.T) {
		v1 := hocon.String("a")
		v2 := hocon.String("b")
		assert.False(t, deepEqualHoconValue(v1, v2))
	})

	t.Run("case3", func(t *testing.T) {
		v1 := hocon.Int(1)
		v2 := hocon.Int(1)
		assert.True(t, deepEqualHoconValue(v1, v2))
	})

	t.Run("case4", func(t *testing.T) {
		v1 := hocon.Int(1)
		v2 := hocon.Int(2)
		assert.False(t, deepEqualHoconValue(v1, v2))
	})

	t.Run("case5", func(t *testing.T) {
		v1 := hocon.Boolean(true)
		v2 := hocon.Boolean(true)
		assert.True(t, deepEqualHoconValue(v1, v2))
	})

	t.Run("case6", func(t *testing.T) {
		v1 := hocon.Boolean(true)
		v2 := hocon.Boolean(false)
		assert.False(t, deepEqualHoconValue(v1, v2))
	})

	t.Run("case7", func(t *testing.T) {
		v1 := hocon.Null("null")
		v2 := hocon.Null("null")
		assert.True(t, deepEqualHoconValue(v1, v2))
	})

	t.Run("case8", func(t *testing.T) {
		v1 := hocon.Null("fake")
		v2 := hocon.Null("")
		assert.True(t, deepEqualHoconValue(v1, v2))
	})

	t.Run("case9", func(t *testing.T) {
		v1 := hocon.Array{hocon.String("a"), hocon.String("b")}
		v2 := hocon.Array{hocon.String("a"), hocon.String("b")}
		assert.True(t, deepEqualHoconValue(v1, v2))
	})

	t.Run("case10", func(t *testing.T) {
		v1 := hocon.Array{hocon.String("a"), hocon.String("b")}
		v2 := hocon.Array{hocon.String("a"), hocon.String("c")}
		assert.False(t, deepEqualHoconValue(v1, v2))
	})

	t.Run("case11", func(t *testing.T) {
		v1 := hocon.Array{hocon.String("a"), hocon.String("b")}
		v2 := hocon.Array{hocon.String("a"), hocon.String("b"), hocon.String("c")}
		assert.False(t, deepEqualHoconValue(v1, v2))
	})

	t.Run("case12", func(t *testing.T) {
		v1 := hocon.Object{"a": hocon.Int(1), "b": hocon.Int(2)}
		v2 := hocon.Object{"b": hocon.Int(2), "a": hocon.Int(1)}
		assert.True(t, deepEqualHoconValue(v1, v2))
	})

	t.Run("case13", func(t *testing.T) {
		v1 := hocon.Object{"a": hocon.Int(1), "b": hocon.Int(2)}
		v2 := hocon.Object{"a": hocon.Int(1), "c": hocon.Int(3)}
		assert.False(t, deepEqualHoconValue(v1, v2))
	})

	t.Run("case13", func(t *testing.T) {
		v1 := hocon.Object{"a": hocon.Int(1), "b": hocon.Int(2)}
		v2 := hocon.Object{"a": hocon.Int(1), "b": hocon.Int(2), "c": hocon.Int(3)}
		assert.False(t, deepEqualHoconValue(v1, v2))
	})

	t.Run("case14", func(t *testing.T) {
		v1 := hocon.Object{
			"a": hocon.String("a1"),
			"b": hocon.Object{"b1": hocon.String("b1"), "b2": hocon.String("b2")},
			"c": hocon.Array{hocon.String("c1"), hocon.String("c2")},
		}
		v2 := hocon.Object{
			"c": hocon.Array{hocon.String("c1"), hocon.String("c2")},
			"b": hocon.Object{"b2": hocon.String("b2"), "b1": hocon.String("b1")},
			"a": hocon.String("a1"),
		}
		assert.True(t, deepEqualHoconValue(v1, v2))
	})

	t.Run("case15", func(t *testing.T) {
		v1 := hocon.Object{
			"a": hocon.String("a1"),
			"b": hocon.Object{"b1": hocon.String("b1"), "b2": hocon.String("b2")},
			"c": hocon.Array{hocon.String("c1"), hocon.String("c2")},
		}
		v2 := hocon.Object{
			"c": hocon.Array{hocon.String("c2"), hocon.String("c1")},
			"b": hocon.Object{"b2": hocon.String("b2"), "b1": hocon.String("b1")},
			"a": hocon.String("a1"),
		}
		assert.False(t, deepEqualHoconValue(v1, v2))
	})
}
