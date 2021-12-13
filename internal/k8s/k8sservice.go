package k8s

import (
	"context"
	"fmt"
	"github.com/prometheus/common/log"
	"github.com/thalleslmF/go-operator/internal/common"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

type DynamicService struct {
	Client dynamic.Interface
}

func (s DynamicService) Create(resource unstructured.Unstructured) error {

	resSchema := common.GetGroupVersion(resource)
	alreadyCreatedResource, err := s.GetResource(resource)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	if alreadyCreatedResource != nil {
		log.Info(fmt.Sprintf("Resource %s/%s already created", resource.GroupVersionKind(), resource.GetName()))
		return nil
	}
	_, err = s.Client.Resource(resSchema).Namespace(resource.GetNamespace()).Create(context.TODO(), &resource, v1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (s DynamicService) GetResource(resource unstructured.Unstructured) (*unstructured.Unstructured, error) {
	resSchema := common.GetGroupVersion(resource)
	return s.Client.Resource(resSchema).Namespace(resource.GetNamespace()).Get(context.TODO(), resource.GetName(), v1.GetOptions{})
}
