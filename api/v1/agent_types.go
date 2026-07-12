package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AgentSpec defines the desired state of Agent
type AgentSpec struct {
	// Model is the LLM model name to use (e.g. llama3.2, gpt-4o)
	Model string `json:"model"`

	// Provider is the LLM backend (ollama, openai, anthropic, vllm)
	// +kubebuilder:validation:Enum=ollama;openai;deepseek;groq
	Provider string `json:"provider"`

	// SystemPrompt is the system-level instruction given to the agent
	SystemPrompt string `json:"systemPrompt"`

	// Replicas is the number of agent pods to run (default 1)
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`
	// +optional
	APIKeySecretRef *corev1.SecretKeySelector `json:"apiKeySecretRef,omitempty"`
}

// AgentStatus defines the observed state of Agent
type AgentStatus struct {
	// Phase is the current lifecycle phase: Pending, Running, Failed
	Phase string `json:"phase,omitempty"`

	// ReadyReplicas is the number of pods currently ready
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// LastReconciled is the timestamp of the last successful reconcile
	LastReconciled *metav1.Time `json:"lastReconciled,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Model",type="string",JSONPath=".spec.model"
//+kubebuilder:printcolumn:name="Provider",type="string",JSONPath=".spec.provider"
//+kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
//+kubebuilder:printcolumn:name="Ready",type="integer",JSONPath=".status.readyReplicas"

// Agent is the Schema for the agents API
type Agent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AgentSpec   `json:"spec,omitempty"`
	Status AgentStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// AgentList contains a list of Agent
type AgentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Agent `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Agent{}, &AgentList{})
}
