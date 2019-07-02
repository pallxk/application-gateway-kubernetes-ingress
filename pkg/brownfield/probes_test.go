// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package brownfield

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	mtv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressmanagedtarget/v1"
	ptv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressprohibitedtarget/v1"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests/fixtures"
)

var _ = Describe("test blacklist/whitelist health probes", func() {

	whitelistedProbe := fixtures.GetApplicationGatewayProbe(nil, to.StringPtr(fixtures.PathFoo))
	blacklistedProbe := fixtures.GetApplicationGatewayProbe(nil, to.StringPtr(fixtures.PathBar))
	neutralProbe := fixtures.GetApplicationGatewayProbe(to.StringPtr(tests.OtherHost), nil)

	whiteListedProbeWeirdURL := fixtures.GetApplicationGatewayProbe(nil, to.StringPtr(fixtures.PathFoo+"/healthz"))
	blackListedProbeWeirdURL := fixtures.GetApplicationGatewayProbe(nil, to.StringPtr(fixtures.PathBar+"/healthz"))
	neutralProbeWeirdURL := fixtures.GetApplicationGatewayProbe(to.StringPtr(tests.OtherHost), to.StringPtr("/healthz"))

	probes := []n.ApplicationGatewayProbe{
		whitelistedProbe, // /foo
		blacklistedProbe, // /bar
		neutralProbe,

		whiteListedProbeWeirdURL, // /foo/healthz
		blackListedProbeWeirdURL, // /bar/healthz
		neutralProbeWeirdURL,     // /healthz
	}

	Context("Test GetManagedProbes() with a blacklist and a whitelist", func() {

		It("should be able to merge lists of probes", func() {

			managedTargets := fixtures.GetManagedTargets()       // /foo /bar /baz
			prohibitedTargets := fixtures.GetProhibitedTargets() // /fox  /bar

			// !! Action !!
			actual := GetManagedProbes(probes, managedTargets, prohibitedTargets)

			// When there is both blacklist and whitelist - the whitelist is ignored.
			Expect(len(actual)).To(Equal(4))

			// not explicitly blacklisted (does not matter that it is in blacklist)
			Expect(actual).To(ContainElement(whitelistedProbe))
			Expect(actual).To(ContainElement(whiteListedProbeWeirdURL))

			// not explicitly blacklisted -- it is in neither blacklist or whitelist
			Expect(actual).To(ContainElement(neutralProbe))
			Expect(actual).To(ContainElement(neutralProbeWeirdURL))

			// explicitly blacklisted
			Expect(actual).ToNot(ContainElement(blacklistedProbe))
			Expect(actual).ToNot(ContainElement(blackListedProbeWeirdURL))
		})
	})

	Context("Test GetManagedProbes() with a blacklist only", func() {

		It("should be able to merge lists of probes", func() {

			var managedTargets []*mtv1.AzureIngressManagedTarget
			prohibitedTargets := fixtures.GetProhibitedTargets()

			// !! Action !!
			actual := GetManagedProbes(probes, managedTargets, prohibitedTargets)

			// The returned list contains only explicitly blacklisted health probes
			// Eevn if there was a whitelist it would have been ignored.
			Expect(len(actual)).To(Equal(4))

			Expect(actual).To(ContainElement(whitelistedProbe))
			Expect(actual).To(ContainElement(whiteListedProbeWeirdURL))

			// This is the only blacklisted probe
			Expect(actual).ToNot(ContainElement(blacklistedProbe))
			Expect(actual).ToNot(ContainElement(blackListedProbeWeirdURL))

			Expect(actual).To(ContainElement(neutralProbe))
			Expect(actual).To(ContainElement(neutralProbeWeirdURL))
		})
	})

	Context("Test GetManagedProbes() with a whitelist only", func() {

		It("should be able to merge lists of probes", func() {

			managedTargets := fixtures.GetManagedTargets()
			var prohibitedTargets []*ptv1.AzureIngressProhibitedTarget

			// !! Action !!
			actual := GetManagedProbes(probes, managedTargets, prohibitedTargets)

			// Since we do not have a blacklist, whitelist is taken into account - the returned list contains only
			// explicitly whitelisted health probes.
			// Health probes not covered by the whitelist are excluded.
			Expect(len(actual)).To(Equal(4))

			Expect(actual).To(ContainElement(whitelistedProbe))
			Expect(actual).To(ContainElement(whiteListedProbeWeirdURL))

			Expect(actual).To(ContainElement(blacklistedProbe))
			Expect(actual).To(ContainElement(blackListedProbeWeirdURL))

			Expect(actual).ToNot(ContainElement(neutralProbe))
			Expect(actual).ToNot(ContainElement(neutralProbeWeirdURL))
		})
	})

	Context("Test MergeProbes()", func() {

		probeList1 := []n.ApplicationGatewayProbe{
			whitelistedProbe,
		}

		probeList2 := []n.ApplicationGatewayProbe{
			whitelistedProbe,
			blacklistedProbe,
		}

		probeList3 := []n.ApplicationGatewayProbe{
			blacklistedProbe,
		}

		It("should correctly merge lists of probes", func() {
			merge1 := MergeProbes(probeList2, probeList3)
			Expect(len(merge1)).To(Equal(2))
			Expect(merge1).To(ContainElement(whitelistedProbe))
			Expect(merge1).To(ContainElement(blacklistedProbe))

			merge2 := MergeProbes(probeList1, probeList3)
			Expect(len(merge2)).To(Equal(2))
			Expect(merge1).To(ContainElement(whitelistedProbe))
			Expect(merge1).To(ContainElement(blacklistedProbe))
		})
	})

	Context("test inProbeList", func() {
		{
			probe := fixtures.GetApplicationGatewayProbe(nil, nil)
			actual := inProbeList(&probe, getProhibitedTargetList(fixtures.GetProhibitedTargets()))
			It("should be able to find probe in prohibited Target list", func() {
				Expect(actual).To(BeFalse())
			})
		}
		{
			probe := fixtures.GetApplicationGatewayProbe(nil, nil)
			actual := inProbeList(&probe, getManagedTargetList(fixtures.GetManagedTargets()))
			It("should be able to find probe in managed Target list", func() {
				Expect(actual).To(BeTrue())
			})
		}
	})

	Context("test inProbeList with health probe to a sub-path", func() {
		target := Target{
			Hostname: tests.Host,
			Port:     int32(80),
			Path:     to.StringPtr("/abc/*"),
		}
		targetList := []Target{target}

		It("should be able to match a probe in the sub-path of the target", func() {
			probe := fixtures.GetApplicationGatewayProbe(to.StringPtr(tests.Host), to.StringPtr("/abc/healthz"))
			actual := inProbeList(&probe, &targetList)
			Expect(actual).To(BeTrue())
		})

		It("should be able to find probe exactly matching the path of the target", func() {
			probe := fixtures.GetApplicationGatewayProbe(to.StringPtr(tests.Host), to.StringPtr("/abc"))

			actual := inProbeList(&probe, &targetList)
			Expect(actual).To(BeTrue())
		})

		It("should not be able to find probe not matching the target", func() {
			probe := fixtures.GetApplicationGatewayProbe(to.StringPtr(tests.Host), to.StringPtr("/xyz"))
			actual := inProbeList(&probe, &targetList)
			Expect(actual).To(BeFalse())
		})

	})
})
