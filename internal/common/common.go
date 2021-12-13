package common

import (
	"context"
	"fmt"
	iocharlescdv1 "github.com/thalleslmF/go-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/pointer"
	"strings"
	"time"
)

func BuildInformerForResource(client dynamic.Interface, gvr schema.GroupVersionResource, context context.Context) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
				return client.Resource(gvr).List(context, opts)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return client.Resource(gvr).Watch(context, options)
			},
		},
		&unstructured.Unstructured{},
		time.Minute,
		cache.Indexers{},
	)
}
func CreateOwnerReference(u *unstructured.Unstructured, deployment iocharlescdv1.CharlesDeployment) {
	newOwnerReference := metav1.OwnerReference{
		APIVersion:         deployment.APIVersion,
		Name:               deployment.Name,
		Kind:               deployment.Kind,
		UID:                deployment.GetUID(),
		Controller:         pointer.Bool(true),
		BlockOwnerDeletion: pointer.Bool(true),
	}
	ownerReferences := u.GetOwnerReferences()
	ownerReferences = append(ownerReferences, newOwnerReference)
	u.SetOwnerReferences(ownerReferences)
}

func GetGroupVersion(resource unstructured.Unstructured) schema.GroupVersionResource {
	plural := fmt.Sprintf("%s%s", strings.ToLower(resource.GetKind()), "s")

	return schema.GroupVersionResource{Version: resource.GroupVersionKind().Version, Group: resource.GroupVersionKind().Group, Resource: plural}
}
