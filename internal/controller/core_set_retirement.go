/*
Copyright 2025-2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"fmt"
	"strconv"
	"time"

	crd "github.com/emqx/emqx-operator/api/v3beta1"
	util "github.com/emqx/emqx-operator/internal/controller/util"
	appsv1 "k8s.io/api/apps/v1"
)

// coreSetRetirement serializes reuse of core StatefulSet ordinals after
// scale-down. For a StatefulSet with N replicas, ordinals in [N, watermark)
// are retiring and must not be reused.
//   - Watermark equal to N means that no ordinal is awaiting retirement.
//   - Values above N may represent multiple retirements that are drained from
//     the highest ordinal down.
type coreSetRetirement struct {
	// watermark is the exclusive upper bound of the retiring ordinal range.
	// User is allowed to overwrite it as an escape hatch for stuck retirement
	// progress, implementation must anticipate this.
	watermark int32
	// updatedAt marks the beginning of the current retirement epoch for
	// retirement deadlines. It is deliberately preserved while a multi-ordinal
	// range is drained.
	updatedAt time.Time
}

var coreSetRetirementAnnotations = []string{
	crd.AnnotationCoreRetirementOrdinalWatermark,
	crd.AnnotationCoreRetirementOrdinalUpdatedAt,
}

// loadCoreSetRetirement initializes missing StatefulSet annotations and loads
// the reusable core pod ordinal state for the rest of the reconciliation round.
type loadCoreSetRetirement struct {
	*EMQXReconciler
}

func (l *loadCoreSetRetirement) reconcile(r *reconcileRound, _ *crd.EMQX) subResult {
	coreSet := r.state.coreSet()
	if coreSet == nil {
		return subResult{}
	}
	state, err := readCoreSetRetirementState(coreSet)
	if err != nil {
		return reconcileError(err)
	}
	if state == nil || state.watermark < util.NumReplicas(coreSet) {
		r.coreRetirement = initialCoreSetRetirementState(coreSet)
		util.AttachAnnotations(coreSet, r.coreRetirement.annotations())
		if err := l.Client.Update(r.ctx, coreSet); err != nil {
			return reconcileError(fmt.Errorf("failed to reinitialize retirement state: %w", err))
		}
		return subResult{}
	}

	r.coreRetirement = state
	return subResult{}
}

func loadCoreSetRetirementState(r *reconcileRound) (*coreSetRetirement, error) {
	coreSet := r.state.coreSet()
	if coreSet == nil {
		return nil, nil
	}
	state, err := readCoreSetRetirementState(coreSet)
	if err != nil {
		return nil, err
	}
	return state, nil
}

func newCoreSetRetirementState(watermark int32) *coreSetRetirement {
	return &coreSetRetirement{
		watermark: watermark,
		updatedAt: time.Now(),
	}
}

func initialCoreSetRetirementState(coreSet *appsv1.StatefulSet) *coreSetRetirement {
	return newCoreSetRetirementState(util.NumReplicas(coreSet))
}

func readCoreSetRetirementState(coreSet *appsv1.StatefulSet) (*coreSetRetirement, error) {
	var err error
	annotations := coreSet.GetAnnotations()
	watermarkString, hasWatermark := annotations[crd.AnnotationCoreRetirementOrdinalWatermark]
	updatedAtString, hasUpdatedAt := annotations[crd.AnnotationCoreRetirementOrdinalUpdatedAt]

	if !hasWatermark || !hasUpdatedAt {
		return nil, nil
	}

	watermark, err := strconv.ParseInt(watermarkString, 10, 32)
	if err != nil || watermark < 0 {
		return nil, fmt.Errorf("invalid reusable core pod ordinal watermark %q", watermarkString)
	}
	updatedAt, err := time.Parse(time.RFC3339Nano, updatedAtString)
	if err != nil {
		return nil, fmt.Errorf("invalid reusable core pod ordinal update timestamp %q", updatedAtString)
	}

	return &coreSetRetirement{watermark: int32(watermark), updatedAt: updatedAt}, nil
}

func (s *coreSetRetirement) annotations() map[string]string {
	return map[string]string{
		crd.AnnotationCoreRetirementOrdinalWatermark: strconv.FormatInt(int64(s.watermark), 10),
		crd.AnnotationCoreRetirementOrdinalUpdatedAt: s.updatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func (s *coreSetRetirement) isRetiringOrdinal(coreSet *appsv1.StatefulSet, ordinal int) bool {
	return ordinal >= int(util.NumReplicas(coreSet)) && ordinal < int(s.watermark)
}
