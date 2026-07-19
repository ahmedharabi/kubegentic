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

package controller

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	agentv1 "github.com/ahmedharabi/kubegentic/api/v1"

	"encoding/json"

	rbacv1 "k8s.io/api/rbac/v1"
)

// ToolReconciler reconciles a Tool object
type ToolReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=agent.kubegentic.dev,resources=tools,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=agent.kubegentic.dev,resources=tools/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=agent.kubegentic.dev,resources=tools/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Tool object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.0/pkg/reconcile

func (r *ToolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var tool agentv1.Tool
	if err := r.Get(ctx, req.NamespacedName, &tool); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	port := tool.Spec.Port
	if port == 0 {
		port = 8000
	}

	// 1. RBAC (only if the tool needs cluster access). Returns the SA name to run under.
	saName, err := r.reconcileRBAC(ctx, &tool)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("reconcile rbac: %w", err)
	}

	// 2. Build the pod spec: base container + storage + override.
	podSpec, err := buildPodSpec(&tool, saName, port)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("build pod spec: %w", err)
	}

	labels := map[string]string{"app": tool.Name, "kubegentic/tool": tool.Name}

	// 3. Deployment.
	deploy := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: tool.Name, Namespace: tool.Namespace}}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, deploy, func() error {
		replicas := int32(1)
		deploy.Spec.Replicas = &replicas
		deploy.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
		deploy.Spec.Template.ObjectMeta.Labels = labels
		deploy.Spec.Template.Spec = podSpec
		return controllerutil.SetControllerReference(&tool, deploy, r.Scheme)
	}); err != nil {
		return ctrl.Result{}, fmt.Errorf("reconcile deployment: %w", err)
	}

	// 4. Service.
	svcName := tool.Name + "-svc"
	svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: svcName, Namespace: tool.Namespace}}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, svc, func() error {
		svc.Spec.Selector = labels
		svc.Spec.Ports = []corev1.ServicePort{{Port: port, TargetPort: intstr.FromInt32(port)}}
		return controllerutil.SetControllerReference(&tool, svc, r.Scheme)
	}); err != nil {
		return ctrl.Result{}, fmt.Errorf("reconcile service: %w", err)
	}

	// 5. Status.
	tool.Status.Endpoint = fmt.Sprintf("http://%s:%d", svcName, port)
	tool.Status.Phase = "Ready"
	if err := r.Status().Update(ctx, &tool); err != nil {
		return ctrl.Result{}, fmt.Errorf("update status: %w", err)
	}

	logger.Info("reconciled tool", "endpoint", tool.Status.Endpoint, "access", tool.Spec.Access)
	return ctrl.Result{}, nil
}

// reconcileRBAC creates SA + Role + RoleBinding based on access level.
// Returns the ServiceAccount name to run the pod under ("" if access is none).
func (r *ToolReconciler) reconcileRBAC(ctx context.Context, tool *agentv1.Tool) (string, error) {
	if tool.Spec.Access == "" || tool.Spec.Access == "none" {
		return "", nil
	}

	saName := tool.Name + "-sa"
	roleName := tool.Name + "-role"

	sa := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: saName, Namespace: tool.Namespace}}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, sa, func() error {
		return controllerutil.SetControllerReference(tool, sa, r.Scheme)
	}); err != nil {
		return "", err
	}

	// read -> get/list/watch; readwrite -> all verbs. Namespace-scoped (Role, not
	// ClusterRole), so "readwrite" is full access WITHIN the tool's namespace only.
	verbs := []string{"get", "list", "watch"}
	if tool.Spec.Access == "readwrite" {
		verbs = []string{"*"}
	}

	role := &rbacv1.Role{ObjectMeta: metav1.ObjectMeta{Name: roleName, Namespace: tool.Namespace}}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, role, func() error {
		role.Rules = []rbacv1.PolicyRule{{
			APIGroups: []string{"*"},
			Resources: []string{"*"},
			Verbs:     verbs,
		}}
		return controllerutil.SetControllerReference(tool, role, r.Scheme)
	}); err != nil {
		return "", err
	}

	rb := &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: tool.Name + "-rolebinding", Namespace: tool.Namespace}}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, rb, func() error {
		rb.Subjects = []rbacv1.Subject{{Kind: "ServiceAccount", Name: saName, Namespace: tool.Namespace}}
		rb.RoleRef = rbacv1.RoleRef{Kind: "Role", Name: roleName, APIGroup: "rbac.authorization.k8s.io"}
		return controllerutil.SetControllerReference(tool, rb, r.Scheme)
	}); err != nil {
		return "", err
	}

	return saName, nil
}

// buildPodSpec assembles the tool's pod: the base container, optional scratch
// storage, then the user's podSpecOverride strategic-merged on top.
func buildPodSpec(tool *agentv1.Tool, saName string, port int32) (corev1.PodSpec, error) {
	container := corev1.Container{
		Name:            "tool",
		Image:           tool.Spec.Image,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Ports:           []corev1.ContainerPort{{ContainerPort: port}},
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{Path: "/health", Port: intstr.FromInt32(port)},
			},
			InitialDelaySeconds: 3,
			PeriodSeconds:       10,
		},
	}

	base := corev1.PodSpec{Containers: []corev1.Container{container}}
	if saName != "" {
		base.ServiceAccountName = saName
	}

	// Ephemeral scratch volume (emptyDir).
	if tool.Spec.Storage != nil && tool.Spec.Storage.Enabled {
		mountPath := tool.Spec.Storage.MountPath
		if mountPath == "" {
			mountPath = "/data"
		}
		emptyDir := &corev1.EmptyDirVolumeSource{}
		if tool.Spec.Storage.SizeLimit != "" {
			q, err := resource.ParseQuantity(tool.Spec.Storage.SizeLimit)
			if err != nil {
				return corev1.PodSpec{}, fmt.Errorf("invalid storage sizeLimit: %w", err)
			}
			emptyDir.SizeLimit = &q
		}
		base.Volumes = append(base.Volumes, corev1.Volume{
			Name:         "scratch",
			VolumeSource: corev1.VolumeSource{EmptyDir: emptyDir},
		})
		base.Containers[0].VolumeMounts = append(base.Containers[0].VolumeMounts,
			corev1.VolumeMount{Name: "scratch", MountPath: mountPath})
	}

	// Escape hatch: strategic-merge the raw override over the base. Strategic merge
	// understands PodSpec's merge keys (containers merged by name, volumes by name),
	// so an override touching container "tool" merges into the base container.
	if tool.Spec.PodSpecOverride != nil && len(tool.Spec.PodSpecOverride.Raw) > 0 {
		baseJSON, err := json.Marshal(base)
		if err != nil {
			return corev1.PodSpec{}, err
		}
		mergedJSON, err := strategicpatch.StrategicMergePatch(baseJSON, tool.Spec.PodSpecOverride.Raw, corev1.PodSpec{})
		if err != nil {
			return corev1.PodSpec{}, fmt.Errorf("merge podSpecOverride: %w", err)
		}
		var merged corev1.PodSpec
		if err := json.Unmarshal(mergedJSON, &merged); err != nil {
			return corev1.PodSpec{}, err
		}
		base = merged
	}

	return base, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ToolReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&agentv1.Tool{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&rbacv1.Role{}).
		Owns(&rbacv1.RoleBinding{}).
		Complete(r)
}
