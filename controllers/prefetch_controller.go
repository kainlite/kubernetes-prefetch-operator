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
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/common/log"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	set "k8s.io/apimachinery/pkg/labels"

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
// +kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch

// Generate an in-cluster config
func getClientSet() (*kubernetes.Clientset, error) {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	return clientset, err
}

// Fetch all deployments and then make a list of all the images used in init containers and normal containers
func fetchImagesWithTags(clientset *kubernetes.Clientset, labels map[string]string) []string {
	list := []string{}
	labelsAsString := set.FormatLabels(labels)
	fmt.Printf("labelsAsString: %+v\n", labelsAsString)

	// List Deployments
	deploymentsClient := clientset.AppsV1().Deployments("")
	DeploymentList, err := deploymentsClient.List(context.TODO(), metav1.ListOptions{LabelSelector: labelsAsString})
	if err != nil {
		fmt.Printf("Error fetching deployments, check your labels: %+v\n", err)
	}
	for _, d := range DeploymentList.Items {
		for _, f := range d.Spec.Template.Spec.InitContainers {
			fmt.Printf("Adding image %s to the list\n", f.Image)
			list = append(list, fmt.Sprintf("%s", f.Image))
		}

		for _, f := range d.Spec.Template.Spec.Containers {
			fmt.Printf("Adding image %s to the list\n", f.Image)
			list = append(list, fmt.Sprintf("%s", f.Image))
		}
	}

	return list
}

// Fetch all node names to be able to iterate to the ones that we want specify by the filter
func fetchNodeNames(clientset *kubernetes.Clientset, prefetch *cachev1.Prefetch) []string {
	list := []string{}
	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Printf("Error fetching nodes, check your permissions: %+v\n", err)
	}

	for _, node := range nodes.Items {
		if strings.Contains(node.Name, prefetch.Spec.NodeFilter) {
			list = append(list, node.Name)
		}
	}

	fmt.Printf("Node list: %+v\n", list)

	return list
}

// This is where the pod is created and we use affinity to make sure it uses the right host
func PrefetchImages(r *PrefetchReconciler, prefetch *cachev1.Prefetch) {
	id := uuid.New()
	prefix := "prefetch-operator-pod"
	name := prefix + "-" + id.String()
	labels := map[string]string{
		"app": prefix,
	}

	clientset, _ := getClientSet()
	imagesWithTags := fetchImagesWithTags(clientset, prefetch.Spec.FilterByLabels)
	nodeList := fetchNodeNames(clientset, prefetch)

	for _, node := range nodeList {
		for index, image := range imagesWithTags {
			// command := fmt.Sprintf("docker pull %s")
			command := fmt.Sprintf("/bin/sh -c exit")

			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-%s-%d", name, node, index),
					Namespace: prefetch.Namespace,
					Labels:    labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    "prefetch",
							Command: strings.Split(command, " "),
							Image:   image,
							// Initially I was going to use a privileged container
							// to talk to the docker daemon, but I then realized
							// it's easier to call the image with exit 0
							// Image:           "docker/dind",
							// SecurityContext: &v1.SecurityContext{Privileged: &privileged},
						},
					},
					RestartPolicy: corev1.RestartPolicyOnFailure,
					Affinity: &corev1.Affinity{
						NodeAffinity: &corev1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
								NodeSelectorTerms: []corev1.NodeSelectorTerm{
									{
										MatchExpressions: []corev1.NodeSelectorRequirement{
											{
												Key:      "kubernetes.io/hostname",
												Operator: "In",
												Values:   []string{node},
											},
										},
									},
								},
							},
						},
					},
				},
			}

			if prefetch.Status.Phase == "" || prefetch.Status.Phase == "PENDING" {
				prefetch.Status.Phase = cachev1.PhaseRunning
			}

			// Transition our own status
			// the status is not really relevant here
			// but we want to keep track of our pods
			// so if the operator goes away takes it's pods
			// with it
			switch prefetch.Status.Phase {
			case cachev1.PhasePending:
				prefetch.Status.Phase = cachev1.PhaseRunning
			case cachev1.PhaseRunning:
				err := controllerutil.SetControllerReference(prefetch, pod, r.Scheme)
				found := &corev1.Pod{}
				nsName := types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}
				err = r.Get(context.TODO(), nsName, found)
				if err != nil && errors.IsNotFound(err) {
					_ = r.Create(context.TODO(), pod)
					fmt.Printf("Pod launched with name: %+v\n", pod.Name)
				} else if found.Status.Phase == corev1.PodFailed ||
					found.Status.Phase == corev1.PodSucceeded {
					fmt.Printf("Container terminated reason with message: %+v, and status: %+v",
						found.Status.Reason, found.Status.Message)
					prefetch.Status.Phase = cachev1.PhaseFailed
				}
			}

			// Update the At prefetch, setting the status to the respective phase:
			err := r.Status().Update(context.TODO(), prefetch)
			err = r.Create(context.TODO(), pod)
			if err != nil {
				fmt.Printf("There was an error invoking the pod: %+v\n", err)
			}
		}
	}
}

// Delete all pods that have already ran and are in Completed/Succeeded status
func DeleteCompletedPods(prefetch *cachev1.Prefetch) {
	fieldSelectorFilter := "status.phase=Succeeded"
	labelsAsString := "app=prefetch-operator-pod"
	clientset, _ := getClientSet()

	pods, err := clientset.CoreV1().Pods(prefetch.Namespace).List(context.TODO(),
		metav1.ListOptions{FieldSelector: fieldSelectorFilter, LabelSelector: labelsAsString})
	if err != nil {
		fmt.Printf("failed to retrieve Pods: %+v\n", err)
	}

	for _, pod := range pods.Items {
		fmt.Printf("Deleting pod: %+v\n", pod.Name)
		if err := clientset.CoreV1().Pods(prefetch.Namespace).Delete(context.TODO(), pod.Name, metav1.DeleteOptions{}); err != nil {
			fmt.Printf("Failed to delete Pod: %+v", err)
		}
	}
}

// This would be like the main of our code
func (r *PrefetchReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	r.Log.WithValues("prefetch", req.NamespacedName)

	prefetch := &cachev1.Prefetch{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, prefetch)
	if err != nil {
		log.Error(err, "failed to get Prefetch resource\n")
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after
			// reconcile request—return and don't requeue:
			return reconcile.Result{}, client.IgnoreNotFound(err)
		}
		// Error reading the object—requeue the request:
		return reconcile.Result{}, err
	}

	fmt.Printf("Filter by labels %+v\n", prefetch.Spec.FilterByLabels)
	fmt.Printf("RetryAfter %+v\n", prefetch.Spec.RetryAfter)

	var retryAfter int
	if prefetch.Spec.RetryAfter != 0 {
		retryAfter = prefetch.Spec.RetryAfter
	} else {
		retryAfter = 300
	}

	if len(prefetch.Spec.FilterByLabels) > 0 {
		PrefetchImages(r, prefetch)
	} else {
		fmt.Printf("Skipping empty labels\n")
	}

	DeleteCompletedPods(prefetch)

	if err != nil {
		return ctrl.Result{RequeueAfter: time.Second * time.Duration(retryAfter)}, nil
	} else {
		return ctrl.Result{RequeueAfter: time.Second * time.Duration(retryAfter)}, err
	}
}

func (r *PrefetchReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cachev1.Prefetch{}).
		Complete(r)
}
