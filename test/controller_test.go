// Copyright (c) straggler team and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.
package test

import (
	"context"
	"fmt"
	"os/exec"
	"straggler/pkg/cmd"
	"straggler/pkg/controller"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Happy Case Scenario", func() {
	Context("When creating a deployment", func() {
		It("should create deployment and verify pacing", func() {
			// Create a Deployment with the StaggerGroup label
			deploymentName := "test-deployment"

			k8sClient, err := client.New(testEnv.Config, client.Options{})
			Expect(err).ToNot(HaveOccurred())

			replicas := int32(10)

			logger.Info("Creating the Deployment", "name", deploymentName, "namespace", Namespace)
			deployment := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      deploymentName,
					Namespace: Namespace,
				},
				Spec: appsv1.DeploymentSpec{
					Strategy: appsv1.DeploymentStrategy{
						Type: appsv1.RollingUpdateDeploymentStrategyType,
					},
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
									Name: "busybox",
									// this should be different for each test
									Image:   "busybox:latest",
									Command: []string{"sleep", "3600"},
									ReadinessProbe: &corev1.Probe{
										ProbeHandler: corev1.ProbeHandler{
											Exec: &corev1.ExecAction{
												Command: []string{
													// check if the file /tmp/ready exists
													"test", "-f", "/tmp/ready",
												},
											},
										},
										InitialDelaySeconds: 1,
										PeriodSeconds:       1,
										SuccessThreshold:    1,
									},
								},
							},
						},
					},
				},
			}

			By("Creating the Deployment")
			ctx := context.Background()
			err = k8sClient.Create(ctx, deployment)
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(func() {
				zero := int64(0)
				k8sClient.Delete(context.Background(), deployment, &client.DeleteOptions{GracePeriodSeconds: &zero})
			})

			labels := map[string]string{"app": "test-app"}

			// Step 1: 0 ready, 1 starting, 9 blocked
			starting := waitForPodsConditionAndReturnStartingPods(
				ctx,
				k8sClient,
				Namespace,
				labels,
				0,                    // expectedReady
				1,                    // expectedStarting
				9,                    // expectedBlocked
				"busybox",            // containerName
				time.Minute,          // timeout
				500*time.Millisecond, // interval
				"All must be pending, expect 1 should be starting", // description
			)

			// Make the starting pod ready
			makePodsReady(ctx, starting)

			// Step 2: 1 ready, 1 starting, 8 blocked
			starting = waitForPodsConditionAndReturnStartingPods(
				ctx,
				k8sClient,
				Namespace,
				labels,
				1,                    // expectedReady
				1,                    // expectedStarting
				8,                    // expectedBlocked
				"busybox",            // containerName
				time.Minute,          // timeout
				500*time.Millisecond, // interval
				"1 pod should be ready, 1 starting, 8 blocked", // description
			)

			// Make the starting pods ready
			makePodsReady(ctx, starting)

			// Step 3: 2 ready, 1 starting, 7 blocked
			starting = waitForPodsConditionAndReturnStartingPods(
				ctx,
				k8sClient,
				Namespace,
				labels,
				2,                    // expectedReady
				1,                    // expectedStarting
				7,                    // expectedBlocked
				"busybox",            // containerName
				time.Minute,          // timeout
				500*time.Millisecond, // interval
				"2 pods should be ready, 1 starting, 7 blocked", // description
			)

			// Make the starting pods ready
			makePodsReady(ctx, starting)

			// Step 4: 3 ready, 1 starting, 6 blocked
			starting = waitForPodsConditionAndReturnStartingPods(
				ctx,
				k8sClient,
				Namespace,
				labels,
				3,                    // expectedReady
				1,                    // expectedStarting
				6,                    // expectedBlocked
				"busybox",            // containerName
				time.Minute,          // timeout
				500*time.Millisecond, // interval
				"3 pods should be ready, 1 starting, 6 blocked", // description
			)

			// Make the starting pods ready
			makePodsReady(ctx, starting)

			// Step 5: unblock all: 4 ready, 6 starting, 0 blocked
			starting = waitForPodsConditionAndReturnStartingPods(
				ctx,
				k8sClient,
				Namespace,
				labels,
				4,                    // expectedReady
				6,                    // expectedStarting
				0,                    // expectedBlocked
				"busybox",            // containerName
				time.Minute,          // timeout
				500*time.Millisecond, // interval
				"4 pods should be ready, 6 starting, 0 blocked", // description
			)

			// Make the starting pods ready
			makePodsReady(ctx, starting)

			// Step 6: 10 ready, 0 starting, 0 blocked
			waitForPodsConditionAndReturnStartingPods(
				ctx,
				k8sClient,
				Namespace,
				labels,
				10,                        // expectedReady
				0,                         // expectedStarting
				0,                         // expectedBlocked
				"busybox",                 // containerName
				time.Minute,               // timeout
				500*time.Millisecond,      // interval
				"10 pods should be ready", // description
			)
		})
	})
})

// waitForPodsReady waits until all Pods matching the label selector are running and ready.
func getPodCounts(ctx context.Context, c client.Client, namespace string, labelSelector map[string]string) (ready, starting, blocked []corev1.Pod, err error) {
	listOpts := []client.ListOption{
		client.InNamespace(namespace),
		client.MatchingLabels(labelSelector),
	}

	blocker, err := cmd.NewBlocker(cmd.NewOptions())
	if err != nil {
		err = fmt.Errorf("failed to create blocker: %v", err)
		return
	}

	pods := &corev1.PodList{}
	err = c.List(ctx, pods, listOpts...)
	if err != nil {
		logger.Error(err, "Failed to list pods", "namespace", namespace, "labelSelector", labelSelector)
		return ready, starting, blocked, err
	}

	if len(pods.Items) == 0 {
		logger.Info("No pods matched the labelSelector", "namespace", namespace, "labelSelector", labelSelector)
		return ready, starting, blocked, nil
	}

	for _, pod := range pods.Items {
		if isPodReady(&pod) {
			ready = append(ready, pod)
		} else if blocker.IsBlocked(&pod.Spec) {
			blocked = append(blocked, pod)
		} else {
			starting = append(starting, pod)
		}
	}

	return ready, starting, blocked, nil
}

// waitForPodsConditionAndReturnStartingPods waits until the pods match the expected ready, starting and blocked counts.
// It also checks if the specified container in the starting pods has started.
func waitForPodsConditionAndReturnStartingPods(
	ctx context.Context,
	k8sClient client.Client,
	namespace string,
	labels map[string]string,
	expectedReady int,
	expectedStarting int,
	expectedBlocked int,
	containerName string,
	timeout time.Duration,
	interval time.Duration,
	description string,
) (startingPods []corev1.Pod) {
	By(description)
	Eventually(func() bool {
		ready, starting, blocked, err := getPodCounts(ctx, k8sClient, namespace, labels)
		Expect(err).ToNot(HaveOccurred())
		fmt.Printf("expect ready: %d, starting: %d, blocked: %d\n", expectedReady, expectedStarting, expectedBlocked)
		fmt.Printf("currnt ready: %d, starting: %d, blocked: %d\n", len(ready), len(starting), len(blocked))
		if len(ready) != expectedReady || len(starting) != expectedStarting || len(blocked) != expectedBlocked {
			return false
		}
		if expectedStarting > 0 {
			if isContainerStarted(containerName, starting...) {
				startingPods = starting
				return true
			}
			return false
		}
		return true
	}, timeout, interval).Should(BeTrue())

	return startingPods
}

// makePodsReady marks each pod in the provided list as ready.
func makePodsReady(ctx context.Context, pods []corev1.Pod) {
	for _, pod := range pods {
		makePodReady(ctx, &pod)
	}
}

func isContainerStarted(containerName string, pod ...corev1.Pod) bool {
	started := 0
	for _, p := range pod {
		for _, status := range p.Status.ContainerStatuses {
			if status.Name == containerName && status.Started != nil && *status.Started {
				started++
			}
		}
	}

	return started == len(pod)
}

func isPodReady(pod *corev1.Pod) bool {
	for _, cond := range pod.Status.Conditions {
		if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func makePodReady(ctx context.Context, pod *corev1.Pod) {
	execCommand := exec.CommandContext(
		ctx,
		"kubectl",
		"--kubeconfig", kubeConfigPath,
		"exec",
		"-n", pod.Namespace,
		pod.Name,
		"--", "touch", "/tmp/ready")
	out, err := execCommand.CombinedOutput()
	Expect(err).ToNot(HaveOccurred(), "Output: %s", out)
}
