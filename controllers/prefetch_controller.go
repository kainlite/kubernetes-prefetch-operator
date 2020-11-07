/*


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
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/common/log"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cachev1 "github.com/kainlite/kubernetes-prefetch-operator/api/v1"
)

// PrefetchReconciler reconciles a Prefetch object
type PrefetchReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// I have been rather permissive than restrictive here, so be aware of that when using this
// +kubebuilder:rbac:groups=cache.techsquad.rocks,resources=prefetches,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cache.techsquad.rocks,resources=cache,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cache.techsquad.rocks,resources=prefetches/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cache.techsquad.rocks,resources=pods/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *PrefetchReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	reqLogger := r.Log.WithValues("prefetch", req.NamespacedName)

	prefetch := &cachev1.Prefetch{}
	fmt.Printf("Labels %+v", prefetch)

	err := r.Client.Get(context.TODO(), req.NamespacedName, prefetch)
	if err != nil {
		log.Error(err, "failed to get Prefetch resource")
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after
			// reconcile request—return and don't requeue:
			return reconcile.Result{}, client.IgnoreNotFound(err)
		}
		// Error reading the object—requeue the request:
		return reconcile.Result{}, err
	}

	reqLogger.Info("Labels %+v", prefetch.FilterByLabels)
	reqLogger.Info("RetryAfter %+v", prefetch.RetryAfter)
	// reqLogger.Info("RetryAfter %+v", prefetch.RetryAfter)
	// retryAfter := prefetch.RetryAfter
	// if retry == nil {
	//	retry := 60
	// }

	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	reqLogger.Info("Fetching deployments")
	pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))
	// reqLogger.Info("Labels %+v", prefetch.Labels)
	// _ = r.Log.WithValues("Time to wait %+v", prefetch.Retry)

	// return ctrl.Result{RequeueAfter: time.Second * prefetch.Retry}, nil
	// strconv.Itoa(cr.Spec.Retry)
	return ctrl.Result{RequeueAfter: time.Second * 60}, nil
}

func (r *PrefetchReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cachev1.Prefetch{}).
		Complete(r)
}
