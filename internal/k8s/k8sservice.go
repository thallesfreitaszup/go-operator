package k8s

import (
	"context"
	"fmt"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"strings"
)

type DynamicService struct {
	Client dynamic.Interface
}

func (s DynamicService) Create(resource unstructured.Unstructured) error {

	resSchema := s.GetGroupVersion(resource)

	_, err := s.Client.Resource(resSchema).Create(context.TODO(), &resource, v1.CreateOptions{}, "")
	if err != nil {
		return err
	}
	return nil
}

func (s DynamicService) GetGroupVersion(resource unstructured.Unstructured) schema.GroupVersionResource {
	plural := fmt.Sprintf("%s%s", strings.ToLower(resource.GetKind()), "s")
	return schema.GroupVersionResource{Version: resource.GroupVersionKind().Version, Group: resource.GroupVersionKind().Group, Resource: plural}
}
