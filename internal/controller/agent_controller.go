// Package controller /*
package controller

import (
	"context"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	agentv1 "github.com/ahmedharabi/kubegentic/api/v1"
)

// AgentReconciler reconciles a Agent object
type AgentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=agent.kubegentic.dev,resources=agents,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=agent.kubegentic.dev,resources=agents/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=agent.kubegentic.dev,resources=agents/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete

func (r *AgentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// 1. Fetch the Agent CR that triggered this reconcile
	agent := &agentv1.Agent{}
	if err := r.Get(ctx, req.NamespacedName, agent); err != nil {
		if errors.IsNotFound(err) {
			// Agent was deleted, owned resources are garbage collected automatically
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	logger.Info("Reconciling Agent", "name", agent.Name, "namespace", agent.Namespace)

	// 2. Set status to Pending if this is a fresh agent with no phase yet
	if agent.Status.Phase == "" {
		agent.Status.Phase = "Pending"
		if err := r.Status().Update(ctx, agent); err != nil {
			return ctrl.Result{}, err
		}
	}

	// 3. Reconcile the Deployment
	if err := r.reconcileDeployment(ctx, agent); err != nil {
		return ctrl.Result{}, err
	}

	// 4. Reconcile the Service
	if err := r.reconcileService(ctx, agent); err != nil {
		return ctrl.Result{}, err
	}

	// 5. Update status to Running
	agent.Status.Phase = "Running"
	now := metav1.Now()
	agent.Status.LastReconciled = &now
	if err := r.Status().Update(ctx, agent); err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("Agent reconciled successfully", "name", agent.Name)
	return ctrl.Result{}, nil
}

// reconcileDeployment creates or updates the Deployment for this Agent
func (r *AgentReconciler) reconcileDeployment(ctx context.Context, agent *agentv1.Agent) error {
	logger := log.FromContext(ctx)

	// Determine replica count — default to 1 if not set
	replicas := int32(1)
	if agent.Spec.Replicas != nil {
		replicas = *agent.Spec.Replicas
	}

	env := []corev1.EnvVar{
		{Name: "AGENT_NAME", Value: agent.Name},
		{Name: "AGENT_MODEL", Value: agent.Spec.Model},
		{Name: "AGENT_PROVIDER", Value: agent.Spec.Provider},
		{Name: "AGENT_SYSTEM_PROMPT", Value: agent.Spec.SystemPrompt},
		{Name: "OLLAMA_BASE_URL", Value: "http://host.minikube.internal:11434"},
	}
	if len(agent.Spec.Tools) > 0 {
		env = append(env, corev1.EnvVar{
			Name:  "TOOL_LIST",
			Value: strings.Join(agent.Spec.Tools, ","),
		})
	}
	for _, name := range agent.Spec.Tools {
		var tool agentv1.Tool
		if err := r.Get(ctx, client.ObjectKey{Name: name, Namespace: agent.Namespace}, &tool); err != nil {
			// tool doesn't exist -> fail-fast: don't wire the agent to a missing tool
			return fmt.Errorf("agent references tool %q which was not found: %w", name, err)
		}
		if tool.Status.Endpoint == "" {
			// tool exists but isn't Ready yet -> requeue and try again shortly
			return fmt.Errorf("tool %q not ready yet", name)
		}
		env = append(env, corev1.EnvVar{
			Name:  "TOOL_" + strings.ToUpper(name) + "_ENDPOINT",
			Value: tool.Status.Endpoint,
		})
	}

	if agent.Spec.APIKeySecretRef != nil {
		env = append(env, corev1.EnvVar{
			Name: "LLM_API_KEY",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: agent.Spec.APIKeySecretRef, // straight through, no translation
			},
		})
	}

	// Build the desired Deployment
	desired := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      agent.Name,
			Namespace: agent.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":              agent.Name,
					"kubegentic/agent": agent.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":              agent.Name,
						"kubegentic/agent": agent.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "agent-runtime",
							Image: "kubegentic-runtime:latest",
							// Tell Kubernetes not to pull from registry — use local image
							ImagePullPolicy: corev1.PullIfNotPresent,
							Ports: []corev1.ContainerPort{
								{ContainerPort: 8000},
							},
							// Inject agent config as environment variables
							Env: env,

							// Readiness probe — Kubernetes won't send traffic until /health returns 200
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/health",
										Port: intstr.FromInt(8000),
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       10,
							},
						},
					},
				},
			},
		},
	}

	// Set owner reference — when Agent CR is deleted, this Deployment is garbage collected
	if err := ctrl.SetControllerReference(agent, desired, r.Scheme); err != nil {
		return fmt.Errorf("failed to set owner reference on Deployment: %w", err)
	}

	// Check if Deployment already exists
	existing := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, existing)

	if errors.IsNotFound(err) {
		// Deployment doesn't exist yet — create it
		logger.Info("Creating Deployment", "name", agent.Name)
		return r.Create(ctx, desired)
	}
	if err != nil {
		return fmt.Errorf("failed to get Deployment: %w", err)
	}

	// Deployment exists — update replicas and image in case spec changed
	existing.Spec.Replicas = desired.Spec.Replicas
	existing.Spec.Template.Spec.Containers[0].Env = desired.Spec.Template.Spec.Containers[0].Env
	logger.Info("Updating Deployment", "name", agent.Name)
	return r.Update(ctx, existing)
}

// reconcileService creates or updates the ClusterIP Service for this Agent
func (r *AgentReconciler) reconcileService(ctx context.Context, agent *agentv1.Agent) error {
	logger := log.FromContext(ctx)

	desired := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      agent.Name + "-svc",
			Namespace: agent.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"kubegentic/agent": agent.Name,
			},
			Ports: []corev1.ServicePort{
				{
					Port:       80,
					TargetPort: intstr.FromInt(8000),
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}

	// Set owner reference
	if err := ctrl.SetControllerReference(agent, desired, r.Scheme); err != nil {
		return fmt.Errorf("failed to set owner reference on Service: %w", err)
	}

	// Check if Service already exists
	existing := &corev1.Service{}
	err := r.Get(ctx, types.NamespacedName{Name: agent.Name + "-svc", Namespace: agent.Namespace}, existing)

	if errors.IsNotFound(err) {
		logger.Info("Creating Service", "name", agent.Name+"-svc")
		return r.Create(ctx, desired)
	}
	if err != nil {
		return fmt.Errorf("failed to get Service: %w", err)
	}

	// Service exists — nothing to update for now
	logger.Info("Service already exists", "name", agent.Name+"-svc")
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AgentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&agentv1.Agent{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
