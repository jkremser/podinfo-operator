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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PodinfoSpec defines the desired state of Podinfo
type PodinfoSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	FrontendReplicas int    `json:"frontend-replicas,omitempty"`
	BackendReplicas  int    `json:"backend-replicas,omitempty"`
	Message          string `json:"message,omitempty"`
}

// PodinfoStatus defines the observed state of Podinfo
type PodinfoStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Podinfo is the Schema for the podinfoes API
type Podinfo struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PodinfoSpec   `json:"spec,omitempty"`
	Status PodinfoStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PodinfoList contains a list of Podinfo
type PodinfoList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Podinfo `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Podinfo{}, &PodinfoList{})
}
