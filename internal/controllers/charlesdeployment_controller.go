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
	"fmt"
	"github.com/prometheus/common/log"
	iocharlescdv1 "github.com/thalleslmF/go-operator/api/v1"
	"github.com/thalleslmF/go-operator/internal/common"
	"github.com/thalleslmF/go-operator/internal/k8s"
	"github.com/thalleslmF/go-operator/internal/kustomize"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	_ "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/dynamic/dynamiclister"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CharlesDeploymentController reconciles a CharlesDeployment object
type CharlesDeploymentController struct {
	client.Client
	Scheme                 *runtime.Scheme
	Informers              map[string]cache.SharedIndexInformer
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
		For(&iocharlescdv1.CharlesDeployment{}).
		Complete(cd)
}

func (cd *CharlesDeploymentController) Sync(key client.ObjectKey) error {
	charlesDeployment := &iocharlescdv1.CharlesDeployment{}

	err := cd.Get(context.TODO(), key, charlesDeployment)
	log.Info("Start reconcile for ", charlesDeployment)
	if err != nil {
		return err
	}
	err = cd.SyncComponents(*charlesDeployment)
	if err != nil {
		return err
	}
	return nil
}

func (cd *CharlesDeploymentController) getNotSyncedComponents(components []iocharlescdv1.Component) []iocharlescdv1.Component {
	notSyncComponents := make([]iocharlescdv1.Component, 0)
	for _, component := range components {
		if component.ChildResources == nil || len(component.ChildResources) == 0 || cd.NotSyncChildren(component) {
			notSyncComponents = append(notSyncComponents, component)
		}

	}
	return notSyncComponents
}

func (cd *CharlesDeploymentController) NotSyncChildren(component iocharlescdv1.Component) bool {
	//for _, child := range component.ChildResources {
	//	if r.Client.Get()
	//}
	return true
}

func (cd *CharlesDeploymentController) SyncComponents(charlesDeployment iocharlescdv1.CharlesDeployment) error {
	for _, component := range charlesDeployment.Spec.Components {
		err := cd.createCharlesComponent(component, charlesDeployment)
		if err != nil {
			log.Info("Error creating charles component", err)
			return err
		}
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

func (cd *CharlesDeploymentController) createCharlesComponent(component iocharlescdv1.Component, charlesDeployment iocharlescdv1.CharlesDeployment) error {
	var unstructured unstructured.Unstructured
	kustomizeWrapper := kustomize.New()
	response, err := kustomizeWrapper.RenderManifests(component.Chart)
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
		common.CreateOwnerReference(&unstructured, charlesDeployment)
		err = cd.DynamicService.Create(unstructured)
		if err != nil {
			return err
		}
		cd.createInformerForResource(unstructured)
		if err != nil {
			return err
		}
	}
	return nil
}

func (cd *CharlesDeploymentController) createInformerForResource(u unstructured.Unstructured) {
	schema := common.GetGroupVersion(u)
	_, ok := cd.Informers[schema.String()]
	if !ok {
		log.Info("Creating informer for resource ", schema)
		cd.Informers[schema.String()] = common.BuildInformerForResource(cd.DynamicClient, schema, context.TODO())
		createdInformer, _ := cd.Informers[schema.String()]
		createdInformer.AddEventHandler(cd.buildInformerHandler())
		go createdInformer.Run(context.TODO().Done())
		sync := cache.WaitForCacheSync(context.TODO().Done(), createdInformer.HasSynced)
		if !sync {
			log.Error("Failed to sync controller ")
		}
	}
}

func (cd *CharlesDeploymentController) buildInformerHandler() cache.ResourceEventHandler {
	return &cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			charlesDeployment := &iocharlescdv1.CharlesDeployment{}
			resource := obj.(*unstructured.Unstructured)
			log.Info(fmt.Sprintf("Resource %s/%s created", resource.GroupVersionKind(), resource.GetName()))
			ownerRefs := resource.GetOwnerReferences()
			for _, ownerRef := range ownerRefs {
				if ownerRef.Kind == "CharlesDeployment" {
					err := cd.Get(context.TODO(), client.ObjectKey{Namespace: resource.GetNamespace(), Name: ownerRef.Name}, charlesDeployment)
					if err != nil {
						log.Error("Error getting charlesDeployment", err)
						return
					}
					key, err := cache.MetaNamespaceKeyFunc(charlesDeployment)
					if err != nil {
						log.Error("Error getting key", err)
						return
					}
					fmt.Printf("Resource %s queued\n", key)
					cd.Queue.Add(key)
				}
			}
		},
		DeleteFunc: func(obj interface{}) {
			charlesDeployment := &iocharlescdv1.CharlesDeployment{}
			resource := obj.(*unstructured.Unstructured)
			ownerRefs := resource.GetOwnerReferences()
			log.Info(fmt.Sprintf("Resource %s/%s deleted", resource.GroupVersionKind(), resource.GetName()))
			for _, ownerRef := range ownerRefs {
				if ownerRef.Kind == "CharlesDeployment" {

					err := cd.Get(context.TODO(), client.ObjectKey{Namespace: resource.GetNamespace(), Name: ownerRef.Name}, charlesDeployment)
					if err != nil {
						log.Error("Error getting charlesDeployment", err)
						return
					}
					key, err := cache.MetaNamespaceKeyFunc(charlesDeployment)
					if err != nil {
						log.Error("Error getting key", err)
						return
					}
					fmt.Printf("Resource %s queued\n", key)
					cd.Queue.Add(key)
				}
			}
		},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			charlesDeployment := &iocharlescdv1.CharlesDeployment{}
			resource := oldObj.(*unstructured.Unstructured)
			ownerRefs := resource.GetOwnerReferences()
			log.Info(fmt.Sprintf("Resource %s/%s updated", resource.GroupVersionKind(), resource.GetName()))
			for _, ownerRef := range ownerRefs {
				if ownerRef.Kind == "CharlesDeployment" {
					err := cd.Get(context.TODO(), client.ObjectKey{Namespace: resource.GetNamespace(), Name: ownerRef.Name}, charlesDeployment)
					if err != nil {
						log.Error("Error getting charlesDeployment", err)
						return
					}
					key, err := cache.MetaNamespaceKeyFunc(charlesDeployment)
					if err != nil {
						log.Error("Error getting key", err)
						return
					}
					fmt.Printf("Resource %s queued\n", key)
					cd.Queue.Add(key)
				}
			}
		},
	}
}
