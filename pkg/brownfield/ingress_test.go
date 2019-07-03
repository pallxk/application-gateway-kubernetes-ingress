// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package brownfield

import (
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests/fixtures"
)

var _ = Describe("test prunning Ingress based on white/white lists", func() {

	Context("Test PruneIngressRules()", func() {
		prohibited := fixtures.GetProhibitedTargets()
		managed := fixtures.GetManagedTargets()

		ingress := v1beta1.Ingress{
			Spec: v1beta1.IngressSpec{
				Rules: []v1beta1.IngressRule{
					{
						Host: tests.Host,
						IngressRuleValue: v1beta1.IngressRuleValue{
							HTTP: &v1beta1.HTTPIngressRuleValue{
								Paths: []v1beta1.HTTPIngressPath{
									{
										Path: fixtures.PathFoo,
										Backend: v1beta1.IngressBackend{
											ServiceName: tests.ServiceName,
											ServicePort: intstr.IntOrString{
												Type:   intstr.Int,
												IntVal: 80,
											},
										},
									},
									{
										Path: fixtures.PathFox,
										Backend: v1beta1.IngressBackend{
											ServiceName: tests.ServiceName,
											ServicePort: intstr.IntOrString{
												Type:   intstr.Int,
												IntVal: 443,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		PruneIngressRules(&ingress, prohibited, managed)

		expected := v1beta1.Ingress{
			Spec: v1beta1.IngressSpec{
				Rules: []v1beta1.IngressRule{
					{
						Host: tests.Host,
						IngressRuleValue: v1beta1.IngressRuleValue{
							HTTP: &v1beta1.HTTPIngressRuleValue{
								Paths: []v1beta1.HTTPIngressPath{
									{
										Path: fixtures.PathFoo,
										Backend: v1beta1.IngressBackend{
											ServiceName: tests.ServiceName,
											ServicePort: intstr.IntOrString{
												Type:   intstr.Int,
												IntVal: 80,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		It("should have trimmed the ingress rules to what AGIC is allowed to manage", func() {
			Expect(ingress.Spec.Rules).To(Equal(expected.Spec.Rules))
		})
	})

	Context("Test shouldKeep()", func() {
		target1 := Target{

			Hostname: tests.Host,
			Port:     80,
			Path:     to.StringPtr("/fox"),
		}

		target2 := Target{

			Hostname: tests.Host,
			Port:     8090,
			Path:     to.StringPtr("/baz"),
		}

		blacklist := []Target{target1}
		whitelist := []Target{target2}

		It("should have trimmed the ingress rules AGIC is not allowed to manage", func() {
			actual := shouldKeep(&target1, &blacklist, &whitelist)
			Expect(actual).To(BeFalse())
		})

		It("should have kept the ingress rules AGIC is allowed to manage", func() {
			actual := shouldKeep(&target2, &blacklist, &whitelist)
			Expect(actual).To(BeTrue())
		})
	})

})
