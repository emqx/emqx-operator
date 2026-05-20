package controller

import (
	"context"
	"math/rand"
	"reflect"
	"sync"

	emperror "emperror.dev/errors"
	ginkgo "github.com/onsi/ginkgo/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

type faultEmitter struct {
	mu              sync.Mutex
	remainingEvents int
	probability     float64
}

var (
	faultRandomMu sync.Mutex
	faultRandom   *rand.Rand
)

func newFaultyClient(
	base client.WithWatch,
	numEvents int,
	faultProbability float64,
) client.Client {
	fault := newFaultEmitter(numEvents, faultProbability)
	if fault == nil {
		return base
	}
	return interceptor.NewClient(base, interceptor.Funcs{
		Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
			if err := fault.err(); err != nil {
				return err
			}
			return c.Get(ctx, key, obj, opts...)
		},
		Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
			if err := fault.err(); err != nil {
				return err
			}
			return c.Create(ctx, obj, opts...)
		},
		Delete: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
			if err := fault.err(); err != nil {
				return err
			}
			return c.Delete(ctx, obj, opts...)
		},
		Update: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
			if err := fault.err(); err != nil {
				restoreObject(ctx, c, obj)
				return err
			}
			return c.Update(ctx, obj, opts...)
		},
	})
}

func newFaultEmitter(numEvents int, faultProbability float64) *faultEmitter {
	if numEvents <= 0 {
		return nil
	}
	if faultProbability <= 0 {
		return nil
	}
	return &faultEmitter{
		remainingEvents: numEvents,
		probability:     faultProbability,
	}
}

func (f *faultEmitter) err() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.remainingEvents == 0 {
		return nil
	}
	f.remainingEvents--
	if faultRandomFloat64() < f.probability {
		err := emperror.WithStackDepth(emperror.NewPlain("injected k8s client error"), 3)
		logger.Info("injecting fault", "error", err)
		return err
	}
	return nil
}

func faultRandomFloat64() float64 {
	faultRandomMu.Lock()
	defer faultRandomMu.Unlock()
	if faultRandom == nil {
		faultRandom = rand.New(rand.NewSource(ginkgo.GinkgoRandomSeed()))
	}
	return faultRandom.Float64()
}

func restoreObject(ctx context.Context, c client.Client, obj client.Object) {
	stored, ok := obj.DeepCopyObject().(client.Object)
	if !ok {
		return
	}
	if err := c.Get(ctx, client.ObjectKeyFromObject(obj), stored); err != nil {
		return
	}
	reflect.ValueOf(obj).Elem().Set(reflect.ValueOf(stored).Elem())
}
