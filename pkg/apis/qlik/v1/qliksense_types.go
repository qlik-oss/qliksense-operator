package v1

import (
	kapis "github.com/qlik-oss/k-apis/pkg/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// QliksenseSpec defines the desired state of Qliksense
type QliksenseSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	kapis.CRSpec `json:",inline"`
}

// QliksenseStatus defines the observed state of Qliksense
type QliksenseStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Qliksense is the Schema for the qliksenses API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=qliksenses,scope=Namespaced,shortName=qs
type Qliksense struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   *kapis.CRSpec   `json:"spec,omitempty"`
	Status QliksenseStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// QliksenseList contains a list of Qliksense
type QliksenseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Qliksense `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Qliksense{}, &QliksenseList{})
}

func (q *Qliksense) GetVersion() string {
	return q.ObjectMeta.Labels["version"]
}
