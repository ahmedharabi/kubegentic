/*
Copyright 2026.

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
	"k8s.io/apimachinery/pkg/runtime"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type StorageSpec struct {
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// Where to mount the scratch volume in the container.
	// +optional
	// +kubebuilder:default="/data"
	MountPath string `json:"mountPath,omitempty"`

	// Optional cap on the emptyDir size, e.g. "1Gi".
	// +optional
	SizeLimit string `json:"sizeLimit,omitempty"`
}

// ToolSpec defines the desired state of Tool

type ToolSpec struct {
	// Container image for the tool service.
	Image string `json:"image"`

	// Port the tool's HTTP server listens on.
	// +optional
	// +kubebuilder:default=8000
	Port int32 `json:"port,omitempty"`

	// Cluster access level. The operator generates a namespace-scoped SA + Role +
	// RoleBinding to match:
	//   none      -> no ServiceAccount/RBAC created
	//   read      -> get/list/watch on all resources in the namespace
	//   readwrite -> all verbs on all resources in the namespace
	// +optional
	// +kubebuilder:default=none
	// +kubebuilder:validation:Enum=none;read;readwrite
	Access string `json:"access,omitempty"`

	// Optional ephemeral scratch storage.
	// +optional
	Storage *StorageSpec `json:"storage,omitempty"`

	// Escape hatch: a partial PodSpec strategic-merged over the operator-generated
	// pod. Use for anything the curated fields do not cover. Free-form -- the API
	// server does not validate its internal structure.
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	PodSpecOverride *runtime.RawExtension `json:"podSpecOverride,omitempty"`
}

// ToolStatus defines the observed state of Tool
type ToolStatus struct {
	// In-cluster address other components use to reach the tool. Written by the controller.
	// +optional
	Endpoint string `json:"endpoint,omitempty"`

	// Phase: Pending | Ready | Failed. Written by the controller.
	// +optional
	Phase string `json:"phase,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Tool is the Schema for the tools API
type Tool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ToolSpec   `json:"spec,omitempty"`
	Status ToolStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ToolList contains a list of Tool
type ToolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Tool `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Tool{}, &ToolList{})
}
