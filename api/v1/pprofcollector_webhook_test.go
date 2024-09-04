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

package v1

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("PprofCollector Webhook", func() {
	var typeNamespacedName = types.NamespacedName{
		Name:      "test-resource",
		Namespace: "default",
	}
	AfterEach(func() {
		resource := &PprofCollector{}
		err := k8sClient.Get(ctx, typeNamespacedName, resource)
		if err != nil {
			return
		}
		By("Cleanup the specific resource instance PprofCollector")
		Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
	})
	Context("When creating PprofCollector under Defaulting Webhook", func() {
		It("Should fill in the default value if a required field is empty", func() {
			resource := &PprofCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name:      typeNamespacedName.Name,
					Namespace: typeNamespacedName.Namespace,
				},
				Spec: PprofCollectorSpec{
					Schedule: "1 * * * *",
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "test",
						},
					},
					Port: 8080,
				},
			}
			err := k8sClient.Create(ctx, resource)
			Expect(err).NotTo(HaveOccurred())

			err = k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())
			Expect(resource.Spec.BasePath).To(Equal("/debug/pprof/profile"))
			Expect(resource.Spec.Duration).To(Equal("10s"))
		})
	})

	Context("When creating PprofCollector under Validating Webhook", func() {
		It("Should deny if a required field is empty", func() {
			resource := &PprofCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name:      typeNamespacedName.Name,
					Namespace: typeNamespacedName.Namespace,
				},
				Spec: PprofCollectorSpec{},
			}
			err := k8sClient.Create(ctx, resource)
			Expect(err).To(HaveOccurred())

		})
		It("Should failed if cron field is not valid", func() {
			resource := &PprofCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name:      typeNamespacedName.Name,
					Namespace: typeNamespacedName.Namespace,
				},
				Spec: PprofCollectorSpec{
					Schedule: "not correct",
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "test",
						},
					},
					Port: 8080,
				},
			}
			err := k8sClient.Create(ctx, resource)
			Expect(err).To(HaveOccurred())

		})
		It("Should failed if basepath field is not valid", func() {
			resource := &PprofCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name:      typeNamespacedName.Name,
					Namespace: typeNamespacedName.Namespace,
				},
				Spec: PprofCollectorSpec{
					Schedule: "*/1 * * * *",
					BasePath: "\a",
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "test",
						},
					},
					Port: 8080,
				},
			}
			err := k8sClient.Create(ctx, resource)
			Expect(err).To(HaveOccurred())
		})
		It("Should failed if durations field is not valid", func() {
			resource := &PprofCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name:      typeNamespacedName.Name,
					Namespace: typeNamespacedName.Namespace,
				},
				Spec: PprofCollectorSpec{
					Schedule: "*/1 * * * *",
					BasePath: "/debug/pprof/profile",
					Duration: "not correct",
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "test",
						},
					},
					Port: 8080,
				},
			}
			err := k8sClient.Create(ctx, resource)
			Expect(err).To(HaveOccurred())
		})
		It("Should failed if durations field is bigger than schedule", func() {
			resource := &PprofCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name:      typeNamespacedName.Name,
					Namespace: typeNamespacedName.Namespace,
				},
				Spec: PprofCollectorSpec{
					Schedule: "*/1 * * * *",
					BasePath: "/debug/pprof/profile",
					Duration: "2m",
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "test",
						},
					},
					Port: 8080,
				},
			}
			err := k8sClient.Create(ctx, resource)
			Expect(err).To(HaveOccurred())
		})

		It("Should admit if all required fields are provided", func() {
			resource := &PprofCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name:      typeNamespacedName.Name,
					Namespace: typeNamespacedName.Namespace,
				},
				Spec: PprofCollectorSpec{
					Schedule: "*/1 * * * *",
					BasePath: "/debug/pprof/profile",
					Duration: "10s",
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "test",
						},
					},
					Port: 8080,
				},
			}
			err := k8sClient.Create(ctx, resource)
			Expect(err).NotTo(HaveOccurred())

		})
	})

})
