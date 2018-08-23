package networkpolicy

import (
	"strings"

	roxV1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/protoconv"
	k8sCoreV1 "k8s.io/api/core/v1"
	networkingV1 "k8s.io/api/networking/v1"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
)

// KubernetesNetworkPolicyWrap wraps a k8s network policy so you can convert it to a proto network policy
type KubernetesNetworkPolicyWrap struct {
	*networkingV1.NetworkPolicy
}

// ToYaml produces a string holding a JSON formatted yaml for the network policy.
func (np KubernetesNetworkPolicyWrap) ToYaml() (string, error) {
	encoder := json.NewYAMLSerializer(json.DefaultMetaFactory, nil, nil)

	stringBuilder := &strings.Builder{}
	err := encoder.Encode(np, stringBuilder)
	if err != nil {
		return "", err
	}
	return stringBuilder.String(), nil
}

// ToRoxNetworkPolicy converts a k8s network policy to a proto network policy
// This code allows for our tests to call the conversion on k8s network policies
func (np KubernetesNetworkPolicyWrap) ToRoxNetworkPolicy() *roxV1.NetworkPolicy {
	return &roxV1.NetworkPolicy{
		Id:          string(np.GetUID()),
		Name:        np.GetName(),
		Namespace:   np.GetNamespace(),
		Labels:      np.GetLabels(),
		Annotations: np.GetAnnotations(),
		Created:     protoconv.ConvertTimeToTimestamp(np.GetCreationTimestamp().Time),
		ApiVersion:  np.APIVersion,
		Spec: &roxV1.NetworkPolicySpec{
			PodSelector: np.convertSelector(&np.Spec.PodSelector),
			Ingress:     np.convertIngressRules(np.Spec.Ingress),
			Egress:      np.convertEgressRules(np.Spec.Egress),
			PolicyTypes: np.convertPolicyTypes(np.Spec.PolicyTypes),
		},
	}
}

func (np KubernetesNetworkPolicyWrap) convertSelector(sel *k8sMetaV1.LabelSelector) *roxV1.LabelSelector {
	if sel == nil {
		return nil
	}
	return &roxV1.LabelSelector{
		MatchLabels: sel.MatchLabels,
	}
}

func (np KubernetesNetworkPolicyWrap) convertProtocol(p *k8sCoreV1.Protocol) roxV1.Protocol {
	if p == nil {
		return roxV1.Protocol_UNSET_PROTOCOL
	}
	switch *p {
	case k8sCoreV1.ProtocolUDP:
		return roxV1.Protocol_UDP_PROTOCOL
	case k8sCoreV1.ProtocolTCP:
		return roxV1.Protocol_TCP_PROTOCOL
	default:
		logger.Warnf("Network protocol %s is not handled", *p)
		return roxV1.Protocol_UNSET_PROTOCOL
	}
}

func (np KubernetesNetworkPolicyWrap) convertPorts(k8sPorts []networkingV1.NetworkPolicyPort) []*roxV1.NetworkPolicyPort {
	ports := make([]*roxV1.NetworkPolicyPort, 0, len(k8sPorts))
	for _, p := range k8sPorts {
		var portVal int32
		if p.Port != nil {
			portVal = p.Port.IntVal
		}
		ports = append(ports, &roxV1.NetworkPolicyPort{
			Port:     portVal,
			Protocol: np.convertProtocol(p.Protocol),
		})
	}
	return ports
}

func (np KubernetesNetworkPolicyWrap) convertIPBlock(ipBlock *networkingV1.IPBlock) *roxV1.IPBlock {
	if ipBlock == nil {
		return nil
	}
	return &roxV1.IPBlock{
		Cidr:   ipBlock.CIDR,
		Except: ipBlock.Except,
	}
}

func (np KubernetesNetworkPolicyWrap) convertNetworkPolicyPeer(k8sPeers []networkingV1.NetworkPolicyPeer) []*roxV1.NetworkPolicyPeer {
	peers := make([]*roxV1.NetworkPolicyPeer, 0, len(k8sPeers))
	for _, peer := range k8sPeers {
		peers = append(peers, &roxV1.NetworkPolicyPeer{
			PodSelector:       np.convertSelector(peer.PodSelector),
			NamespaceSelector: np.convertSelector(peer.NamespaceSelector),
			IpBlock:           np.convertIPBlock(peer.IPBlock),
		})
	}
	return peers
}

func (np KubernetesNetworkPolicyWrap) convertIngressRules(k8sIngressRules []networkingV1.NetworkPolicyIngressRule) []*roxV1.NetworkPolicyIngressRule {
	if k8sIngressRules == nil {
		return nil
	}
	ingressRules := make([]*roxV1.NetworkPolicyIngressRule, 0, len(k8sIngressRules))
	for _, rule := range k8sIngressRules {
		ingressRules = append(ingressRules, &roxV1.NetworkPolicyIngressRule{
			Ports: np.convertPorts(rule.Ports),
			From:  np.convertNetworkPolicyPeer(rule.From),
		})
	}
	return ingressRules
}

func (np KubernetesNetworkPolicyWrap) convertEgressRules(k8sEgressRules []networkingV1.NetworkPolicyEgressRule) []*roxV1.NetworkPolicyEgressRule {
	if k8sEgressRules == nil {
		return nil
	}
	egressRules := make([]*roxV1.NetworkPolicyEgressRule, 0, len(k8sEgressRules))
	for _, rule := range k8sEgressRules {
		egressRules = append(egressRules, &roxV1.NetworkPolicyEgressRule{
			Ports: np.convertPorts(rule.Ports),
			To:    np.convertNetworkPolicyPeer(rule.To),
		})
	}
	return egressRules
}

func (np KubernetesNetworkPolicyWrap) convertPolicyType(t networkingV1.PolicyType) roxV1.NetworkPolicyType {
	switch t {
	case networkingV1.PolicyTypeIngress:
		return roxV1.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE
	case networkingV1.PolicyTypeEgress:
		return roxV1.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE
	default:
		logger.Warnf("network policy type %s is not handled", t)
		return roxV1.NetworkPolicyType_UNSET_NETWORK_POLICY_TYPE
	}
}

func (np KubernetesNetworkPolicyWrap) convertPolicyTypes(k8sTypes []networkingV1.PolicyType) []roxV1.NetworkPolicyType {
	if k8sTypes == nil {
		return nil
	}
	types := make([]roxV1.NetworkPolicyType, 0, len(k8sTypes))
	for _, t := range k8sTypes {
		types = append(types, np.convertPolicyType(t))
	}
	return types
}
