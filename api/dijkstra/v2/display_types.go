/*
Copyright 2024 jinli.

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

package v2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DisplaySpec defines the desired state of Display
type DisplaySpec struct {
	// A type of node identity
	// +optional
	NodeIdentity string `json:"nodeIdentity,omitempty"`
	// Start node information
	// +kubebuilder:validation:Required
	StartNode StartNode `json:"startNode,omitempty"`
	// Target nodes information
	// +optional
	TargetNodes []TargetNode `json:"targetNodes,omitempty"`
	// Algorithms used to calculate the shortest path, including dijkstra and floyd algorithms
	// +kubebuilder:validation:Enum=dijkstra;floyd
	Algorithm string `json:"algorithm,omitempty"`
}

type StartNode struct {
	// Node id
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	ID int32 `json:"id,omitempty"`
	// Node name
	// +optional
	Name string `json:"name,omitempty"`
}

type TargetNode struct {
	// Target node id
	// +optional
	ID int32 `json:"id,omitempty"`
	// Target node name
	// +optional
	Name string `json:"name,omitempty"`
	// Start node to target node distance
	// +optional
	Distance int32 `json:"distance,omitempty"`
	// Start node to target node path
	// +optional
	Path string `json:"path,omitempty"`
}

// DisplayStatus defines the observed state of Display
type DisplayStatus struct {
	// Last Update Time
	// +optional
	LastUpdate metav1.Time `json:"lastUpdate,omitempty"`
	// Dispaly  ShortestPath Compute Status
	// +kubebuilder:validation:Enum=Wait;Succeed;Failed
	ComputeStatus string `json:"computeStatus,omitempty"`
	// Record
	// +optional
	Record map[string]string `json:"record,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion
//+kubebuilder:resource:path=displays,scope=Namespaced,shortName=dp,singular=display
//+kubebuilder:printcolumn:name="Name",type="string",JSONPath=".metadata.name",description="dp name"
//+kubebuilder:printcolumn:name="NodeIdentity",type="string",JSONPath=".spec.nodeIdentity",description="dp id"
//+kubebuilder:printcolumn:name="Algorithm",type="string",JSONPath=".spec.algorithm",description="algorithm used for calculation"
//+kubebuilder:printcolumn:name="StartNodeID",type="string",JSONPath=".spec.startNode.id",description="start node id"
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=.metadata.creationTimestamp,description="how long has it been created"
//+kubebuilder:printcolumn:name="ComputeStatus",type="string",JSONPath=".status.computeStatus",description="computing state"
//+kubebuilder:printcolumn:name="Update",type=date,JSONPath=.status.lastUpdate,description="how long has it been updated"

// Display is the Schema for the displays API
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Display struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DisplaySpec   `json:"spec,omitempty"`
	Status DisplayStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DisplayList contains a list of Display
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type DisplayList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Display `json:"items"`
}
