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
	"fmt"

	"github.com/jkremser/podinfo-operator/controllers/utils"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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
	log := ctrl.Log.WithName("test")

	fmt.Printf("working..")
	podInfo := &v1alpha1.Podinfo{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, podInfo)

	if err != nil {
		if errors.IsNotFound(err) {
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	frontendFound := &appsv1.Deployment{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{
		Name:      podInfo.Name + "-fe",
		Namespace: podInfo.Namespace,
	}, frontendFound)

	if err != nil && errors.IsNotFound(err) {
		// let's create a new one
		frontendDeployment, e := utils.GetDeployment(podInfo.Name+"-fe", podInfo.Namespace, int32(podInfo.Spec.FrontendReplicas), podInfo.Spec.Message)

		fmt.Printf("%+v\n", frontendDeployment)
		fmt.Printf("%s", frontendDeployment.Namespace)

		if e != nil {
			log.Error(err, "Failed to read Deployment from yaml")
			return ctrl.Result{}, e
		}

		e = r.Client.Create(context.TODO(), frontendDeployment)
		if e != nil {
			log.Error(err, "Failed to create new Deployment", "Deployment.Namespace", frontendDeployment.Namespace, "Deployment.Name", frontendDeployment.Name)
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil

	} else if err != nil {
		log.Error(err, "Failed to get Deployment")
		return ctrl.Result{}, err
	}
	// no change (todo: check the changed fields)
	return ctrl.Result{}, nil
	//   backendFound := &appsv1.Deployment{}

	// your logic here
	// testLog := ctrl.Log.WithName("test")
	// testLog.Info("Reconcile")

	// https://github.com/stefanprodan/podinfo/blob/master/deploy/webapp/frontend/deployment.yaml
	// https://github.com/stefanprodan/podinfo/blob/master/deploy/webapp/backend/deployment.yaml
	// + svcs

	// env for custom ui msg:
	// - name: PODINFO_UI_MESSAGE
	//   value: "hello world"

	// return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodinfoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Podinfo{}).
		Complete(r)
}
