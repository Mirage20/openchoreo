// Copyright 2025 The OpenChoreo Authors
// SPDX-License-Identifier: Apache-2.0

package deploymentpipeline

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	openchoreov1alpha1 "github.com/openchoreo/openchoreo/api/v1alpha1"
	"github.com/openchoreo/openchoreo/internal/controller"
	dp "github.com/openchoreo/openchoreo/internal/controller/dataplane"
	env "github.com/openchoreo/openchoreo/internal/controller/environment"
	org "github.com/openchoreo/openchoreo/internal/controller/organization"
	"github.com/openchoreo/openchoreo/internal/controller/testutils"
	"github.com/openchoreo/openchoreo/internal/labels"
)

var _ = Describe("DeploymentPipeline Controller", func() {
	const (
		orgName = "test-org"
		dpName  = "test-dataplane"
		envName = "test-env"
	)

	orgNamespacedName := types.NamespacedName{
		Name: orgName,
	}
	organization := &openchoreov1alpha1.Organization{
		ObjectMeta: metav1.ObjectMeta{
			Name: orgName,
		},
	}

	BeforeEach(func() {
		By("Creating and reconciling organization resource", func() {
			orgReconciler := &org.Reconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Recorder: record.NewFakeRecorder(100),
			}
			testutils.CreateAndReconcileResourceWithCycles(ctx, k8sClient, organization, orgReconciler,
				orgNamespacedName, 2)
		})

		dpNamespacedName := types.NamespacedName{
			Name:      dpName,
			Namespace: orgName,
		}

		dataplane := &openchoreov1alpha1.DataPlane{
			ObjectMeta: metav1.ObjectMeta{
				Name:      dpName,
				Namespace: orgName,
			},
		}

		By("Creating and reconciling the dataplane resource", func() {
			dpReconciler := &dp.Reconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Recorder: record.NewFakeRecorder(100),
			}
			testutils.CreateAndReconcileResource(ctx, k8sClient, dataplane, dpReconciler, dpNamespacedName)
		})

		envNamespacedName := types.NamespacedName{
			Namespace: orgName,
			Name:      envName,
		}

		environment := &openchoreov1alpha1.Environment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      envName,
				Namespace: orgName,
				Labels: map[string]string{
					labels.LabelKeyOrganizationName: orgName,
					labels.LabelKeyName:             envName,
				},
				Annotations: map[string]string{
					controller.AnnotationKeyDisplayName: "Test Environment",
					controller.AnnotationKeyDescription: "Test Environment Description",
				},
			},
			Spec: openchoreov1alpha1.EnvironmentSpec{
				DataPlaneRef: dpName,
				IsProduction: false,
				Gateway: openchoreov1alpha1.GatewayConfig{
					DNSPrefix: envName,
				},
			},
		}

		By("Creating and reconciling the environment resource", func() {
			envReconciler := &env.Reconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Recorder: record.NewFakeRecorder(100),
			}
			testutils.CreateAndReconcileResource(ctx, k8sClient, environment, envReconciler, envNamespacedName)
		})
	})

	AfterEach(func() {
		By("Deleting the organization resource", func() {
			testutils.DeleteResource(ctx, k8sClient, organization, orgNamespacedName)
		})
	})

	const pipelineName = "test-deployment-pipeline"

	pipelineNamespacedName := types.NamespacedName{
		Namespace: orgName,
		Name:      pipelineName,
	}

	pipeline := &openchoreov1alpha1.DeploymentPipeline{}

	It("should successfully create and reconcile deployment pipeline resource", func() {
		By("creating a custom resource for the Kind DeploymentPipeline", func() {
			err := k8sClient.Get(ctx, pipelineNamespacedName, pipeline)
			if err != nil && errors.IsNotFound(err) {
				dp := &openchoreov1alpha1.DeploymentPipeline{
					ObjectMeta: metav1.ObjectMeta{
						Name:      pipelineName,
						Namespace: orgName,
						Labels: map[string]string{
							labels.LabelKeyOrganizationName: orgName,
							labels.LabelKeyName:             pipelineName,
						},
						Annotations: map[string]string{
							controller.AnnotationKeyDisplayName: "Test Deployment pipeline",
							controller.AnnotationKeyDescription: "Test Deployment pipeline Description",
						},
					},
					Spec: openchoreov1alpha1.DeploymentPipelineSpec{
						PromotionPaths: []openchoreov1alpha1.PromotionPath{
							{
								SourceEnvironmentRef:  "test-env",
								TargetEnvironmentRefs: make([]openchoreov1alpha1.TargetEnvironmentRef, 0),
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, dp)).To(Succeed())
			}
		})

		By("Reconciling the deploymentPipeline resource", func() {
			depReconciler := &Reconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Recorder: record.NewFakeRecorder(100),
			}
			result, err := depReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: pipelineNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())
		})

		By("Checking the deploymentPipeline resource", func() {
			deploymentPipeline := &openchoreov1alpha1.DeploymentPipeline{}
			Eventually(func() error {
				return k8sClient.Get(ctx, pipelineNamespacedName, deploymentPipeline)
			}, time.Second*10, time.Millisecond*500).Should(Succeed())
			Expect(deploymentPipeline.Name).To(Equal(pipelineName))
			Expect(deploymentPipeline.Namespace).To(Equal(orgName))
			Expect(deploymentPipeline.Spec).NotTo(BeNil())
		})

		By("Deleting the deploymentPipeline resource", func() {
			err := k8sClient.Get(ctx, pipelineNamespacedName, pipeline)
			Expect(err).NotTo(HaveOccurred())
			Expect(k8sClient.Delete(ctx, pipeline)).To(Succeed())
		})

		By("Checking the deploymentPipeline resource deletion", func() {
			Eventually(func() error {
				return k8sClient.Get(ctx, pipelineNamespacedName, pipeline)
			}, time.Second*10, time.Millisecond*500).ShouldNot(Succeed())
		})

		By("Reconciling the deploymentPipeline resource after deletion", func() {
			dpReconciler := &Reconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Recorder: record.NewFakeRecorder(100),
			}
			result, err := dpReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: pipelineNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())
		})
	})
})
