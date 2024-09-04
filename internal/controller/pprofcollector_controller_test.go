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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/robfig/cron"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	collectorv1 "github.com/Mariscal6/pprof-operator/api/v1"
	"github.com/Mariscal6/pprof-operator/internal/collector"
)

var _ = Describe("PprofCollector Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"
		ctx := context.Background()
		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		pprofcollector := &collectorv1.PprofCollector{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resourceName,
				Namespace: "default",
			},
			Spec: collectorv1.PprofCollectorSpec{
				Schedule: "* * 10 * *",
				Duration: "1s",
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "test",
					},
				},
				Port: 8080,
			},
		}

		BeforeEach(func() {
			By("creating the custom resource for the Kind PprofCollector")
			err := k8sClient.Get(ctx, typeNamespacedName, pprofcollector)
			if err != nil && errors.IsNotFound(err) {
				Expect(k8sClient.Create(ctx, pprofcollector)).To(Succeed())
			}
		})

		AfterEach(func() {
			resource := &collectorv1.PprofCollector{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())
			By("Cleanup the specific resource instance PprofCollector")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})

		It("should successfully reconcile", func() {
			err := k8sClient.Create(ctx, &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "default",
					Labels: map[string]string{
						"app": "test",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-container",
							Image: "localhost:5000/gopher-pprof:latest",
						},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			// wait until the pod is running
			Eventually(func() bool {
				pod := &corev1.Pod{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      "test-pod",
					Namespace: "default",
				}, pod)
				if err != nil {
					return false
				}
				return pod.Status.Phase == corev1.PodRunning
			}, 20*time.Second, 5*time.Second).Should(BeTrue())

			By("Reconciling the created resource")
			controllerReconciler := &PprofCollectorReconciler{
				Client:         k8sClient,
				Scheme:         k8sClient.Scheme(),
				pprofCollector: &collector.GoPprofCollector{},
			}
			now := time.Now()
			schedule, err := cron.ParseStandard(pprofcollector.Spec.Schedule)
			Expect(err).NotTo(HaveOccurred())
			res, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(reconcile.Result{
				Requeue:      true,
				RequeueAfter: schedule.Next(now).Sub(now),
			}))

		})
	})
})
