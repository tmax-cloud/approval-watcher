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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ApprovalSpec defines the desired state of Approval
type ApprovalSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

	PodName string `json:"podName"`

	Users []string `json:"users"`
}

// ApprovalStatus defines the observed state of Approval
type ApprovalStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

	Result       Result      `json:"result"`
	DecisionTime metav1.Time `json:"decisionTime,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Approval is the Schema for the approvals API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=approvals,scope=Namespaced
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
