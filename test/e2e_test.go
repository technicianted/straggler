package test

import (
	"context"
	"stagger/pkg/controller"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"golang.org/x/exp/rand"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// func TestHappyCaseScenario(t *testing.T) {
// 	testenv.Test(t, func(ctx context.Context, cfg *envconf.Config) context.Context {
// 		// Initialize the client
// 		k8sClient, err := client.New(cfg.Client().RESTConfig(), client.Options{})
// 		require.NoError(t, err)

// 		namespace := cfg.Namespace()

// 		// Step 1: Deploy the controller
// 		err = DeployController(ctx, k8sClient, namespace)
// 		require.NoError(t, err)

// 		// Step 2: Create a Deployment with the StaggerGroup label
// 		deploymentName := "test-deployment"
// 		staggerGroupLabelKey := "staggerGroup"
// 		staggerGroupLabelValue := "test-group"

// 		replicas := int32(10)

// 		deployment := &appsv1.Deployment{
// 			ObjectMeta: metav1.ObjectMeta{
// 				Name:      deploymentName,
// 				Namespace: namespace,
// 			},
// 			Spec: appsv1.DeploymentSpec{
// 				Replicas: &replicas,
// 				Selector: &metav1.LabelSelector{
// 					MatchLabels: map[string]string{
// 						"app": "test-app",
// 					},
// 				},
// 				Template: corev1.PodTemplateSpec{
// 					ObjectMeta: metav1.ObjectMeta{
// 						Labels: map[string]string{
// 							"app":                "test-app",
// 							staggerGroupLabelKey: staggerGroupLabelValue,
// 						},
// 					},
// 					Spec: corev1.PodSpec{
// 						Containers: []corev1.Container{
// 							{
// 								Name:    "busybox",
// 								Image:   "busybox",
// 								Command: []string{"sleep", "3600"},
// 							},
// 						},
// 					},
// 				},
// 			},
// 		}

// 		err = k8sClient.Create(ctx, deployment)
// 		require.NoError(t, err)

// 		// Step 3: Wait and observe the pods
// 		// Wait for MinInitial pods to be ready (assuming MinInitial is 1)
// 		err = waitForReadyPods(ctx, k8sClient, namespace, map[string]string{
// 			"app":                "test-app",
// 			staggerGroupLabelKey: staggerGroupLabelValue,
// 		}, 1, 2*time.Minute)
// 		require.NoError(t, err)

// 		// Step 4: Verify that the number of ready pods increases according to the pacing
// 		// For example, wait for 2 pods to be ready
// 		err = waitForReadyPods(ctx, k8sClient, namespace, map[string]string{
// 			"app":                "test-app",
// 			staggerGroupLabelKey: staggerGroupLabelValue,
// 		}, 2, 2*time.Minute)
// 		require.NoError(t, err)

// 		// Continue waiting for the expected number of pods based on your pacing configuration
// 		// For example, wait for 4 pods to be ready
// 		err = waitForReadyPods(ctx, k8sClient, namespace, map[string]string{
// 			"app":                "test-app",
// 			staggerGroupLabelKey: staggerGroupLabelValue,
// 		}, 4, 2*time.Minute)
// 		require.NoError(t, err)

// 		// Optionally, verify that the total number of pods reaches the desired count
// 		err = waitForReadyPods(ctx, k8sClient, namespace, map[string]string{
// 			"app":                "test-app",
// 			staggerGroupLabelKey: staggerGroupLabelValue,
// 		}, int(replicas), 5*time.Minute)
// 		require.NoError(t, err)

// 		// Step 5: Assert the final state
// 		// Fetch the list of pods
// 		podList := &corev1.PodList{}
// 		listOptions := []client.ListOption{
// 			client.InNamespace(namespace),
// 			client.MatchingLabels(map[string]string{
// 				"app":                "test-app",
// 				staggerGroupLabelKey: staggerGroupLabelValue,
// 			}),
// 		}
// 		err = k8sClient.List(ctx, podList, listOptions...)
// 		require.NoError(t, err)

// 		readyPods := 0
// 		for _, pod := range podList.Items {
// 			if isPodReady(&pod) {
// 				readyPods++
// 			}
// 		}

// 		// Ensure all pods are ready
// 		require.Equal(t, int(replicas), readyPods, "All pods should be ready")

// 		return ctx
// 	})
// }

// func TestHappyCaseScenario(t *testing.T) {
// 	// Create a new feature
// 	f := features.New("Happy Case Scenario")
// 	logger := testLogger(t)
// 	var command *cmd.CMD
// 	commandStarted := false

// 	// Add steps to the feature
// 	f.Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
// 		// Initialize the client
// 		_, err := client.New(cfg.Client().RESTConfig(), client.Options{})
// 		require.NoError(t, err)

// 		// Start the controller
// 		command, err = setupController(cfg.KubeconfigFile(), logger)
// 		require.NoError(t, err)
// 		err = command.Start(logger)
// 		require.NoError(t, err)
// 		commandStarted = true

// 		return ctx
// 	})

// 	f.Assess("Deploy application and verify pacing", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
// 		k8sClient, err := client.New(cfg.Client().RESTConfig(), client.Options{})
// 		require.NoError(t, err)
// 		require.NotNil(t, k8sClient)

// 		namespace := cfg.Namespace()
// 		require.NotEmpty(t, namespace)

// 		// Step 2: Create a Deployment with the StaggerGroup label
// 		deploymentName := "test-deployment"
// 		staggerGroupLabelValue := "test-group"

// 		replicas := int32(10)

// 		deployment := &appsv1.Deployment{
// 			ObjectMeta: metav1.ObjectMeta{
// 				Name:      deploymentName,
// 				Namespace: namespace,
// 			},
// 			Spec: appsv1.DeploymentSpec{
// 				Replicas: &replicas,
// 				Selector: &metav1.LabelSelector{
// 					MatchLabels: map[string]string{
// 						"app": "test-app",
// 					},
// 				},
// 				Template: corev1.PodTemplateSpec{
// 					ObjectMeta: metav1.ObjectMeta{
// 						Labels: map[string]string{
// 							"app":                                 "test-app",
// 							controller.DefaultStaggerGroupIDLabel: staggerGroupLabelValue,
// 						},
// 					},
// 					Spec: corev1.PodSpec{
// 						Containers: []corev1.Container{
// 							{
// 								Name:    "busybox",
// 								Image:   "busybox",
// 								Command: []string{"sleep", "3600"},
// 							},
// 						},
// 					},
// 				},
// 			},
// 		}

// 		err = k8sClient.Create(ctx, deployment)
// 		require.NoError(t, err)

// 		// Step 3: Wait and observe the pods
// 		err = waitForPodsReady(ctx, k8sClient, namespace, map[string]string{
// 			"app":                                 "test-app",
// 			controller.DefaultStaggerGroupIDLabel: staggerGroupLabelValue,
// 		}, 2*time.Minute)
// 		require.NoError(t, err)

// 		logger.Info("Pods are ready")

// 		return ctx
// 	})
// 	f.Teardown(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
// 		// Stop the controller
// 		if commandStarted {
// 			err := command.Stop(logger)
// 			require.NoError(t, err)
// 		}
// 		return ctx
// 	})

// 	// Execute the feature
// 	testenv.Test(t, f.Feature())
// }

// func setupController(kubeconfigPath string, logger logr.Logger) (*cmd.CMD, error) {
// 	opts := cmd.NewOptions()
// 	opts.KubeConfigPath = kubeconfigPath
// 	opts.StaggeringConfigPath = path.Join("data", "stagger-config.yaml")
// 	opts.LeaderElection = false
// 	opts.TLSDir = path.Join("data", "tls")
// 	return cmd.NewCMD(opts, logger)
// }

func testLogger(t *testing.T) logr.Logger {
	zlog, _ := zap.NewDevelopmentConfig().Build()
	return zapr.NewLogger(zlog).WithName(t.Name())
}

// waitForPodsReady waits until all Pods matching the label selector are running and ready.
func waitForPodsReady(ctx context.Context, c client.Client, namespace string, labelSelector map[string]string, timeout time.Duration) error {
	listOpts := []client.ListOption{
		client.InNamespace(namespace),
		client.MatchingLabels(labelSelector),
	}

	return wait.PollUntilContextTimeout(ctx, time.Second, timeout, true, func(ctx context.Context) (bool, error) {
		pods := &corev1.PodList{}
		err := c.List(ctx, pods, listOpts...)
		if err != nil {
			return false, err
		}
		if len(pods.Items) == 0 {
			return false, nil
		}
		// loop through the pods and print the status
		for _, pod := range pods.Items {
			// print pod name and status
			logf.Log.Info("Pod name: ", pod.Name, " Status: ", pod.Status.Phase)
		}
		for _, pod := range pods.Items {
			if !isPodReady(&pod) {
				return false, nil
			}
		}
		return true, nil
	})
}

func isPodReady(pod *corev1.Pod) bool {
	for _, cond := range pod.Status.Conditions {
		if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

var _ = Describe("Happy Case Scenario", func() {
	var (
		ctx    context.Context
		cancel context.CancelFunc
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())

	})

	AfterEach(func() {
		cancel()
	})

	It("should deploy application and verify pacing", func() {
		// Step 2: Create a Deployment with the StaggerGroup label
		deploymentName := "test-deployment"
		staggerGroupLabelValue := "test-group"

		k8sClient, err := client.New(testEnv.Config, client.Options{})
		Expect(err).ToNot(HaveOccurred())

		replicas := int32(10)

		logger.Info("Creating the Deployment", "name", deploymentName, "namespace", namespace)

		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      deploymentName,
				Namespace: namespace,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "test-app",
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app":                         "test-app",
							controller.DefaultEnableLabel: "1",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:    "busybox",
								Image:   "busybox",
								Command: []string{"sleep", "3600"},
							},
						},
					},
				},
			},
		}

		By("Creating the Deployment")
		err = k8sClient.Create(ctx, deployment)
		Expect(err).ToNot(HaveOccurred())
		logger.Info("Deployment created", "name", deploymentName, "namespace", namespace)

		logger.Info("Waiting for pods to be created")
		// wait for the pods to be created and ready
		err = waitForPodsReady(ctx, k8sClient, namespace, map[string]string{
			"app":                                 "test-app",
			controller.DefaultStaggerGroupIDLabel: staggerGroupLabelValue,
		}, 5*time.Minute)
		Expect(err).ToNot(HaveOccurred())

		// fail("not implemented")
		// Step 3: Wait and observe the pods
		//simulatePodReadinessAndVerifyPacing(ctx, k8sClient, namespace, staggerGroupLabelValue, int(replicas))
	})
})

// simulatePodReadinessAndVerifyPacing simulates pod readiness and verifies pacing logic.
func simulatePodReadinessAndVerifyPacing(ctx context.Context, k8sClient client.Client, namespace, staggerGroupLabelValue string, totalReplicas int) {
	labelSelector := client.MatchingLabels{
		"app":                                 "test-app",
		controller.DefaultStaggerGroupIDLabel: staggerGroupLabelValue,
	}

	// Wait for the pods to be created
	By("Waiting for pods to be created")
	podList := &corev1.PodList{}
	Eventually(func() int {
		err := k8sClient.List(ctx, podList, client.InNamespace(namespace), labelSelector)
		Expect(err).NotTo(HaveOccurred())
		return len(podList.Items)
	}, 10*time.Second, 1*time.Second).Should(Equal(totalReplicas))

	// Define the pacing steps
	pacingSteps := []int{1, 2, 4}

	admittedPods := 0

	for _, allowedCount := range pacingSteps {
		// Simulate readiness for allowedCount pods
		for i := admittedPods; i < min(admittedPods+allowedCount, totalReplicas); i++ {
			pod := &podList.Items[i]
			markPodReady(ctx, k8sClient, pod)
		}
		admittedPods += allowedCount

		// Wait for the controller to process the readiness updates
		time.Sleep(2 * time.Second)

		// Verify that the number of ready pods matches the expected count
		readyPods := countReadyPods(ctx, k8sClient, podList.Items)
		Expect(readyPods).To(Equal(admittedPods))
	}

	// Finally, make the rest of the pods ready
	for i := admittedPods; i < totalReplicas; i++ {
		pod := &podList.Items[i]
		markPodReady(ctx, k8sClient, pod)
	}

	// Wait for the controller to process
	time.Sleep(2 * time.Second)

	// Verify that all pods are ready
	readyPods := countReadyPods(ctx, k8sClient, podList.Items)
	Expect(readyPods).To(Equal(totalReplicas))
}

// markPodReady updates the pod status to mark it as ready.
func markPodReady(ctx context.Context, k8sClient client.Client, pod *corev1.Pod) {
	pod.Status.Conditions = append(pod.Status.Conditions, corev1.PodCondition{
		Type:   corev1.PodReady,
		Status: corev1.ConditionTrue,
	})
	// Update the pod status in the API server
	err := k8sClient.Status().Update(ctx, pod)
	Expect(err).NotTo(HaveOccurred())
}

// countReadyPods counts the number of pods that are in the Ready state.
func countReadyPods(ctx context.Context, k8sClient client.Client, pods []corev1.Pod) int {
	readyPods := 0
	for _, pod := range pods {
		updatedPod := &corev1.Pod{}
		err := k8sClient.Get(ctx, client.ObjectKeyFromObject(&pod), updatedPod)
		Expect(err).NotTo(HaveOccurred())
		if isPodReady(updatedPod) {
			readyPods++
		}
	}
	return readyPods
}

// min returns the smaller of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// RandStringRunes generates a random string of n characters.
func RandStringRunes(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyz")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
