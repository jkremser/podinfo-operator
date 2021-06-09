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

	"github.com/go-logr/logr"
	"github.com/jkremser/podinfo-operator/controllers/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1alpha1 "github.com/jkremser/podinfo-operator/api/v1alpha1"
)

// PodinfoReconciler reconciles a Podinfo object
type PodinfoReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=info.podinfo-operator.io,resources=podinfoes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=info.podinfo-operator.io,resources=podinfoes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=info.podinfo-operator.io,resources=podinfoes/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=deployments,verbs=get;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;watch;list;create;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Podinfo object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *PodinfoReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	log := ctrl.Log.WithName("podinfo_controller")

	// get podinfo that triggered the event
	podinfo := &v1alpha1.Podinfo{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, podinfo)

	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("podinfo was deleted", "name", req.NamespacedName.Name, "namespace", req.NamespacedName.Namespace)
			err = r.DeleteAll(req.NamespacedName, log)
			return ctrl.Result{}, err
		}
		// Error reading the object - requeue the request -> can loop
		return ctrl.Result{}, err
	}

	// create deployment and service for backend
	err = r.CreateIfNotExist(podinfo, true, log)
	if err != nil {
		log.Error(err, "Unable to deploy backend for podinfo")
		return ctrl.Result{}, err
	}

	// create deployment and service for frontend
	err = r.CreateIfNotExist(podinfo, false, log)
	if err != nil {
		log.Error(err, "Unable to deploy frontend for podinfo")
		return ctrl.Result{}, err
	}
	// Don't requeue
	return ctrl.Result{}, nil
}

func (r *PodinfoReconciler) CreateIfNotExist(podinfo *v1alpha1.Podinfo, backend bool, log logr.Logger) error {
	imgSuffix := "-fe"
	if backend {
		imgSuffix = "-be"
	}

	// deployment
	deploymentFound := &appsv1.Deployment{}
	err := r.Get(context.TODO(), types.NamespacedName{
		Name:      podinfo.Name + imgSuffix,
		Namespace: podinfo.Namespace,
	}, deploymentFound)

	if err != nil && errors.IsNotFound(err) {
		// let's create a new one <this uses yaml as a template, but failed during the client.create>
		// frontendDeployment, e := utils.GetDeployment(podInfo.Name+imgSuffix, "podinfo-operator-system", int32(podInfo.Spec.FrontendReplicas), podInfo.Spec.Message)

		if backend {
			log.Info("podinfo was created", "name", podinfo.Name, "namespace", podinfo.Namespace)
		}
		deployment := utils.PodinfoDeployment(podinfo, backend)
		// fmt.Printf("%+v\n", deployment)

		e := r.Create(context.TODO(), deployment)
		if e != nil {
			log.Error(err, "Failed to create new Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
			return err
		}
	} else if err == nil { // change in podinfo custom resource
		if backend {
			log.Info("podinfo was changed", "name", podinfo.Name, "namespace", podinfo.Namespace)
		}
		deployment := utils.PodinfoDeployment(podinfo, backend)
		r.Update(context.TODO(), deployment)
	} else if err != nil {
		log.Error(err, "Failed to get Deployment")
		return err
	}

	// service
	svcFound := &corev1.Service{}
	err = r.Get(context.TODO(), types.NamespacedName{
		Name:      podinfo.Name + imgSuffix,
		Namespace: podinfo.Namespace,
	}, svcFound)

	if err != nil && errors.IsNotFound(err) {
		svc := utils.PodinfoService(podinfo, backend)
		// fmt.Printf("%+v\n", svc)

		e := r.Create(context.TODO(), svc)
		if e != nil {
			log.Error(err, "Failed to create new Service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
			return err
		}
		return nil

	} else if err == nil {
		log.Info("Service is already there, no need to change it")
	} else if err != nil {
		log.Error(err, "Failed to get Service")
		return err
	}
	return nil
}

func (r *PodinfoReconciler) DeleteAll(nn types.NamespacedName, log logr.Logger) error {
	for _, suffix := range [2]string{"-fe", "-be"} {
		commonMeta := metav1.ObjectMeta{
			Name:      nn.Name + suffix,
			Namespace: nn.Namespace,
		}
		err := r.Delete(context.TODO(), &corev1.Service{
			ObjectMeta: commonMeta,
		})
		if err != nil {
			log.Error(err, "Unable to delete service", "service.name", nn.Name)
			return err
		}

		err = r.Delete(context.TODO(), &appsv1.Deployment{
			ObjectMeta: commonMeta,
		})
		if err != nil {
			log.Error(err, "Unable to delete deployment", "deployment.name", nn.Name)
			return err
		}
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodinfoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Podinfo{}).
		// Owns(&appsv1.Deployment{}).
		// Owns(&corev1.Service{}).
		Complete(r)
}
