package brownfield

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
)

// BackendPoolToTargets is a mapping of a backend pool name to the traffic targets the given pool is responsible for.
type BackendPoolToTargets map[string][]Target

// BrownfieldContext (as it relates to the brownfield package) is the basket of
// App Gateway configs necessary to determine what settings should be
// managed and what should be left as-is.
type BrownfieldContext struct {
	Listeners    []*n.ApplicationGatewayHTTPListener
	RoutingRules []n.ApplicationGatewayRequestRoutingRule
	PathMaps     []n.ApplicationGatewayURLPathMap
}
