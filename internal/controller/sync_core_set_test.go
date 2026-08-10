package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSortByOrdinal(t *testing.T) {
	mkPod := func(name string) *corev1.Pod {
		return &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: name},
		}
	}

	t.Run("fewer than 10 pods", func(t *testing.T) {
		pods := []*corev1.Pod{
			mkPod("emqx-core-2"),
			mkPod("emqx-core-0"),
			mkPod("emqx-core-1"),
		}
		sortByOrdinal(pods)
		assert.Equal(t, "emqx-core-0", pods[0].Name)
		assert.Equal(t, "emqx-core-1", pods[1].Name)
		assert.Equal(t, "emqx-core-2", pods[2].Name)
	})

	t.Run("10+ pods sort numerically not lexicographically", func(t *testing.T) {
		pods := []*corev1.Pod{
			mkPod("emqx-core-9"),
			mkPod("emqx-core-10"),
			mkPod("emqx-core-2"),
			mkPod("emqx-core-11"),
			mkPod("emqx-core-0"),
			mkPod("emqx-core-1"),
		}
		sortByOrdinal(pods)
		expected := []string{
			"emqx-core-0",
			"emqx-core-1",
			"emqx-core-2",
			"emqx-core-9",
			"emqx-core-10",
			"emqx-core-11",
		}
		got := make([]string, len(pods))
		for i, p := range pods {
			got[i] = p.Name
		}
		assert.Equal(t, expected, got)
	})

	t.Run("pods without ordinal suffix sort first", func(t *testing.T) {
		pods := []*corev1.Pod{
			mkPod("emqx-core-1"),
			mkPod("emqx-core-nonum"),
			mkPod("emqx-core-0"),
		}
		sortByOrdinal(pods)
		assert.Equal(t, "emqx-core-nonum", pods[0].Name)
		assert.Equal(t, "emqx-core-0", pods[1].Name)
		assert.Equal(t, "emqx-core-1", pods[2].Name)
	})
}
