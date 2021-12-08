/*
Copyright 2021.

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

package controllers

import (
	"context"
	"errors"
	"github.com/prometheus/common/log"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	_ "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/dynamic/dynamiclister"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	iocharlescdv1beta1 "operator-sdk/api/v1"
	"operator-sdk/internal/k8s"
	"operator-sdk/internal/kustomize"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CharlesDeploymentController reconciles a CharlesDeployment object
type CharlesDeploymentController struct {
	client.Client
	Scheme                 *runtime.Scheme
	Informers              map[string]informers.GenericInformer
	Queue                  workqueue.RateLimitingInterface
	DynamicClient          dynamic.Interface
	DynamicService         k8s.DynamicService
	DynamicInformerFactory dynamicinformer.DynamicSharedInformerFactory
	ChildInformerHandler   cache.ResourceEventHandler
	CharlesLister          dynamiclister.Lister
}

//+kubebuilder:rbac:groups=io.charlescd.my.domain,resources=charlesdeployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=io.charlescd.my.domain,resources=charlesdeployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=io.charlescd.my.domain,resources=charlesdeployments/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the CharlesDeployment object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (cd *CharlesDeploymentController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	err := cd.Sync(req.NamespacedName)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (cd *CharlesDeploymentController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&iocharlescdv1beta1.CharlesDeployment{}).
		Complete(cd)
}

func (cd *CharlesDeploymentController) Sync(key client.ObjectKey) error {
	charlesDeployment := &iocharlescdv1beta1.CharlesDeployment{}

	err := cd.Get(context.TODO(), key, charlesDeployment)
	if err != nil {
		return err
	}
	err = cd.SyncComponents(charlesDeployment.Spec.Components)
	if err != nil {
		return err
	}
	return nil
}

func (cd *CharlesDeploymentController) getNotSyncedComponents(components []iocharlescdv1beta1.Component) []iocharlescdv1beta1.Component {
	notSyncComponents := make([]iocharlescdv1beta1.Component, 0)
	for _, component := range components {
		if component.ChildResources == nil || len(component.ChildResources) == 0 || cd.NotSyncChildren(component) {
			notSyncComponents = append(notSyncComponents, component)
		}

	}
	return notSyncComponents
}

func (cd *CharlesDeploymentController) NotSyncChildren(component iocharlescdv1beta1.Component) bool {
	//for _, child := range component.ChildResources {
	//	if r.Client.Get()
	//}
	return true
}

func (cd *CharlesDeploymentController) SyncComponents(components []iocharlescdv1beta1.Component) error {
	for _, component := range components {
		err := cd.createCharlesComponent(component)
		if err != nil {
			return err
		}
		//for _, v := range component.ChildResources {
		//	registerChildInformer(v)
		//}
	}
	return nil
}

func (cd *CharlesDeploymentController) Start() error {
	for cd.processNextWorkItem() {
	}
	return errors.New("error processing  item")
}

func (cd *CharlesDeploymentController) processNextWorkItem() bool {
	key, stop := cd.Queue.Get()
	if stop {
		return false
	}
	defer cd.Queue.Done(key)
	namespace, name, err := cache.SplitMetaNamespaceKey(key.(string))
	if err != nil {
		log.Error("Error getting object key", err)
		return false
	}
	err = cd.Sync(client.ObjectKey{Name: name, Namespace: namespace})
	if err != nil {
		return true
	}
	cd.Queue.Forget(key)
	return true
}

func registerChildInformer(v iocharlescdv1beta1.Child) {
	//informers.NewSharedInformerFactory()
}

func (cd *CharlesDeploymentController) createCharlesComponent(component iocharlescdv1beta1.Component) error {
	var unstructured unstructured.Unstructured
	kustomizeWrapper := kustomize.New()
	response, err := kustomizeWrapper.Kustomizer.Run(kustomizeWrapper.Filesys, component.Chart)
	if err != nil {
		return err
	}
	resources := response.Resources()

	for _, resource := range resources {
		resourceBytes, err := json.Marshal(resource)
		if err != nil {
			return err
		}
		err = json.Unmarshal(resourceBytes, &unstructured)
		if err != nil {
			return err
		}
		err = cd.DynamicService.Create(unstructured)
		if err != nil {
			return err
		}
		cd.createInformerForResource(unstructured)
	}
	if err != nil {
		return err
	}
	return nil
}

func (cd *CharlesDeploymentController) createInformerForResource(u unstructured.Unstructured) {
	schema := cd.DynamicService.GetGroupVersion(u)
	informer := cd.DynamicInformerFactory.ForResource(schema)
	informer.Informer().AddEventHandler(cd.childInformerHandler())
}

func (cd *CharlesDeploymentController) childInformerHandler() cache.ResourceEventHandler {
	return &cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			resource := obj.(unstructured.Unstructured)
			ownerRefs := resource.GetOwnerReferences()

			for _, ownerRef := range ownerRefs {
				if ownerRef.Kind == "CharlesDeployment" {
					charlesDeployment, err := cd.CharlesLister.Get(ownerRef.Name)
					if err != nil {
						return
					}
					key, err := cache.MetaNamespaceKeyFunc(charlesDeployment)
					if err != nil {
						return
					}
					cd.Queue.Add(key)
				}
			}
		},
		DeleteFunc: func(obj interface{}) {
			resource := obj.(unstructured.Unstructured)
			ownerRefs := resource.GetOwnerReferences()

			for _, ownerRef := range ownerRefs {
				if ownerRef.Kind == "CharlesDeployment" {
					charlesDeployment, err := cd.CharlesLister.Get(ownerRef.Name)
					if err != nil {
						return
					}
					key, err := cache.MetaNamespaceKeyFunc(charlesDeployment)
					if err != nil {
						return
					}
					cd.Queue.Add(key)
				}
			}
		},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			resource := oldObj.(unstructured.Unstructured)
			ownerRefs := resource.GetOwnerReferences()

			for _, ownerRef := range ownerRefs {
				if ownerRef.Kind == "CharlesDeployment" {
					charlesDeployment, err := cd.CharlesLister.Get(ownerRef.Name)
					if err != nil {
						return
					}
					key, err := cache.MetaNamespaceKeyFunc(charlesDeployment)
					if err != nil {
						return
					}
					cd.Queue.Add(key)
				}
			}
		},
	}
}
