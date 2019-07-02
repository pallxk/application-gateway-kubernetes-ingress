// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package brownfield

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/golang/glog"
	"strings"

	mtv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressmanagedtarget/v1"
	ptv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressprohibitedtarget/v1"
)

// GetManagedProbes filters the given list of health probes to the list pools that AGIC is allowed to manage.
func GetManagedProbes(probes []n.ApplicationGatewayProbe, managed []*mtv1.AzureIngressManagedTarget, prohibited []*ptv1.AzureIngressProhibitedTarget) []n.ApplicationGatewayProbe {
	blacklist := getProhibitedTargetList(prohibited)
	whitelist := getManagedTargetList(managed)

	if len(*blacklist) == 0 && len(*whitelist) == 0 {
		return probes
	}

	// Ignore the targetWhitelist if blacklist exists.
	if len(*blacklist) > 0 {
		return applyBlacklist(probes, blacklist)
	}
	return applyWhitelist(probes, whitelist)
}

// MergeProbes merges list of lists of health probes into a single list, maintaining uniqueness.
func MergeProbes(probesBuckets ...[]n.ApplicationGatewayProbe) []n.ApplicationGatewayProbe {
	uniqProbes := make(probesByName)
	for _, bucket := range probesBuckets {
		for _, probe := range bucket {
			uniqProbes[probeName(*probe.Name)] = probe
		}
	}
	return probesMapToList(uniqProbes)
}

// PruneManagedProbes removes the managed health probes from the given list of probes; resulting in a list of probes not managed by AGIC.
func PruneManagedProbes(probes []n.ApplicationGatewayProbe, managedTargets []*mtv1.AzureIngressManagedTarget, prohibitedTargets []*ptv1.AzureIngressProhibitedTarget) []n.ApplicationGatewayProbe {
	managedProbes := GetManagedProbes(probes, managedTargets, prohibitedTargets)
	if managedProbes == nil {
		return probes
	}
	managedByName := indexProbesByName(managedProbes)
	var unmanagedProbes []n.ApplicationGatewayProbe
	for _, probe := range probes {
		if _, isManaged := managedByName[probeName(*probe.Name)]; isManaged {
			continue
		}
		unmanagedProbes = append(unmanagedProbes, probe)
	}
	return unmanagedProbes
}

func applyBlacklist(probes []n.ApplicationGatewayProbe, blacklist targetBlacklist) []n.ApplicationGatewayProbe {
	var managedProbes []n.ApplicationGatewayProbe
	for _, probe := range probes {
		if inProbeList(&probe, blacklist) {
			logProbe(5, probe, "in blacklist")
			continue
		}
		logProbe(5, probe, "implicitly managed")
		managedProbes = append(managedProbes, probe)
	}
	return managedProbes
}

func applyWhitelist(probes []n.ApplicationGatewayProbe, whitelist targetWhitelist) []n.ApplicationGatewayProbe {
	var managedProbes []n.ApplicationGatewayProbe
	for _, probe := range probes {
		if inProbeList(&probe, whitelist) {
			logProbe(5, probe, "in whitelist")
			managedProbes = append(managedProbes, probe)
			continue
		}
		logProbe(5, probe, "NOT in whitelist")
	}
	return managedProbes
}

func logProbe(verbosity glog.Level, probe n.ApplicationGatewayProbe, message string) {
	t, _ := probe.MarshalJSON()
	glog.V(verbosity).Infof("Probe %s is "+message+": %s", *probe.Name, t)
}

func inProbeList(probe *n.ApplicationGatewayProbe, targetList *[]Target) bool {
	for _, t := range *targetList {
		if t.Hostname == *probe.Host {
			if t.Path == nil {
				// Host matches; No paths - found it
				return true
			} else if strings.HasPrefix(normalizePathWithTail(*probe.Path), normalizePathWithTail(*t.Path)) {
				// Matches a path or sub-path - found it
				// If the target is: /abc -- will match probes for "/abc", as well as "/abc/healthz"
				return true
			}
		}
	}

	// Did not find it
	return false
}

func indexProbesByName(probes []n.ApplicationGatewayProbe) probesByName {
	probesByName := make(probesByName)
	for _, probe := range probes {
		probesByName[probeName(*probe.Name)] = probe
	}
	return probesByName
}

func probesMapToList(probes probesByName) []n.ApplicationGatewayProbe {
	var managedPools []n.ApplicationGatewayProbe
	for _, probe := range probes {
		managedPools = append(managedPools, probe)
	}
	return managedPools
}
