// Copyright 2025 The OpenChoreo Authors
// SPDX-License-Identifier: Apache-2.0

package deployableartifact

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"

	openchoreov1alpha1 "github.com/openchoreo/openchoreo/api/v1alpha1"
	"github.com/openchoreo/openchoreo/internal/controller"
)

// Reconciler reconciles a DeployableArtifact object
type Reconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the DeployableArtifact object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling DeployableArtifact")

	// Fetch the DeployableArtifact instance
	deployableartifact := &openchoreov1alpha1.DeployableArtifact{}
	if err := r.Get(ctx, req.NamespacedName, deployableartifact); err != nil {
		if apierrors.IsNotFound(err) {
			// The DeployableArtifact resource may have been deleted since it triggered the reconcile
			logger.Info("DeployableArtifact resource not found. Ignoring since it must be deleted.")
			return ctrl.Result{}, nil
		}
		// Error reading the object
		logger.Error(err, "Failed to get DeployableArtifact")
		return ctrl.Result{}, err
	}

	// Keep a copy of the original object for comparison
	old := deployableartifact.DeepCopy()

	// Handle the deletion of the build
	if !deployableartifact.DeletionTimestamp.IsZero() {
		logger.Info("Finalizing deployable artifact")
		return r.finalize(ctx, old, deployableartifact)
	}

	// Ensure the finalizer is added to the deployable artifact
	if finalizerAdded, err := r.ensureFinalizer(ctx, deployableartifact); err != nil || finalizerAdded {
		return ctrl.Result{}, err
	}

	// Handle create
	// Ignore reconcile if the DeployableArtifact is already available since this is a one-time createß
	if r.shouldIgnoreReconcile(deployableartifact) {
		return ctrl.Result{}, nil
	}

	// Set the observed generation
	deployableartifact.Status.ObservedGeneration = deployableartifact.Generation

	// Update the status condition to indicate the deployableArtifact is created/ready
	meta.SetStatusCondition(
		&deployableartifact.Status.Conditions,
		NewDeployableArtifactAvailableCondition(deployableartifact.Generation),
	)

	// Update status if needed
	if err := controller.UpdateStatusConditions(ctx, r.Client, old, deployableartifact); err != nil {
		return ctrl.Result{}, err
	}

	r.Recorder.Event(deployableartifact, corev1.EventTypeNormal, "ReconcileComplete", "Successfully created "+deployableartifact.Name)

	return ctrl.Result{}, nil
}

func (r *Reconciler) shouldIgnoreReconcile(deployableArtifact *openchoreov1alpha1.DeployableArtifact) bool {
	return meta.FindStatusCondition(deployableArtifact.Status.Conditions, string(controller.TypeAvailable)) != nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	if r.Recorder == nil {
		r.Recorder = mgr.GetEventRecorderFor("deployableartifact-controller")
	}

	// Set up the index for the deployable artifact reference
	if err := r.setupDeployableArtifactRefIndex(context.Background(), mgr); err != nil {
		return fmt.Errorf("failed to setup deployment artifact reference index: %w", err)
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&openchoreov1alpha1.DeployableArtifact{}).
		Named("deployableartifact").
		// Watch for Deployment changes to reconcile the component
		Watches(
			&openchoreov1alpha1.Deployment{},
			handler.EnqueueRequestsFromMapFunc(controller.HierarchyWatchHandler[*openchoreov1alpha1.Deployment, *openchoreov1alpha1.DeployableArtifact](
				r.Client, controller.GetDeployableArtifact)),
		).
		Complete(r)
}
