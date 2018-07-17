package protoconv

import (
	pkgV1 "bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var logger = logging.LoggerForModule()

// ConvertNetworkPolicy converts a k8s network policy to a proto network policy
// This code allows for our tests to call the conversion on k8s network policies
func ConvertNetworkPolicy(networkPolicy *v1.NetworkPolicy) *pkgV1.NetworkPolicy {
	return &pkgV1.NetworkPolicy{
		Id:          string(networkPolicy.GetUID()),
		Name:        networkPolicy.GetName(),
		Namespace:   networkPolicy.GetNamespace(),
		Labels:      networkPolicy.GetLabels(),
		Annotations: networkPolicy.GetAnnotations(),
		Spec: &pkgV1.NetworkPolicySpec{
			PodSelector: convertSelector(&networkPolicy.Spec.PodSelector),
			Ingress:     convertIngressRules(networkPolicy.Spec.Ingress),
			Egress:      convertEgressRules(networkPolicy.Spec.Egress),
			PolicyTypes: convertPolicyTypes(networkPolicy.Spec.PolicyTypes),
		},
	}
}

func convertSelector(sel *metav1.LabelSelector) *pkgV1.LabelSelector {
	if sel == nil {
		return nil
	}
	return &pkgV1.LabelSelector{
		MatchLabels: sel.MatchLabels,
	}
}

func convertProtocol(p *coreV1.Protocol) pkgV1.Protocol {
	if p == nil {
		return pkgV1.Protocol_UNSET_PROTOCOL
	}
	switch *p {
	case coreV1.ProtocolUDP:
		return pkgV1.Protocol_UDP_PROTOCOL
	case coreV1.ProtocolTCP:
		return pkgV1.Protocol_TCP_PROTOCOL
	default:
		logger.Warnf("Network protocol %s is not handled", *p)
		return pkgV1.Protocol_UNSET_PROTOCOL
	}
}

func convertPorts(k8sPorts []v1.NetworkPolicyPort) []*pkgV1.NetworkPolicyPort {
	ports := make([]*pkgV1.NetworkPolicyPort, 0, len(k8sPorts))
	for _, p := range k8sPorts {
		ports = append(ports, &pkgV1.NetworkPolicyPort{
			Port:     p.Port.IntVal,
			Protocol: convertProtocol(p.Protocol),
		})
	}
	return ports
}

func convertIPBlock(ipBlock *v1.IPBlock) *pkgV1.IPBlock {
	if ipBlock == nil {
		return nil
	}
	return &pkgV1.IPBlock{
		Cidr:   ipBlock.CIDR,
		Except: ipBlock.Except,
	}
}

func convertNetworkPolicyPeer(k8sPeers []v1.NetworkPolicyPeer) []*pkgV1.NetworkPolicyPeer {
	peers := make([]*pkgV1.NetworkPolicyPeer, 0, len(k8sPeers))
	for _, peer := range k8sPeers {
		peers = append(peers, &pkgV1.NetworkPolicyPeer{
			PodSelector:       convertSelector(peer.PodSelector),
			NamespaceSelector: convertSelector(peer.NamespaceSelector),
			IpBlock:           convertIPBlock(peer.IPBlock),
		})
	}
	return peers
}

func convertIngressRules(k8sIngressRules []v1.NetworkPolicyIngressRule) []*pkgV1.NetworkPolicyIngressRule {
	ingressRules := make([]*pkgV1.NetworkPolicyIngressRule, 0, len(k8sIngressRules))
	for _, rule := range k8sIngressRules {
		ingressRules = append(ingressRules, &pkgV1.NetworkPolicyIngressRule{
			Ports: convertPorts(rule.Ports),
			From:  convertNetworkPolicyPeer(rule.From),
		})
	}
	return ingressRules
}

func convertEgressRules(k8sEgressRules []v1.NetworkPolicyEgressRule) []*pkgV1.NetworkPolicyEgressRule {
	egressRules := make([]*pkgV1.NetworkPolicyEgressRule, 0, len(k8sEgressRules))
	for _, rule := range k8sEgressRules {
		egressRules = append(egressRules, &pkgV1.NetworkPolicyEgressRule{
			Ports: convertPorts(rule.Ports),
			To:    convertNetworkPolicyPeer(rule.To),
		})
	}
	return egressRules
}

func convertPolicyType(t v1.PolicyType) pkgV1.NetworkPolicyType {
	switch t {
	case v1.PolicyTypeIngress:
		return pkgV1.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE
	case v1.PolicyTypeEgress:
		return pkgV1.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE
	default:
		logger.Warnf("network policy type %s is not handled", t)
		return pkgV1.NetworkPolicyType_UNSET_NETWORK_POLICY_TYPE
	}
}

func convertPolicyTypes(k8sTypes []v1.PolicyType) []pkgV1.NetworkPolicyType {
	types := make([]pkgV1.NetworkPolicyType, 0, len(k8sTypes))
	for _, t := range k8sTypes {
		types = append(types, convertPolicyType(t))
	}
	return types
}
