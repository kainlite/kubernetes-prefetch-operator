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
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cachev1 "github.com/kainlite/kubernetes-prefetch-operator/api/v1"
)

// PrefetchReconciler reconciles a Prefetch object
type PrefetchReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=cache.techsquad.rocks,resources=prefetches,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cache.techsquad.rocks,resources=cache,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cache.techsquad.rocks,resources=prefetches/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cache.techsquad.rocks,resources=pods/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete

func (r *PrefetchReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("prefetch", req.NamespacedName)

	prefetch := &cachev1.Prefetch{}
	err := r.Get(context.TODO(), req.NamespacedName, prefetch)
	if err != nil {
		panic(err.Error())
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	reqLogger := r.Log.WithValues("namespace", req.Namespace, "MapForward", req.Name)
	reqLogger.Info("=== Reconciling Forward Map")

	pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	_ = r.Log.WithValues("There are %d pods in the cluster\n", len(pods.Items))
	_ = r.Log.WithValues("Labels %+v", prefetch.Labels)
	_ = r.Log.WithValues("Time to wait %+v", prefetch.WaitInSeconds)

	return ctrl.Result{RequeueAfter: time.Second * prefetch.WaitInSeconds}, nil
}

func (r *PrefetchReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cachev1.Prefetch{}).
		Complete(r)
}
