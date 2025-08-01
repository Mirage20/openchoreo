// Copyright 2025 The OpenChoreo Authors
// SPDX-License-Identifier: Apache-2.0

package build

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	openchoreov1alpha1 "github.com/openchoreo/openchoreo/api/v1alpha1"
)

var _ = Describe("Build Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		build := &openchoreov1alpha1.Build{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind Build")
			err := k8sClient.Get(ctx, typeNamespacedName, build)
			if err != nil && errors.IsNotFound(err) {
				resource := &openchoreov1alpha1.Build{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: openchoreov1alpha1.BuildSpec{
						Owner: openchoreov1alpha1.BuildOwner{
							ProjectName:   "test-project",
							ComponentName: "test-component",
						},
						Repository: openchoreov1alpha1.Repository{
							URL: "https://github.com/test/repo",
							Revision: openchoreov1alpha1.Revision{
								Branch: "main",
							},
							AppPath: "/",
						},
						TemplateRef: openchoreov1alpha1.TemplateRef{
							Name: "test-template",
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &openchoreov1alpha1.Build{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance Build")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &Reconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.
		})
	})
})
