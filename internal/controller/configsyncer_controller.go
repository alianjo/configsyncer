/*
Copyright 2024.

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

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/alianjo/configsyncer/api/v1alpha1"
	configsyncerv1alpha1 "github.com/alianjo/configsyncer/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConfigSyncerReconciler reconciles a ConfigSyncer object
type ConfigSyncerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=configsyncer.intodevops.com,resources=configsyncers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=configsyncer.intodevops.com,resources=configsyncers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=configsyncer.intodevops.com,resources=configsyncers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ConfigSyncer object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *ConfigSyncerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	configMapSync := v1alpha1.ConfigSyncer{}
	// ctx := context.Background()

	if err := r.Get(ctx, req.NamespacedName, &configMapSync); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	sourceConfigMap := &corev1.ConfigMap{}
	sourceConfigMapName := types.NamespacedName{
		Namespace: configMapSync.Spec.SourceNamespace,
		Name:      configMapSync.Spec.ConfigMapName,
	}
	if err := r.Get(ctx, sourceConfigMapName, sourceConfigMap); err != nil {
		return ctrl.Result{}, err
	}
	destinationConfigMap := &corev1.ConfigMap{}
	destinationConfigMapName := types.NamespacedName{
		Namespace: configMapSync.Spec.DestinationNamespace,
		Name:      configMapSync.Spec.ConfigMapName,
	}
	namespace := &corev1.Namespace{}
	namespaceName := types.NamespacedName{
		Namespace: configMapSync.Spec.DestinationNamespace,
		Name:      configMapSync.Spec.DestinationNamespace,
	}
	if err := r.Get(ctx, namespaceName, namespace); err != nil {
		if errors.IsNotFound(err) {
			log.Log.Info("Creating namespace", "Namespace", configMapSync.Spec.DestinationNamespace)
			namespace = &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: configMapSync.Spec.DestinationNamespace,
				},
			}
			if err := r.Create(ctx, namespace); err != nil {
				return ctrl.Result{}, err
			}
		}
	}
	if err := r.Get(ctx, destinationConfigMapName, destinationConfigMap); err != nil {
		if errors.IsNotFound(err) {
			log.Log.Info("Creating ConfigMap in destination namespace", "Namespace", configMapSync.Spec.DestinationNamespace)
			destinationConfigMap = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      configMapSync.Spec.ConfigMapName,
					Namespace: configMapSync.Spec.DestinationNamespace,
				},
				Data: sourceConfigMap.Data, // Copy data from source to destination
			}
			if err := r.Create(ctx, destinationConfigMap); err != nil {
				return ctrl.Result{}, err
			}
		} else {
			return ctrl.Result{}, err
		}
	} else {
		log.Log.Info("Updating ConfigMap in destination namespace", "Namespace", configMapSync.Spec.DestinationNamespace)
		destinationConfigMap.Data = sourceConfigMap.Data // Update data from source to destination
		if err := r.Update(ctx, destinationConfigMap); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ConfigSyncerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&configsyncerv1alpha1.ConfigSyncer{}).
		Complete(r)
}
