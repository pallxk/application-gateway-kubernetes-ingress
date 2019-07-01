package brownfield

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
)

// BFContext (as it relates to the brownfield package) is the basket of
// App Gateway configs necessary to determine what settings should be
// managed and what should be left as-is.
type BFContext struct {
	Listeners    []*n.ApplicationGatewayHTTPListener
	RoutingRules []n.ApplicationGatewayRequestRoutingRule
	PathMaps     []n.ApplicationGatewayURLPathMap
}

type backendPoolName string

// BackendPoolToTargets is a mapping of a backend pool name to the traffic targets the given pool is responsible for.
type BackendPoolToTargets map[backendPoolName][]Target

type listenerName string

// listenersByName indexes HTTPListeners by their name.
func (c BFContext) listenersByName() map[listenerName]*n.ApplicationGatewayHTTPListener {
	// Index listeners by their Name
	listenerMap := make(map[listenerName]*n.ApplicationGatewayHTTPListener)
	for _, listener := range c.Listeners {
		listenerMap[listenerName(*listener.Name)] = listener
	}
	return listenerMap
}

type pathmapName string

// pathsByName indexes URLPathMaps by their name.
func (c BFContext) pathsByName() map[pathmapName]n.ApplicationGatewayURLPathMap {
	pathNameToPath := make(map[pathmapName]n.ApplicationGatewayURLPathMap)
	for _, pm := range c.PathMaps {
		pathNameToPath[pathmapName(*pm.Name)] = pm
	}
	return pathNameToPath
}
