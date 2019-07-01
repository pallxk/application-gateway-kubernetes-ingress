// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package brownfield

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/glog"

	mtv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressmanagedtarget/v1"
	ptv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressprohibitedtarget/v1"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
)

// GetPoolToTargetMapping creates a map from backend pool to targets this backend pool is responsible for.
func GetPoolToTargetMapping(ctx BFContext) BackendPoolToTargets {

	listenerMap := ctx.listenersByName()
	pathNameToPath := ctx.pathsByName()

	poolToTarget := make(BackendPoolToTargets)

	for _, rule := range ctx.RoutingRules {

		listenerName := listenerName(utils.GetLastChunkOfSlashed(*rule.HTTPListener.ID))

		var hostName string
		if listener, found := listenerMap[listenerName]; !found {
			continue
		} else {
			hostName = *listener.HostName
		}

		target := Target{
			Hostname: hostName,
			Port:     portFromListener(listenerMap[listenerName]),
		}

		if rule.URLPathMap == nil {
			// SSL Redirects do not have BackendAddressPool
			if rule.BackendAddressPool == nil {
				continue
			}
			poolName := backendPoolName(utils.GetLastChunkOfSlashed(*rule.BackendAddressPool.ID))
			poolToTarget[poolName] = append(poolToTarget[poolName], target)
		} else {
			// Follow the path map
			pathMapName := pathmapName(utils.GetLastChunkOfSlashed(*rule.URLPathMap.ID))
			for _, pathRule := range *pathNameToPath[pathMapName].PathRules {
				if pathRule.BackendAddressPool == nil {
					glog.Errorf("Path Rule %+v does not have BackendAddressPool", *pathRule.Name)
					continue
				}
				poolName := backendPoolName(utils.GetLastChunkOfSlashed(*pathRule.BackendAddressPool.ID))
				if pathRule.Paths == nil {
					glog.V(5).Infof("Path Rule %+v does not have paths list", *pathRule.Name)
					continue
				}
				for _, path := range *pathRule.Paths {
					target.Path = to.StringPtr(normalizePath(path))
					poolToTarget[poolName] = append(poolToTarget[poolName], target)
				}
			}
		}
	}
	return poolToTarget
}

// MergePools merges list of lists of backend address pools into a single list, maintaining uniqueness.
func MergePools(pools ...[]n.ApplicationGatewayBackendAddressPool) []n.ApplicationGatewayBackendAddressPool {
	uniqPool := make(map[string]n.ApplicationGatewayBackendAddressPool)
	for _, bucket := range pools {
		for _, p := range bucket {
			uniqPool[*p.Name] = p
		}
	}
	var merged []n.ApplicationGatewayBackendAddressPool
	for _, pool := range uniqPool {
		merged = append(merged, pool)
	}
	return merged
}

// GetManagedPools filters the given list of backend pools to the list pools that AGIC is allowed to manage.
func GetManagedPools(pools []n.ApplicationGatewayBackendAddressPool, managed []*mtv1.AzureIngressManagedTarget, prohibited []*ptv1.AzureIngressProhibitedTarget, ctx BFContext) []n.ApplicationGatewayBackendAddressPool {
	blacklist := getProhibitedTargetList(prohibited)
	whitelist := getManagedTargetList(managed)

	if len(*blacklist) == 0 && len(*whitelist) == 0 {
		// There is neither blacklist nor whitelist -- AGIC will manage all.
		return pools
	}

	poolToTarget := GetPoolToTargetMapping(ctx)

	// Ignore the whitelist if blacklist exists
	if len(*blacklist) > 0 {
		return applyBlacklist(pools, poolToTarget, blacklist)
	}
	return applyWhitelist(pools, poolToTarget, whitelist)
}

func logTarget(verbosity glog.Level, target Target, message string) {
	t, _ := target.MarshalJSON()
	glog.V(verbosity).Infof(message+": %s", t)
}

// PruneManagedPools removes the managed pools from the given list and returns a list of pools that is NOT managed by AGIC.
func PruneManagedPools(pools []n.ApplicationGatewayBackendAddressPool, managedTargets []*mtv1.AzureIngressManagedTarget, prohibitedTargets []*ptv1.AzureIngressProhibitedTarget, ctx BFContext) []n.ApplicationGatewayBackendAddressPool {
	managedPools := GetManagedPools(pools, managedTargets, prohibitedTargets, ctx)
	if managedPools == nil {
		return pools
	}
	managedByName := indexByName(managedPools)
	var unmanagedPools []n.ApplicationGatewayBackendAddressPool
	for _, pool := range pools {
		if _, isManaged := managedByName[backendPoolName(*pool.Name)]; !isManaged {
			unmanagedPools = append(unmanagedPools, pool)
		}
	}
	return unmanagedPools
}

func indexByName(pools []n.ApplicationGatewayBackendAddressPool) map[backendPoolName]n.ApplicationGatewayBackendAddressPool {
	indexed := make(map[backendPoolName]n.ApplicationGatewayBackendAddressPool)
	for _, pool := range pools {
		indexed[backendPoolName(*pool.Name)] = pool
	}
	return indexed
}

func applyBlacklist(pools []n.ApplicationGatewayBackendAddressPool, poolToTarget BackendPoolToTargets, blacklist *[]Target) []n.ApplicationGatewayBackendAddressPool {
	managedPools := make(map[string]n.ApplicationGatewayBackendAddressPool)
	// Apply blacklist
	for _, pool := range pools {
		for _, target := range poolToTarget[backendPoolName(*pool.Name)] {
			if target.IsIn(blacklist) {
				logTarget(5, target, "Target is in blacklist")
				continue
			}
			logTarget(5, target, "Target is implicitly managed")
			managedPools[*pool.Name] = pool
		}
	}
	return poolsMapToList(managedPools)
}

func applyWhitelist(pools []n.ApplicationGatewayBackendAddressPool, poolToTarget BackendPoolToTargets, whitelist *[]Target) []n.ApplicationGatewayBackendAddressPool {
	managedPools := make(map[string]n.ApplicationGatewayBackendAddressPool)
	// There was no blacklist; now look for explicitly white-listed backend pools.
	for _, pool := range pools {
		for _, target := range poolToTarget[backendPoolName(*pool.Name)] {
			if !target.IsIn(whitelist) {
				logTarget(5, target, "Target is NOT in whitelist")
				continue

			}
			logTarget(5, target, "Target is in whitelist")
			managedPools[*pool.Name] = pool
		}
	}
	return poolsMapToList(managedPools)
}

func poolsMapToList(poolSet map[string]n.ApplicationGatewayBackendAddressPool) []n.ApplicationGatewayBackendAddressPool {
	var managedPools []n.ApplicationGatewayBackendAddressPool
	for _, pool := range poolSet {
		managedPools = append(managedPools, pool)
	}
	return managedPools
}
