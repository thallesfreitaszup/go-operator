package common

import (
	"fmt"
	iocharlescdv1 "github.com/thalleslmF/go-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/pointer"
	"strings"
)

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
