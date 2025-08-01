// Copyright 2025 The OpenChoreo Authors
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachineryruntime "k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	openchoreov1alpha1 "github.com/openchoreo/openchoreo/api/v1alpha1"
	"github.com/openchoreo/openchoreo/internal/labels"
)

const (
	testNamespace = "test-namespace"
	testPipeline  = "test-pipeline"
	testOrg       = "test-org"
)

var _ = Describe("Project Webhook", func() {
	var (
		obj       *openchoreov1alpha1.Project
		oldObj    *openchoreov1alpha1.Project
		validator ProjectCustomValidator
		defaulter ProjectCustomDefaulter
	)

	BeforeEach(func() {
		obj = &openchoreov1alpha1.Project{}
		oldObj = &openchoreov1alpha1.Project{}
		validator = ProjectCustomValidator{
			client: k8sClient,
		}
		Expect(validator).NotTo(BeNil(), "Expected validator to be initialized")
		defaulter = ProjectCustomDefaulter{}
		Expect(defaulter).NotTo(BeNil(), "Expected defaulter to be initialized")
		Expect(oldObj).NotTo(BeNil(), "Expected oldObj to be initialized")
		Expect(obj).NotTo(BeNil(), "Expected obj to be initialized")
		// TODO (user): Add any setup logic common to all tests
	})

	AfterEach(func() {
		// TODO (user): Add any teardown logic common to all tests
	})

	// Helper functions
	createFakeClientBuilder := func() *fake.ClientBuilder {
		scheme := apimachineryruntime.NewScheme()
		err := openchoreov1alpha1.AddToScheme(scheme)
		Expect(err).NotTo(HaveOccurred())

		err = admissionv1.AddToScheme(scheme)
		Expect(err).NotTo(HaveOccurred())

		return fake.NewClientBuilder().WithScheme(scheme)
	}

	createValidOrganization := func(orgName string, orgNamespace string) *openchoreov1alpha1.Organization {
		org := &openchoreov1alpha1.Organization{
			ObjectMeta: metav1.ObjectMeta{
				Name: "org-" + orgName,
				Labels: map[string]string{
					labels.LabelKeyName: orgName,
				},
			},
			Status: openchoreov1alpha1.OrganizationStatus{
				Namespace: orgNamespace,
			},
		}
		return org
	}

	createValidDeploymentPipeline := func(name string, namespace string) *openchoreov1alpha1.DeploymentPipeline {
		pipeline := &openchoreov1alpha1.DeploymentPipeline{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pipeline-" + name,
				Namespace: namespace,
				Labels: map[string]string{
					labels.LabelKeyName: name,
				},
			},
		}
		return pipeline
	}

	createValidProject := func(name string, orgName string, namespace string, pipelineName string) *openchoreov1alpha1.Project {
		project := &openchoreov1alpha1.Project{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "proj-" + name,
				Namespace: namespace,
				Labels: map[string]string{
					labels.LabelKeyName:             name,
					labels.LabelKeyOrganizationName: orgName,
				},
			},
			Spec: openchoreov1alpha1.ProjectSpec{
				DeploymentPipelineRef: pipelineName,
			},
		}
		return project
	}

	Context("When creating Project under Defaulting Webhook", func() {
		It("Should apply defaults correctly", func() {
			// Currently no defaulting logic is implemented, but the test structure is in place
			By("Creating a basic project")
			obj = createValidProject("test-project", testOrg, testNamespace, testPipeline)

			By("Calling the Default method")
			err := defaulter.Default(ctx, obj)

			By("Verifying defaulting runs without error")
			Expect(err).NotTo(HaveOccurred())

			// If you implement actual defaulting logic, add assertions here
		})
	})

	Context("When validating Project creation", func() {
		It("Should deny creation if required labels are missing", func() {
			By("Creating a project without required labels")
			obj = &openchoreov1alpha1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "missing-labels-project",
					Namespace: testNamespace,
					// Missing required labels
				},
				Spec: openchoreov1alpha1.ProjectSpec{
					DeploymentPipelineRef: testPipeline,
				},
			}

			By("Validating the project creation")
			_, err := validator.ValidateCreate(ctx, obj)

			By("Verifying validation fails with appropriate error")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("required labels missing"))
		})

		It("Should deny creation if organization does not exist", func() {
			By("Setting up client with no organizations")

			By("Creating a project with non-existent organization")
			obj = createValidProject("test-project", "non-existent-org", "test-namespace", "test-pipeline")

			By("Validating the project creation")
			_, err := validator.ValidateCreate(ctx, obj)

			By("Verifying validation fails with appropriate error")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("organization 'non-existent-org' specified in label"))
		})

		It("Should deny creation if project namespace doesn't match organization namespace", func() {
			By("Creating an organization with a specific namespace")
			orgName := testOrg
			orgNamespace := testNamespace
			org := createValidOrganization(orgName, orgNamespace)

			By("Setting up client with the organization")
			validatorWithOrgClient := ProjectCustomValidator{
				client: createFakeClientBuilder().WithObjects(org).Build(),
			}

			By("Creating a project with mismatched namespace")
			obj = createValidProject("test-project", orgName, "different-namespace", testPipeline)

			By("Validating the project creation")
			_, err := validatorWithOrgClient.ValidateCreate(ctx, obj)

			By("Verifying validation fails with appropriate error")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("project namespace 'different-namespace' does not match with the namespace 'test-namespace' of the organization 'test-org'"))
		})

		It("Should deny creation if referenced deployment pipeline does not exist", func() {
			By("Creating an organization")
			orgName := testOrg
			orgNamespace := testNamespace
			org := createValidOrganization(orgName, orgNamespace)

			By("Setting up client with the organization but no deployment pipelines")
			validatorWithOrgClient := ProjectCustomValidator{
				client: createFakeClientBuilder().WithObjects(org).Build(),
			}

			By("Creating a project with non-existent deployment pipeline")
			obj = createValidProject("test-project", orgName, orgNamespace, "non-existent-pipeline")

			By("Validating the project creation")
			_, err := validatorWithOrgClient.ValidateCreate(ctx, obj)

			By("Verifying validation fails with appropriate error")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("deployment pipeline 'non-existent-pipeline' specified in project 'test-project' not found"))
		})

		It("Should deny creation if a duplicate project exists in the organization", func() {
			By("Creating an organization")
			orgName := testOrg
			orgNamespace := testNamespace
			org := createValidOrganization(orgName, orgNamespace)

			By("Creating a deployment pipeline")
			pipelineName := "test-pipeline"
			pipeline := createValidDeploymentPipeline(pipelineName, orgNamespace)

			By("Creating an existing project with the same name")
			existingProject := createValidProject("test-project", orgName, orgNamespace, pipelineName)

			By("Setting up client with existing resources")
			validatorWithExistingProject := ProjectCustomValidator{
				client: createFakeClientBuilder().WithObjects(org, pipeline, existingProject).Build(),
			}

			By("Creating a new project with the same name")
			obj = createValidProject("test-project", orgName, orgNamespace, pipelineName)

			By("Validating the project creation")
			_, err := validatorWithExistingProject.ValidateCreate(ctx, obj)

			By("Verifying validation fails with appropriate error")
			Expect(err).To(HaveOccurred())

			expectedErrMsg := fmt.Sprintf("project 'test-project' specified in label '%s' already exists in organization 'test-org'", labels.LabelKeyName)
			Expect(err.Error()).To(ContainSubstring(expectedErrMsg))
		})

		It("Should allow creation of a valid project", func() {
			By("Creating an organization")
			orgName := testOrg
			orgNamespace := testNamespace
			org := createValidOrganization(orgName, orgNamespace)

			By("Creating a deployment pipeline")
			pipelineName := testPipeline
			pipeline := createValidDeploymentPipeline(pipelineName, orgNamespace)

			By("Setting up client with organization and pipeline")
			validatorWithResources := ProjectCustomValidator{
				client: createFakeClientBuilder().WithObjects(org, pipeline).Build(),
			}

			By("Creating a valid project")
			obj = createValidProject("test-project", orgName, orgNamespace, pipelineName)

			By("Validating the project creation")
			_, err := validatorWithResources.ValidateCreate(ctx, obj)

			By("Verifying validation succeeds")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("When validating Project updates", func() {
		It("Should validate project updates correctly", func() {
			By("Creating an organization")
			orgName := testOrg
			orgNamespace := testNamespace
			org := createValidOrganization(orgName, orgNamespace)

			By("Creating a deployment pipeline")
			pipelineName := testPipeline
			pipeline := createValidDeploymentPipeline(pipelineName, orgNamespace)

			By("Setting up client with organization and pipeline")
			validatorWithResources := ProjectCustomValidator{
				client: createFakeClientBuilder().WithObjects(org, pipeline).Build(),
			}

			By("Creating old and new versions of the project")
			oldObj = createValidProject("test-project", orgName, orgNamespace, pipelineName)
			obj = createValidProject("test-project", orgName, orgNamespace, pipelineName)

			// There is no updates to the project object, so the validation should pass
			By("Validating the project update")
			_, err := validatorWithResources.ValidateUpdate(ctx, oldObj, obj)

			By("Verifying validation succeeds")
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
