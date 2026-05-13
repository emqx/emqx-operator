package controller

import (
	"context"
	"errors"
	"reflect"
	"sync"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

type transientErrorClient struct {
	mu        sync.Mutex
	remaining int
}

func newTransientErrorClient(base client.WithWatch, numErrors int) client.Client {
	fault := &transientErrorClient{remaining: numErrors}
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
		SubResourceUpdate: func(ctx context.Context, c client.Client, subResourceName string, obj client.Object, opts ...client.SubResourceUpdateOption) error {
			if err := fault.err(); err != nil {
				restoreObject(ctx, c, obj)
				return err
			}
			return c.SubResource(subResourceName).Update(ctx, obj, opts...)
		},
	})
}

func (f *transientErrorClient) err() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.remaining == 0 {
		return nil
	}
	f.remaining--
	err := errors.New("injected k8s client error")
	// logger.WithCallDepth(3).Error(err, "injected error", "remaining", f.remaining)
	return err
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
