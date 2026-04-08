package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// JSON checks use assert.JSONEq: both sides are parsed as JSON and compared
// semantically, so key order and whitespace in the expected literal do not matter.
func TestDeleteFieldPath(t *testing.T) {
	t.Run("empty path preserves original bytes", func(t *testing.T) {
		in := []byte(`{"a":1,"b":{"c":2}}`)
		out, err := deleteFieldPath(in, nil)
		require.NoError(t, err)
		assert.Equal(t, in, out)

		out2, err := deleteFieldPath(in, []string{})
		require.NoError(t, err)
		assert.Equal(t, in, out2)
	})

	t.Run("deletes top-level field", func(t *testing.T) {
		in := []byte(`{"keep":1,"drop":{"v":[1,2,3]}}`)
		out, err := deleteFieldPath(in, []string{"drop"})
		require.NoError(t, err)
		assert.JSONEq(t, `{"keep":1}`, string(out))
	})

	t.Run("deletes nested field", func(t *testing.T) {
		in := []byte(`{"spec":{"replicas":3,"template":{"x":1}}}`)
		out, err := deleteFieldPath(in, []string{"spec", "replicas"})
		require.NoError(t, err)
		assert.JSONEq(t, `{"spec":{"template":{"x":1}}}`, string(out))
	})

	t.Run("missing intermediate key returns original bytes", func(t *testing.T) {
		in := []byte(`{"metadata":{"name":"x"}}`)
		out, err := deleteFieldPath(in, []string{"spec", "template"})
		require.NoError(t, err)
		assert.Equal(t, in, out)
	})

	t.Run("intermediate non-object value preserves original bytes", func(t *testing.T) {
		in := []byte(`{"spec":42}`)
		out, err := deleteFieldPath(in, []string{"spec", "foo"})
		require.NoError(t, err)
		assert.Equal(t, in, out)
	})

	t.Run("path into nested array preserves original bytes", func(t *testing.T) {
		in := []byte(`{"items":[1,2,3]}`)
		out, err := deleteFieldPath(in, []string{"items", "x"})
		require.NoError(t, err)
		assert.Equal(t, in, out)
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		_, err := deleteFieldPath([]byte(`not json`), []string{"a"})
		require.Error(t, err)
	})

	t.Run("non-object root returns error", func(t *testing.T) {
		_, err := deleteFieldPath([]byte(`[1,2]`), []string{"0"})
		require.Error(t, err)
	})

	t.Run("missing leaf key preserves original bytes", func(t *testing.T) {
		in := []byte(`{"spec":{"foo":1}}`)
		out, err := deleteFieldPath(in, []string{"spec", "missing"})
		require.NoError(t, err)
		assert.Equal(t, in, out)
	})
}
