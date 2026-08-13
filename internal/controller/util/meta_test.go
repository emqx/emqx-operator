package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAttachAnnotation(t *testing.T) {
	object := &metav1.PartialObjectMetadata{}

	assert.True(t, AttachAnnotation(object, "key", "value"))
	assert.Equal(t, map[string]string{"key": "value"}, object.GetAnnotations())
	assert.False(t, AttachAnnotation(object, "key", "value"))
	assert.True(t, AttachAnnotation(object, "key", "updated"))
	assert.Equal(t, "updated", object.GetAnnotations()["key"])
}

func TestAttachAnnotations(t *testing.T) {
	object := &metav1.PartialObjectMetadata{}

	assert.False(t, AttachAnnotations(object, nil))
	assert.Nil(t, object.GetAnnotations())
	assert.True(t, AttachAnnotations(object, map[string]string{"one": "1", "two": "2"}))
	assert.False(t, AttachAnnotations(object, map[string]string{"one": "1", "two": "2"}))
	assert.Equal(t, map[string]string{"one": "1", "two": "2"}, object.GetAnnotations())
}

func TestPeekAnnotations(t *testing.T) {
	object := &metav1.PartialObjectMetadata{ObjectMeta: metav1.ObjectMeta{
		Annotations: map[string]string{"one": "1", "two": "2", "three": "3"},
	}}

	peeked := PeekAnnotations(object, "one", "three", "missing")

	assert.Equal(t, map[string]string{"one": "1", "three": "3"}, peeked)
	peeked["one"] = "updated"
	assert.Equal(t, "1", object.GetAnnotations()["one"])
}

func TestUnsetAnnotations(t *testing.T) {
	object := &metav1.PartialObjectMetadata{ObjectMeta: metav1.ObjectMeta{
		Annotations: map[string]string{"one": "1", "two": "2"},
	}}

	assert.True(t, UnsetAnnotations(object, "one", "missing"))
	assert.Equal(t, map[string]string{"two": "2"}, object.GetAnnotations())
	assert.False(t, UnsetAnnotations(object, "one", "missing"))
}
