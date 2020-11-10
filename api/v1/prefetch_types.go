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

package v1

import (
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Phases
const (
    PhasePending = "PENDING"
    PhaseRunning = "RUNNING"
    PhaseFailed  = "FAILED"
)

// PrefetchSpec defines the desired state of Prefetch
type PrefetchSpec struct {
    // Labels are the labels to use to filter the deployments
    // +kubebuilder:default={}
    FilterByLabels map[string]string `json:"filter_by_labels,omitempty"`

    // Simple matcher of the hostname of the nodes
    NodeFilter string `json:"node_filter,omitempty"`

    // The default time to wait between fetch and fetch
    // if not specified it will default to 300 seconds
    // +optional
    // +kubebuilder:validation:Minimum=0
    RetryAfter int `json:"retry_after,omitempty"`
}

// PrefetchStatus defines the observed state of Prefetch
type PrefetchStatus struct {
    Phase string `json:"phase,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Prefetch is the Schema for the prefetches API
type Prefetch struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   PrefetchSpec   `json:"spec,omitempty"`
    Status PrefetchStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// PrefetchList contains a list of Prefetch
type PrefetchList struct {
    metav1.TypeMeta `json:",inline"`
    metav1.ListMeta `json:"metadata,omitempty"`
    Items           []Prefetch `json:"items"`
}

func init() {
    SchemeBuilder.Register(&Prefetch{}, &PrefetchList{})
}
