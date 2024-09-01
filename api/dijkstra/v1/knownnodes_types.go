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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// KnownNodesSpec defines the desired state of KnownNodes
type KnownNodesSpec struct {
	// A type of node identity
	// +optional
	NodeIdentity string `json:"nodeIdentity"`
	// To node cost unit
	// +optional
	CostUnit string `json:"costUnit,omitempty"`
	// Known nodes information
	// +optional
	// +kubebuilder:validation:UniqueItems=false
	Nodes []Node `json:"nodes,omitempty"`
}

type Node struct {
	// Node id
	// +optional
	ID int32 `json:"id,omitempty"`
	// Node name
	// +optional
	Name string `json:"name,omitempty"`
	// Previous node
	// +kubebuilder:validation:Required
	PreNode *Node `json:"-"`
	// Node edges
	// +optional
	// +kubebuilder:validation:UniqueItems=false
	Edges []Edge `json:"edges,omitempty"`
}

type Edge struct {
	// To node id
	// +optional
	To int32 `json:"id,omitempty"`
	// To node cost
	// +optional
	Cost int32 `json:"cost,omitempty"`
}

// KnownNodesStatus defines the observed state of KnownNodes
type KnownNodesStatus struct {
	// Last Update Time
	// +optional
	LastUpdate metav1.Time `json:"lastUpdate,omitempty" protobuf:"bytes,1,opt,name=lastUpdate"`
	// Record
	// +optional
	Record map[string]string `json:"record,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=knownnodeses,scope=Namespaced,shortName=kn,singular=knownnodes
//+kubebuilder:printcolumn:name="Name",type="string",JSONPath=".metadata.name",description="kn name"
//+kubebuilder:printcolumn:name="NodeIdentity",type="string",JSONPath=".spec.nodeIdentity",description="kn id"
//+kubebuilder:printcolumn:name="CostUnit",type="string",JSONPath=".spec.costUnit",description="to node cost unit"
//+kubebuilder:printcolumn:name="Nodes",type="string",JSONPath=".annotations.nodes",description="kn has the number of nodes"
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=.metadata.creationTimestamp,description="how long has it been created"
//+kubebuilder:printcolumn:name="Update",type=date,JSONPath=".status.lastUpdate",description="how long has it been updated"

// KnownNodes is the Schema for the knownnodes API
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type KnownNodes struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KnownNodesSpec   `json:"spec,omitempty"`
	Status KnownNodesStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// KnownNodesList contains a list of KnownNodes
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type KnownNodesList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KnownNodes `json:"items"`
}
