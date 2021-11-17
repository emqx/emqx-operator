package cache

import (
	"fmt"
	"sync"

	"github.com/emqx/emqx-operator/api/v1alpha2"
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
	Obj       v1alpha2.Emqx

	Status  v1alpha2.ConditionType
	Message string

	Config map[string]string
}

func newCluster(emqx v1alpha2.Emqx) *Meta {
	return &Meta{
		Status: v1alpha2.ClusterConditionCreating,
		// Config:    e.Spec.Config,
		Obj:       emqx,
		Size:      *emqx.GetReplicas(),
		State:     Create,
		Name:      emqx.GetName(),
		NameSpace: emqx.GetNamespace(),
		Message:   "Bootstrap emqx cluster",
	}
}

// MetaMap cache last EMQ X Cluster and meta data
type MetaMap struct {
	sync.Map
}

func (c *MetaMap) Cache(obj v1alpha2.Emqx) *Meta {
	meta, ok := c.Load(getNamespacedName(obj.GetNamespace(), obj.GetName()))
	if !ok {
		c.Add(obj)
	} else {
		c.Update(meta.(*Meta), obj)
	}
	return c.Get(obj)
}

func (c *MetaMap) Get(obj v1alpha2.Emqx) *Meta {
	meta, _ := c.Load(getNamespacedName(obj.GetNamespace(), obj.GetName()))
	return meta.(*Meta)
}

func (c *MetaMap) Add(obj v1alpha2.Emqx) {
	c.Store(getNamespacedName(obj.GetNamespace(), obj.GetName()), newCluster(obj))
}

func (c *MetaMap) Del(obj v1alpha2.Emqx) {
	c.Delete(getNamespacedName(obj.GetNamespace(), obj.GetName()))
}

func (c *MetaMap) Update(meta *Meta, new v1alpha2.Emqx) {
	if meta.Obj.GetGeneration() == new.GetGeneration() {
		meta.State = Check
		return
	}

	old := meta.Obj
	meta.State = Update
	meta.Size = *old.GetReplicas()
	// Password change is not allowed
	// new.Spec.Password = old.Spec.Password
	// meta.Auth.Password = old.Spec.Password
	meta.Obj = new

	meta.Status = v1alpha2.ClusterConditionUpdating
	meta.Message = "Updating emqx config"
	if isImagesChanged(old, new) {
		meta.Status = v1alpha2.ClusterConditionUpgrading
		meta.Message = fmt.Sprintf("Upgrading to %s", new.GetImage())
	}
	if isScalingDown(old, new) {
		meta.Status = v1alpha2.ClusterConditionScalingDown
		meta.Message = fmt.Sprintf("Scaling down form: %d to: %d", meta.Size, new.GetReplicas())
	}
	if isScalingUp(old, new) {
		meta.Status = v1alpha2.ClusterConditionScaling
		meta.Message = fmt.Sprintf("Scaling up form: %d to: %d", meta.Size, new.GetReplicas())
	}
	// if isResourcesChange(old, new) {
	// 	meta.Message = "Updating compute resources"
	// }
}

func isImagesChanged(old, new v1alpha2.Emqx) bool {
	return old.GetImage() == new.GetImage()
}

func isScalingDown(old, new v1alpha2.Emqx) bool {
	return *old.GetReplicas() > *new.GetReplicas()
}

func isScalingUp(old, new v1alpha2.Emqx) bool {
	return *old.GetReplicas() < *new.GetReplicas()
}

// func isResourcesChange(old, new *v1alpha2.EmqxBroker) bool {
// 	return old.Spec.Resources.Limits.Memory().Size() != new.Spec.Resources.Limits.Memory().Size() ||
// 		old.Spec.Resources.Limits.Cpu().Size() != new.Spec.Resources.Limits.Cpu().Size()
// }

func getNamespacedName(nameSpace, name string) string {
	return fmt.Sprintf("%s%c%s", nameSpace, '/', name)
}
