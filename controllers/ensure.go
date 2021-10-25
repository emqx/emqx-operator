package controllers

import (
	"github.com/emqx/emqx-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Ensure the RedisCluster's components are correct.
func (ech *EmqxClusterHandler) Ensure(e *v1alpha1.Emqx, labels map[string]string, or []metav1.OwnerReference) error {
	if err := ech.eService.EnsureEmqxHeadlessService(e, labels, or); err != nil {
		return err
	}
	// if err := r.rcService.EnsureSentinelService(rc, labels, or); err != nil {
	// 	return err
	// }
	// if err := r.rcService.EnsureSentinelHeadlessService(rc, labels, or); err != nil {
	// 	return err
	// }
	// if err := r.rcService.EnsureSentinelConfigMap(rc, labels, or); err != nil {
	// 	return err
	// }
	// if err := r.rcService.EnsureSentinelProbeConfigMap(rc, labels, or); err != nil {
	// 	return err
	// }
	// if err := r.rcService.EnsureRedisShutdownConfigMap(rc, labels, or); err != nil {
	// 	return err
	// }
	// if err := r.rcService.EnsureRedisStatefulset(rc, labels, or); err != nil {
	// 	return err
	// }
	// if err := r.rcService.EnsureSentinelStatefulset(rc, labels, or); err != nil {
	// 	return err
	// }

	return nil
}
