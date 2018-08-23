package networkpolicy

import (
	"strings"

	roxV1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoconv"
	k8sCoreV1 "k8s.io/api/core/v1"
	k8sV1 "k8s.io/api/networking/v1"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var logger = logging.LoggerForModule()

// RoxNetworkPolicyWrap wraps a proto network policy so you can convert it to a kubernetes network policy
type RoxNetworkPolicyWrap struct {
	*roxV1.NetworkPolicy
}

// ToYaml produces a string holding a JSON formatted yaml for the network policy.
func (np RoxNetworkPolicyWrap) ToYaml() (string, error) {
	k8sNetworkPolicy := np.ToKubernetesNetworkPolicy()
	encoder := json.NewYAMLSerializer(json.DefaultMetaFactory, nil, nil)

	stringBuilder := &strings.Builder{}
	err := encoder.Encode(k8sNetworkPolicy, stringBuilder)
	if err != nil {
		return "", err
	}
	return stringBuilder.String(), nil
}

// ToKubernetesNetworkPolicy converts a proto network policy to a k8s network policy
// This code allows for our tests to call the conversion on proto network policies
func (np RoxNetworkPolicyWrap) ToKubernetesNetworkPolicy() *k8sV1.NetworkPolicy {
	return &k8sV1.NetworkPolicy{
		TypeMeta: k8sMetaV1.TypeMeta{
			Kind:       "NetworkPolicy",
			APIVersion: np.GetApiVersion(),
		},
		ObjectMeta: k8sMetaV1.ObjectMeta{
			Name:        np.GetName(),
			Namespace:   np.GetNamespace(),
			UID:         types.UID(np.GetId()),
			Labels:      np.GetLabels(),
			Annotations: np.GetAnnotations(),
			CreationTimestamp: k8sMetaV1.Time{
				Time: protoconv.ConvertTimestampToTimeOrNow(np.GetCreated()),
			},
		},
		Spec: k8sV1.NetworkPolicySpec{
			PodSelector: *np.convertSelector(np.GetSpec().GetPodSelector()),
			Ingress:     np.convertIngressRules(np.GetSpec().GetIngress()),
			Egress:      np.convertEgressRules(np.GetSpec().GetEgress()),
			PolicyTypes: np.convertPolicyTypes(np.GetSpec().GetPolicyTypes()),
		},
	}
}

func (np RoxNetworkPolicyWrap) convertSelector(sel *roxV1.LabelSelector) *k8sMetaV1.LabelSelector {
	if sel == nil {
		return nil
	}
	return &k8sMetaV1.LabelSelector{
		MatchLabels: sel.MatchLabels,
	}
}

func (np RoxNetworkPolicyWrap) convertProtocol(p roxV1.Protocol) *k8sCoreV1.Protocol {
	var retProtocol k8sCoreV1.Protocol
	switch p {
	case roxV1.Protocol_UNSET_PROTOCOL:
		return nil
	case roxV1.Protocol_TCP_PROTOCOL:
		retProtocol = k8sCoreV1.ProtocolTCP
	case roxV1.Protocol_UDP_PROTOCOL:
		retProtocol = k8sCoreV1.ProtocolUDP
	default:
		logger.Warnf("Network protocol %s is not handled", p)
		return nil
	}
	return &retProtocol
}

func (np RoxNetworkPolicyWrap) convertPorts(protoPorts []*roxV1.NetworkPolicyPort) []k8sV1.NetworkPolicyPort {
	ports := make([]k8sV1.NetworkPolicyPort, 0, len(protoPorts))
	for _, p := range protoPorts {
		var intString *intstr.IntOrString
		if p.GetPort() != 0 {
			t := intstr.FromInt(int(p.GetPort()))
			intString = &t
		}
		ports = append(ports, k8sV1.NetworkPolicyPort{
			Port:     intString,
			Protocol: np.convertProtocol(p.GetProtocol()),
		})
	}
	return ports
}

func (np RoxNetworkPolicyWrap) convertIPBlock(ipBlock *roxV1.IPBlock) *k8sV1.IPBlock {
	if ipBlock == nil {
		return nil
	}
	return &k8sV1.IPBlock{
		CIDR:   ipBlock.GetCidr(),
		Except: ipBlock.GetExcept(),
	}
}

func (np RoxNetworkPolicyWrap) convertNetworkPolicyPeer(protoPeers []*roxV1.NetworkPolicyPeer) []k8sV1.NetworkPolicyPeer {
	peers := make([]k8sV1.NetworkPolicyPeer, 0, len(protoPeers))
	for _, peer := range protoPeers {
		peers = append(peers, k8sV1.NetworkPolicyPeer{
			PodSelector:       np.convertSelector(peer.GetPodSelector()),
			NamespaceSelector: np.convertSelector(peer.GetNamespaceSelector()),
			IPBlock:           np.convertIPBlock(peer.GetIpBlock()),
		})
	}
	return peers
}

func (np RoxNetworkPolicyWrap) convertIngressRules(protoIngressRules []*roxV1.NetworkPolicyIngressRule) []k8sV1.NetworkPolicyIngressRule {
	if protoIngressRules == nil {
		return nil
	}
	ingressRules := make([]k8sV1.NetworkPolicyIngressRule, 0, len(protoIngressRules))
	for _, rule := range protoIngressRules {
		ingressRules = append(ingressRules, k8sV1.NetworkPolicyIngressRule{
			Ports: np.convertPorts(rule.GetPorts()),
			From:  np.convertNetworkPolicyPeer(rule.From),
		})
	}
	return ingressRules
}

func (np RoxNetworkPolicyWrap) convertEgressRules(protoEgressRules []*roxV1.NetworkPolicyEgressRule) []k8sV1.NetworkPolicyEgressRule {
	if protoEgressRules == nil {
		return nil
	}
	egressRules := make([]k8sV1.NetworkPolicyEgressRule, 0, len(protoEgressRules))
	for _, rule := range protoEgressRules {
		egressRules = append(egressRules, k8sV1.NetworkPolicyEgressRule{
			Ports: np.convertPorts(rule.GetPorts()),
			To:    np.convertNetworkPolicyPeer(rule.GetTo()),
		})
	}
	return egressRules
}

func (np RoxNetworkPolicyWrap) convertPolicyType(t roxV1.NetworkPolicyType) k8sV1.PolicyType {
	switch t {
	case roxV1.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE:
		return k8sV1.PolicyTypeIngress
	case roxV1.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE:
		return k8sV1.PolicyTypeEgress
	default:
		logger.Warnf("network policy type %s is not handled", t)
		return k8sV1.PolicyTypeIngress
	}
}

func (np RoxNetworkPolicyWrap) convertPolicyTypes(protoTypes []roxV1.NetworkPolicyType) []k8sV1.PolicyType {
	if protoTypes == nil {
		return nil
	}
	types := make([]k8sV1.PolicyType, 0, len(protoTypes))
	for _, t := range protoTypes {
		types = append(types, np.convertPolicyType(t))
	}
	return types
}
