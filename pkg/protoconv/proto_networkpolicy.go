package protoconv

import (
	pkgV1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var logger = logging.LoggerForModule()

// ProtoNetworkPolicyWrap wraps a proto network policy so you can convert it to a kubernetes network policy
type ProtoNetworkPolicyWrap struct {
	*pkgV1.NetworkPolicy
}

// ConvertNetworkPolicy converts a proto network policy to a k8s network policy
// This code allows for our tests to call the conversion on proto network policies
func (np ProtoNetworkPolicyWrap) ConvertNetworkPolicy() *v1.NetworkPolicy {
	return &v1.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "NetworkPolicy",
			APIVersion: np.GetApiVersion(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        np.GetName(),
			Namespace:   np.GetNamespace(),
			UID:         types.UID(np.GetId()),
			Labels:      np.GetLabels(),
			Annotations: np.GetAnnotations(),
			CreationTimestamp: metav1.Time{
				Time: ConvertTimestampToTimeOrNow(np.GetCreated()),
			},
		},
		Spec: v1.NetworkPolicySpec{
			PodSelector: *np.convertSelector(np.GetSpec().GetPodSelector()),
			Ingress:     np.convertIngressRules(np.GetSpec().GetIngress()),
			Egress:      np.convertEgressRules(np.GetSpec().GetEgress()),
			PolicyTypes: np.convertPolicyTypes(np.GetSpec().GetPolicyTypes()),
		},
	}
}

func (np ProtoNetworkPolicyWrap) convertSelector(sel *pkgV1.LabelSelector) *metav1.LabelSelector {
	if sel == nil {
		return nil
	}
	return &metav1.LabelSelector{
		MatchLabels: sel.MatchLabels,
	}
}

func (np ProtoNetworkPolicyWrap) convertProtocol(p pkgV1.Protocol) *coreV1.Protocol {
	var retProtocol coreV1.Protocol
	switch p {
	case pkgV1.Protocol_UNSET_PROTOCOL:
		return nil
	case pkgV1.Protocol_TCP_PROTOCOL:
		retProtocol = coreV1.ProtocolTCP
	case pkgV1.Protocol_UDP_PROTOCOL:
		retProtocol = coreV1.ProtocolUDP
	default:
		logger.Warnf("Network protocol %s is not handled", p)
		return nil
	}
	return &retProtocol
}

func (np ProtoNetworkPolicyWrap) convertPorts(protoPorts []*pkgV1.NetworkPolicyPort) []v1.NetworkPolicyPort {
	ports := make([]v1.NetworkPolicyPort, 0, len(protoPorts))
	for _, p := range protoPorts {
		var intString *intstr.IntOrString
		if p.GetPort() != 0 {
			t := intstr.FromInt(int(p.GetPort()))
			intString = &t
		}
		ports = append(ports, v1.NetworkPolicyPort{
			Port:     intString,
			Protocol: np.convertProtocol(p.GetProtocol()),
		})
	}
	return ports
}

func (np ProtoNetworkPolicyWrap) convertIPBlock(ipBlock *pkgV1.IPBlock) *v1.IPBlock {
	if ipBlock == nil {
		return nil
	}
	return &v1.IPBlock{
		CIDR:   ipBlock.GetCidr(),
		Except: ipBlock.GetExcept(),
	}
}

func (np ProtoNetworkPolicyWrap) convertNetworkPolicyPeer(protoPeers []*pkgV1.NetworkPolicyPeer) []v1.NetworkPolicyPeer {
	peers := make([]v1.NetworkPolicyPeer, 0, len(protoPeers))
	for _, peer := range protoPeers {
		peers = append(peers, v1.NetworkPolicyPeer{
			PodSelector:       np.convertSelector(peer.GetPodSelector()),
			NamespaceSelector: np.convertSelector(peer.GetNamespaceSelector()),
			IPBlock:           np.convertIPBlock(peer.GetIpBlock()),
		})
	}
	return peers
}

func (np ProtoNetworkPolicyWrap) convertIngressRules(protoIngressRules []*pkgV1.NetworkPolicyIngressRule) []v1.NetworkPolicyIngressRule {
	if protoIngressRules == nil {
		return nil
	}
	ingressRules := make([]v1.NetworkPolicyIngressRule, 0, len(protoIngressRules))
	for _, rule := range protoIngressRules {
		ingressRules = append(ingressRules, v1.NetworkPolicyIngressRule{
			Ports: np.convertPorts(rule.GetPorts()),
			From:  np.convertNetworkPolicyPeer(rule.From),
		})
	}
	return ingressRules
}

func (np ProtoNetworkPolicyWrap) convertEgressRules(protoEgressRules []*pkgV1.NetworkPolicyEgressRule) []v1.NetworkPolicyEgressRule {
	if protoEgressRules == nil {
		return nil
	}
	egressRules := make([]v1.NetworkPolicyEgressRule, 0, len(protoEgressRules))
	for _, rule := range protoEgressRules {
		egressRules = append(egressRules, v1.NetworkPolicyEgressRule{
			Ports: np.convertPorts(rule.GetPorts()),
			To:    np.convertNetworkPolicyPeer(rule.GetTo()),
		})
	}
	return egressRules
}

func (np ProtoNetworkPolicyWrap) convertPolicyType(t pkgV1.NetworkPolicyType) v1.PolicyType {
	switch t {
	case pkgV1.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE:
		return v1.PolicyTypeIngress
	case pkgV1.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE:
		return v1.PolicyTypeEgress
	default:
		logger.Warnf("network policy type %s is not handled", t)
		return v1.PolicyTypeIngress
	}
}

func (np ProtoNetworkPolicyWrap) convertPolicyTypes(protoTypes []pkgV1.NetworkPolicyType) []v1.PolicyType {
	if protoTypes == nil {
		return nil
	}
	types := make([]v1.PolicyType, 0, len(protoTypes))
	for _, t := range protoTypes {
		types = append(types, np.convertPolicyType(t))
	}
	return types
}
