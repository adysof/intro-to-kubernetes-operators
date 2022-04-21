/*
Copyright 2022.

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

package controllers

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	operatorv1alpha1 "github.com/adysof/course-operator/api/v1alpha1"
)

// GravReconciler reconciles a Grav object
type GravReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=operator.adysof.com,resources=gravs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.adysof.com,resources=gravs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=operator.adysof.com,resources=gravs/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=*,resources=ingresses,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Grav object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *GravReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	grav := &operatorv1alpha1.Grav{}
	if err := r.Client.Get(ctx, req.NamespacedName, grav); err != nil {
		if k8serrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: v1.ObjectMeta{
			Name:      grav.GetName(),
			Namespace: grav.GetNamespace(),
		},
	}

	result, err := controllerutil.CreateOrUpdate(ctx, r.Client, deployment, mutateDeployment(ctx, deployment, grav, r.Scheme))
	if err != nil {
		return ctrl.Result{}, err
	}
	if result != controllerutil.OperationResultNone {
		if err := r.updateStatus(ctx, req.NamespacedName, func(grav *operatorv1alpha1.Grav) {
			grav.Status.Deployment.Name = deployment.GetName()
			grav.Status.Deployment.DeploymentStatus = deployment.Status
		}); err != nil {
			return ctrl.Result{}, err
		}

		logger.Info(fmt.Sprintf("%s deployment has been configured", deployment.GetName()))
		return ctrl.Result{}, nil
	}

	// Service
	service := &corev1.Service{
		ObjectMeta: v1.ObjectMeta{
			Name:      grav.GetName(),
			Namespace: grav.GetNamespace(),
		},
	}

	result, err = controllerutil.CreateOrUpdate(ctx, r.Client, service, mutateService(ctx, service, grav, r.Scheme))
	if err != nil {
		return ctrl.Result{}, err
	}
	if result != controllerutil.OperationResultNone {
		if err := r.updateStatus(ctx, req.NamespacedName, func(grav *operatorv1alpha1.Grav) {
			grav.Status.Service.Name = service.GetName()
			grav.Status.Service.ServiceStatus = service.Status
		}); err != nil {
			return ctrl.Result{}, err
		}

		logger.Info(fmt.Sprintf("%s service has been configured", service.GetName()))
		return ctrl.Result{}, nil
	}	

	// Ingress
	ingress := &networkingv1.Ingress{
		ObjectMeta: v1.ObjectMeta{
			Name:      grav.GetName(),
			Namespace: grav.GetNamespace(),
		},
	}

	result, err = controllerutil.CreateOrUpdate(ctx, r.Client, ingress, mutateIngress(ctx, ingress, grav, r.Scheme))
	if err != nil {
		return ctrl.Result{}, err
	}
	if result != controllerutil.OperationResultNone {
		if err := r.updateStatus(ctx, req.NamespacedName, func(grav *operatorv1alpha1.Grav) {
			grav.Status.Ingress.Name = ingress.GetName()
			grav.Status.Ingress.IngressStatus = ingress.Status
		}); err != nil {
			return ctrl.Result{}, err
		}

		logger.Info(fmt.Sprintf("%s ingress has been configured", ingress.GetName()))
		return ctrl.Result{}, nil
	}	

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GravReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.Grav{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&networkingv1.Ingress{}).
		Complete(r)
}

func (r *GravReconciler) updateStatus(ctx context.Context, namespacedName k8stypes.NamespacedName, update func(grav *operatorv1alpha1.Grav)) error {
	grav := &operatorv1alpha1.Grav{}

	if err := r.Client.Get(ctx, namespacedName, grav); err != nil {
		return err
	}

	update(grav)

	if err := r.Status().Update(ctx, grav); err != nil {
		return fmt.Errorf("error updating grav status: %s", err)
	}

	return nil
}

func mutateDeployment(ctx context.Context, deployment *appsv1.Deployment, grav *operatorv1alpha1.Grav, scheme *runtime.Scheme) func() error {
	return func() error {
		deployment.Spec.Template.ObjectMeta = metav1.ObjectMeta{
			Labels: map[string]string{
				"app": grav.GetName(),
			},
		}
		deployment.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"app": grav.GetName(),
			},
		}
		deployment.Spec.Template.Spec.Containers = []corev1.Container{
			{
				Name:                     "grav",
				Image:                    "adysof/grav",
				ImagePullPolicy:          corev1.PullIfNotPresent,
				TerminationMessagePath:   "/dev/termination-log",
				TerminationMessagePolicy: "File",
			},
		}

		return controllerutil.SetControllerReference(grav, deployment, scheme)
	}
}

func mutateService(ctx context.Context, service *corev1.Service, grav *operatorv1alpha1.Grav, scheme *runtime.Scheme) func() error {
	return func() error {
		service.ObjectMeta.Labels = map[string]string{
			"app": grav.GetName(),
		}
		service.Spec.Selector = map[string]string{
			"app": grav.GetName(),
		}

		service.Spec.Ports = []corev1.ServicePort{
			{
				Protocol:   corev1.ProtocolTCP,
				Port:       80,
				TargetPort: intstr.FromInt(80),
			},
		}

		return controllerutil.SetControllerReference(grav, service, scheme)
	}
}

func mutateIngress(ctx context.Context, ingress *networkingv1.Ingress, grav *operatorv1alpha1.Grav, scheme *runtime.Scheme) func() error {
	return func() error {
		ingress.ObjectMeta.Labels = map[string]string{
			"app": grav.GetName(),
		}

		pathType := networkingv1.PathTypePrefix

		ingress.Spec.Rules = []networkingv1.IngressRule{
			{
				Host: grav.Spec.Domain,
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{
							{
								Path:     "/",
								PathType: &pathType,
								Backend: networkingv1.IngressBackend{
									Service: &networkingv1.IngressServiceBackend{
										Name: grav.Status.Service.Name,
										Port: networkingv1.ServiceBackendPort{
											Number: 80,
										},
									},
								},
							},
						},
					},
				},
			},
		}

		return controllerutil.SetControllerReference(grav, ingress, scheme)
	}
}
