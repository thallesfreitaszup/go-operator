package common

import (
	"context"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"
	"time"
)

func BuildInformerForResource(client dynamic.Interface, gvr schema.GroupVersionResource, namespace string, resyncPeriod time.Duration, context context.Context) cache.SharedIndexInformer {
	fmt.Println(gvr)
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
				return client.Resource(gvr).Namespace(namespace).List(context, opts)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return client.Resource(gvr).Namespace(namespace).Watch(context, options)
			},
		},
		&unstructured.Unstructured{},
		resyncPeriod,
		cache.Indexers{
			cache.NamespaceIndex: cache.MetaNamespaceIndexFunc,
		},
	)
}
