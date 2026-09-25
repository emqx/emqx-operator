package controller

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// Clones the given selector and returns a new selector with the given key and value added.
// Returns the given selector, if labelKey is empty.
func CloneSelectorAndAddLabel(selector *metav1.LabelSelector, labelKey, labelValue string) *metav1.LabelSelector {
	// Clone.
	newSelector := new(metav1.LabelSelector)

	// TODO(madhusudancs): Check if you can use deepCopy_extensions_LabelSelector here.
	newSelector.MatchLabels = make(map[string]string)
	if selector.MatchLabels != nil {
		for key, val := range selector.MatchLabels {
			newSelector.MatchLabels[key] = val
		}
	}
	newSelector.MatchLabels[labelKey] = labelValue

	if selector.MatchExpressions != nil {
		newMExps := make([]metav1.LabelSelectorRequirement, len(selector.MatchExpressions))
		for i, me := range selector.MatchExpressions {
			newMExps[i].Key = me.Key
			newMExps[i].Operator = me.Operator
			if me.Values != nil {
				newMExps[i].Values = make([]string, len(me.Values))
				copy(newMExps[i].Values, me.Values)
			} else {
				newMExps[i].Values = nil
			}
		}
		newSelector.MatchExpressions = newMExps
	} else {
		newSelector.MatchExpressions = nil
	}

	return newSelector
}

func CloneAnnotations(annotations map[string]string) map[string]string {
	clone := make(map[string]string)
	for k, v := range annotations {
		clone[k] = v
	}
	return clone
}

func AttachAnnotation(object metav1.Object, name, value string) bool {
	return AttachAnnotations(object, map[string]string{name: value})
}

func AttachAnnotations(object metav1.Object, annotations map[string]string) bool {
	dirty := false
	attached := object.GetAnnotations()
	if len(annotations) > 0 && attached == nil {
		attached = make(map[string]string, len(annotations))
	}
	for name, value := range annotations {
		valueWas, present := attached[name]
		if !present || valueWas != value {
			attached[name] = value
			dirty = true
		}
	}
	if dirty {
		object.SetAnnotations(attached)
	}
	return dirty
}

func PeekAnnotations(object metav1.Object, names ...string) map[string]string {
	peeked := make(map[string]string, len(names))
	annotations := object.GetAnnotations()
	for _, name := range names {
		if value, present := annotations[name]; present {
			peeked[name] = value
		}
	}
	return peeked
}

func UnsetAnnotations(object metav1.Object, names ...string) bool {
	dirty := false
	annotations := object.GetAnnotations()
	for _, name := range names {
		if _, present := annotations[name]; present {
			dirty = true
			delete(annotations, name)
		}
	}
	if dirty {
		object.SetAnnotations(annotations)
	}
	return dirty
}

func IsManagedBy(object metav1.Object, manager metav1.Object) bool {
	controller := metav1.GetControllerOf(object)
	if controller != nil && controller.UID == manager.GetUID() {
		return true
	}
	return false
}
