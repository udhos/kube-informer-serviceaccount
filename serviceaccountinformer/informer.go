// Package serviceaccountinformer implements a service account discovery helper.
package serviceaccountinformer

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// Options define config for informer.
type Options struct {
	// Client provides Clientset.
	Client *kubernetes.Clientset

	// Restrict namespace.
	Namespace string

	// LabelSelector restricts serviceAccounts by label.
	// Empty LabelSelector matches everything.
	// Example: "app=miniapi,tier=backend"
	LabelSelector string

	// OnUpdate is required callback function for serviceAccount discovery.
	OnUpdate func(serviceAccounts []ServiceAccount)

	ResyncPeriod time.Duration
}

// ServiceAccount holds information about discovered service account.
type ServiceAccount struct {
	Namespace   string
	Name        string
	Annotations map[string]string
}

// ServiceAccountInformer holds informer state.
type ServiceAccountInformer struct {
	options   Options
	stopCh    chan struct{}
	cancelCtx context.Context
	cancel    func()
	informer  cache.SharedIndexInformer
}

// New creates an informer.
func New(options Options) *ServiceAccountInformer {

	if options.OnUpdate == nil {
		panic("Options.OnUpdate is nil")
	}

	ctx, cancel := context.WithCancel(context.Background())

	i := &ServiceAccountInformer{
		options:   options,
		stopCh:    make(chan struct{}),
		cancelCtx: ctx,
		cancel:    cancel,
	}

	return i
}

// Run runs the informer.
func (i *ServiceAccountInformer) Run() error {

	const me = "ServiceAccountInformer.Run"

	listWatch := &cache.ListWatch{
		ListFunc: func(options meta_v1.ListOptions) (runtime.Object, error) {
			options.LabelSelector = i.options.LabelSelector
			return i.options.Client.CoreV1().ServiceAccounts(i.options.Namespace).List(i.cancelCtx, options)
		},
		WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
			options.LabelSelector = i.options.LabelSelector
			return i.options.Client.CoreV1().ServiceAccounts(i.options.Namespace).Watch(i.cancelCtx, options)
		},
	}

	i.informer = cache.NewSharedIndexInformer(
		listWatch,
		&core_v1.ServiceAccount{},
		i.options.ResyncPeriod,
		cache.Indexers{},
	)

	i.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			debugf("%s: add: '%s': error:%v", me, key, err)
			i.update()
		},
		UpdateFunc: func(obj, _ any) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			debugf("%s: update: '%s': error:%v", me, key, err)
			i.update()
		},
		DeleteFunc: func(obj any) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			debugf("%s: delete: '%s': error:%v", me, key, err)
			i.update()
		},
	})

	i.informer.Run(i.stopCh)

	return nil
}

func debugf(format string, v ...any) {
	slog.Debug(fmt.Sprintf(format, v...))
}

func errorf(format string, v ...any) {
	slog.Info(fmt.Sprintf(format, v...))
}

func (i *ServiceAccountInformer) update() {

	const me = "ServiceAccountInformer.update"

	list := i.informer.GetStore().List()
	size := len(list)

	debugf("%s: listing service accounts: %d", me, size)

	serviceAccounts := make([]ServiceAccount, 0, size)

	for _, obj := range list {
		sa, ok := obj.(*core_v1.ServiceAccount)
		if !ok {
			errorf("%s: unexpected object type: %T", me, obj)
			continue
		}
		p := ServiceAccount{
			Namespace:   sa.Namespace,
			Name:        sa.Name,
			Annotations: sa.Annotations,
		}
		serviceAccounts = append(serviceAccounts, p)
	}

	i.options.OnUpdate(serviceAccounts)
}

// Stop stops the informer to release resources.
func (i *ServiceAccountInformer) Stop() {
	i.cancel()
	close(i.stopCh)
}
