package k8s

import (
	"context"
	"fmt"
	"github.com/prometheus/common/log"
	"k8s.io/apimachinery/pkg/api/errors"
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
	alreadyCreatedResource, err := s.GetResource(resource)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	if alreadyCreatedResource != nil {
		log.Info("Resource already created", resource.GetName())
		return nil
	}
	_, err = s.Client.Resource(resSchema).Namespace(resource.GetNamespace()).Create(context.TODO(), &resource, v1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (s DynamicService) GetGroupVersion(resource unstructured.Unstructured) schema.GroupVersionResource {
	plural := fmt.Sprintf("%s%s", strings.ToLower(resource.GetKind()), "s")

	return schema.GroupVersionResource{Version: resource.GroupVersionKind().Version, Group: resource.GroupVersionKind().Group, Resource: plural}
}

func (s DynamicService) GetResource(resource unstructured.Unstructured) (*unstructured.Unstructured, error) {
	resSchema := s.GetGroupVersion(resource)
	return s.Client.Resource(resSchema).Namespace(resource.GetNamespace()).Get(context.TODO(), resource.GetName(), v1.GetOptions{})
}
