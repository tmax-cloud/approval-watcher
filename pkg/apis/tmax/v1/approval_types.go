package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Result string

const (
	ResultWaiting  Result = "Waiting"
	ResultApproved Result = "Approved"
	ResultRejected Result = "Rejected"
	ResultFailed   Result = "Failed"
	ResultCanceled Result = "Canceled"
)

// ApprovalSpec defines the desired state of Approval
type ApprovalSpec struct {
	// PodName represents the name of the pod to be approved to proceed
	PodName string `json:"podName"`

	// Users are the list of the users who are requested to approve the Approval
	Users []string `json:"users"`
}

// ApprovalStatus defines the observed state of Approval
type ApprovalStatus struct {
	// Decision result of Approval
	Result Result `json:"result"`

	// Decision message
	Reason string `json:"reason,omitempty"`

	// Decision time of Approval
	DecisionTime metav1.Time `json:"decisionTime,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Approval is the Schema for the approvals API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=approvals,scope=Namespaced
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.result",description="Current status of Approval"
// +kubebuilder:printcolumn:name="Created",type="date",JSONPath=".metadata.creationTimestamp",description="Created time"
// +kubebuilder:printcolumn:name="Decided",type="date",JSONPath=".status.decisionTime",description="Decided time"
type Approval struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApprovalSpec   `json:"spec,omitempty"`
	Status ApprovalStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ApprovalList contains a list of Approval
type ApprovalList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Approval `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Approval{}, &ApprovalList{})
}
