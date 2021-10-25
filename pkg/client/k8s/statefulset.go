package k8s

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// StatefulSet the StatefulSet client that knows how to interact with kubernetes to manage them
type StatefulSet interface {
	// GetStatefulSet get StatefulSet from kubernetes with namespace and name
	GetStatefulSet(namespace, name string) (*appsv1.StatefulSet, error)
	// GetStatefulSetPods will retrieve the pods managed by a given StatefulSet
	GetStatefulSetPods(namespace, name string) (*corev1.PodList, error)
	// CreateStatefulSet will create the given StatefulSet
	CreateStatefulSet(namespace string, statefulSet *appsv1.StatefulSet) error
	// UpdateStatefulSet will update the given StatefulSet
	UpdateStatefulSet(namespace string, statefulSet *appsv1.StatefulSet) error
	// CreateOrUpdateStatefulSet will update the given StatefulSet or create it if does not exist
	CreateOrUpdateStatefulSet(namespace string, StatefulSet *appsv1.StatefulSet) error
	// DeleteStatefulSet will delete the given StatefulSet
	DeleteStatefulSet(namespace string, name string) error
	// ListStatefulSets get set of StatefulSet on a given namespace
	ListStatefulSets(namespace string) (*appsv1.StatefulSetList, error)
	CreateIfNotExistsStatefulSet(namespace string, statefulSet *appsv1.StatefulSet) error
}

// StatefulSetOption is the StatefulSet client implementation using API calls to kubernetes.
type StatefulSetOption struct {
	client client.Client
	logger logr.Logger
}

// NewStatefulSet returns a new StatefulSet client.
func NewStatefulSet(kubeClient client.Client, logger logr.Logger) StatefulSet {
	logger = logger.WithValues("service", "k8s.statefulSet")
	return &StatefulSetOption{
		client: kubeClient,
		logger: logger,
	}
}

// GetStatefulSet implement the StatefulSet.Interface
func (s *StatefulSetOption) GetStatefulSet(namespace, name string) (*appsv1.StatefulSet, error) {
	statefulSet := &appsv1.StatefulSet{}
	err := s.client.Get(context.TODO(), types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, statefulSet)
	if err != nil {
		return nil, err
	}
	return statefulSet, err
}

// GetStatefulSetPods implement the StatefulSet.Interface
func (s *StatefulSetOption) GetStatefulSetPods(namespace, name string) (*corev1.PodList, error) {
	statefulSet, err := s.GetStatefulSet(namespace, name)
	if err != nil {
		return nil, err
	}

	labelSet := make(map[string]string)
	for k, v := range statefulSet.Spec.Selector.MatchLabels {
		labelSet[k] = v
	}
	labelSelector := labels.SelectorFromSet(labelSet)
	foundPods := &corev1.PodList{}
	err = s.client.List(context.TODO(), foundPods, &client.ListOptions{Namespace: namespace, LabelSelector: labelSelector})
	return foundPods, err
}

// CreateStatefulSet implement the StatefulSet.Interface
func (s *StatefulSetOption) CreateStatefulSet(namespace string, statefulSet *appsv1.StatefulSet) error {
	err := s.client.Create(context.TODO(), statefulSet)
	if err != nil {
		return err
	}
	s.logger.WithValues("namespace", namespace, "statefulSet", statefulSet.ObjectMeta.Name).Info("statefulSet created")
	return err
}

// UpdateStatefulSet implement the StatefulSet.Interface
func (s *StatefulSetOption) UpdateStatefulSet(namespace string, statefulSet *appsv1.StatefulSet) error {
	err := s.client.Update(context.TODO(), statefulSet)
	if err != nil {
		return err
	}
	s.logger.WithValues("namespace", namespace, "statefulSet", statefulSet.ObjectMeta.Name).Info("statefulSet updated")
	return err
}

// CreateOrUpdateStatefulSet implement the StatefulSet.Interface
func (s *StatefulSetOption) CreateOrUpdateStatefulSet(namespace string, statefulSet *appsv1.StatefulSet) error {
	storedStatefulSet, err := s.GetStatefulSet(namespace, statefulSet.Name)
	if err != nil {
		// If no resource we need to create.
		if errors.IsNotFound(err) {
			return s.CreateStatefulSet(namespace, statefulSet)
		}
		return err
	}

	// Already exists, need to Update.
	// Set the correct resource version to ensure we are on the latest version. This way the only valid
	// namespace is our spec(https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#concurrency-control-and-consistency),
	// we will replace the current namespace state.
	s.logger.WithValues("namespace", namespace, "storedStatefulSet", statefulSet.ObjectMeta.Name).V(5).Info(fmt.Sprintf("storedStatefulSet Spec:\n %+v", storedStatefulSet))

	statefulSet.ResourceVersion = storedStatefulSet.ResourceVersion
	s.logger.WithValues("namespace", namespace, "statefulSet", statefulSet.ObjectMeta.Name).V(5).Info(fmt.Sprintf("Stateful Spec:\n %+v", statefulSet))

	return s.UpdateStatefulSet(namespace, statefulSet)
}

// CreateIfNotExistsStatefulSet implement the StatefulSet.Interface
func (s *StatefulSetOption) CreateIfNotExistsStatefulSet(namespace string, statefulSet *appsv1.StatefulSet) error {
	_, err := s.GetStatefulSet(namespace, statefulSet.Name)
	if err != nil {
		// If no resource we need to create.
		if errors.IsNotFound(err) {
			return s.CreateStatefulSet(namespace, statefulSet)
		}
		return err
	}

	return nil
}

// DeleteStatefulSet implement the StatefulSet.Interface
func (s *StatefulSetOption) DeleteStatefulSet(namespace, name string) error {
	statefulset := &appsv1.StatefulSet{}
	if err := s.client.Get(context.TODO(), types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, statefulset); err != nil {
		return err
	}
	return s.client.Delete(context.TODO(), statefulset)
}

// ListStatefulSets implement the StatefulSet.Interface
func (s *StatefulSetOption) ListStatefulSets(namespace string) (*appsv1.StatefulSetList, error) {
	statelfulSets := &appsv1.StatefulSetList{}
	listOps := &client.ListOptions{
		Namespace: namespace,
	}
	err := s.client.List(context.TODO(), statelfulSets, listOps)
	return statelfulSets, err
}
