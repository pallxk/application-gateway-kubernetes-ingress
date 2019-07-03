package brownfield

import (
	"k8s.io/api/extensions/v1beta1"

	mtv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressmanagedtarget/v1"
	ptv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressprohibitedtarget/v1"
)

// PruneIngressRules mutates the ingress struct to remove targets, which AGIC should not create configuration for.
func PruneIngressRules(ing *v1beta1.Ingress, prohibitedTargets []*ptv1.AzureIngressProhibitedTarget, managedTargets []*mtv1.AzureIngressManagedTarget) {
	blacklist := GetTargetBlacklist(prohibitedTargets)
	whitelist := GetTargetWhitelist(managedTargets)

	var rules []v1beta1.IngressRule

	for _, rule := range ing.Spec.Rules {
		if rule.HTTP == nil {
			continue
		}

		target := Target{
			Hostname: rule.Host,
		}

		if rule.HTTP.Paths == nil {
			if shouldKeep(&target, blacklist, whitelist) {
				rules = append(rules, rule)
			}
			continue // to next rule
		}

		newRule := v1beta1.IngressRule{
			Host: rule.Host,
			IngressRuleValue: v1beta1.IngressRuleValue{
				HTTP: &v1beta1.HTTPIngressRuleValue{
					Paths: []v1beta1.HTTPIngressPath{},
				},
			},
		}
		for _, path := range rule.HTTP.Paths {
			target.Path = &path.Path
			if shouldKeep(&target, blacklist, whitelist) {
				newRule.HTTP.Paths = append(newRule.HTTP.Paths, path)
			}
		}
		if len(newRule.HTTP.Paths) > 0 {
			rules = append(rules, newRule)
		}
	}

	ing.Spec.Rules = rules
}
