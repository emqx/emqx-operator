package cache

import (
	"fmt"
	"sync"

	"github.com/emqx/emqx-operator/api/v1alpha1"
)

type StateType string

const (
	Create StateType = "create"
	Update StateType = "update"
	Check  StateType = "check"
)

// Meta contains EMQ X Cluster some metadata
type Meta struct {
	NameSpace string
	Name      string
	State     StateType
	Size      int32
	Obj       *v1alpha1.Emqx

	Status  v1alpha1.ConditionType
	Message string

	Config map[string]string
}

func newCluster(e *v1alpha1.Emqx) *Meta {
	return &Meta{
		Status: v1alpha1.ClusterConditionCreating,
		// Config:    e.Spec.Config,
		Obj:       e,
		Size:      *e.Spec.Replicas,
		State:     Create,
		Name:      e.GetName(),
		NameSpace: e.GetNamespace(),
		Message:   "Bootstrap emqx cluster",
	}
}

// MetaMap cache last EMQ X Cluster and meta data
type MetaMap struct {
	sync.Map
}

func (c *MetaMap) Cache(obj *v1alpha1.Emqx) *Meta {
	meta, ok := c.Load(getNamespacedName(obj.GetNamespace(), obj.GetName()))
	if !ok {
		c.Add(obj)
	} else {
		c.Update(meta.(*Meta), obj)
	}
	return c.Get(obj)
}

func (c *MetaMap) Get(obj *v1alpha1.Emqx) *Meta {
	meta, _ := c.Load(getNamespacedName(obj.GetNamespace(), obj.GetName()))
	return meta.(*Meta)
}

func (c *MetaMap) Add(obj *v1alpha1.Emqx) {
	c.Store(getNamespacedName(obj.GetNamespace(), obj.GetName()), newCluster(obj))
}

func (c *MetaMap) Del(obj *v1alpha1.Emqx) {
	c.Delete(getNamespacedName(obj.GetNamespace(), obj.GetName()))
}

func (c *MetaMap) Update(meta *Meta, new *v1alpha1.Emqx) {
	if meta.Obj.GetGeneration() == new.GetGeneration() {
		meta.State = Check
		return
	}

	old := meta.Obj
	meta.State = Update
	meta.Size = *old.Spec.Replicas
	// Password change is not allowed
	// new.Spec.Password = old.Spec.Password
	// meta.Auth.Password = old.Spec.Password
	meta.Obj = new

	meta.Status = v1alpha1.ClusterConditionUpdating
	meta.Message = "Updating emqx config"
	if isImagesChanged(old, new) {
		meta.Status = v1alpha1.ClusterConditionUpgrading
		meta.Message = fmt.Sprintf("Upgrading to %s", new.Spec.Image)
	}
	if isScalingDown(old, new) {
		meta.Status = v1alpha1.ClusterConditionScalingDown
		meta.Message = fmt.Sprintf("Scaling down form: %d to: %d", meta.Size, new.Spec.Replicas)
	}
	if isScalingUp(old, new) {
		meta.Status = v1alpha1.ClusterConditionScaling
		meta.Message = fmt.Sprintf("Scaling up form: %d to: %d", meta.Size, new.Spec.Replicas)
	}
	// if isResourcesChange(old, new) {
	// 	meta.Message = "Updating compute resources"
	// }
}

func isImagesChanged(old, new *v1alpha1.Emqx) bool {
	return old.Spec.Image == new.Spec.Image
}

func isScalingDown(old, new *v1alpha1.Emqx) bool {
	return *old.Spec.Replicas > *new.Spec.Replicas
}

func isScalingUp(old, new *v1alpha1.Emqx) bool {
	return *old.Spec.Replicas < *new.Spec.Replicas
}

// func isResourcesChange(old, new *v1alpha1.Emqx) bool {
// 	return old.Spec.Resources.Limits.Memory().Size() != new.Spec.Resources.Limits.Memory().Size() ||
// 		old.Spec.Resources.Limits.Cpu().Size() != new.Spec.Resources.Limits.Cpu().Size()
// }

func getNamespacedName(nameSpace, name string) string {
	return fmt.Sprintf("%s%c%s", nameSpace, '/', name)
}
