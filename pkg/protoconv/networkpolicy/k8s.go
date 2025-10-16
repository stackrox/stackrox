package networkpolicy

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/protoconv/k8s"
	"github.com/stackrox/rox/pkg/utils"
	k8sCoreV1 "k8s.io/api/core/v1"
	networkingV1 "k8s.io/api/networking/v1"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// KubernetesNetworkPolicyWrap wraps a k8s network policy so you can convert it to a proto network policy
type KubernetesNetworkPolicyWrap struct {
	*networkingV1.NetworkPolicy
}

// ToYaml produces a string holding a JSON formatted yaml for the network policy.
func (np KubernetesNetworkPolicyWrap) ToYaml() (string, error) {
	// Kubernetes added a 'status' field for NetworkPolicies in 1.24. See:
	// * (https://github.com/kubernetes/kubernetes/blob/master/CHANGELOG/CHANGELOG-1.24.md#api-change-3)
	// * (https://github.com/kubernetes/kubernetes/pull/107963)
	// Using the `NewSerializerWithOptions` from `k8s.io/apimachinery/pkg/runtime/serializer/json` with a `v1.NetworkPolicy` will return the `status` field.
	// The problem is: Old cluster using kubernetes < 1.24 will fail to apply NetworkPolicies generated this way.
	// Since the `status` field is not handled by ACS, we delete the field manually if present here.
	// This code might not be necessary in the future since the feature was withdrawn by the sig-network. See:
	// * (https://github.com/kubernetes/enhancements/tree/master/keps/sig-network/2943-networkpolicy-status#implementation-history)
	// * (https://github.com/kubernetes/kubernetes/pull/107963#issuecomment-1400220883)
	uObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(np.NetworkPolicy)
	if err != nil {
		return "", err
	}
	delete(uObj, "status")

	encoder := json.NewSerializerWithOptions(json.DefaultMetaFactory, nil, nil, json.SerializerOptions{
		Yaml: true,
	})

	stringBuilder := &strings.Builder{}
	err = encoder.Encode(&unstructured.Unstructured{Object: uObj}, stringBuilder)
	if err != nil {
		return "", err
	}
	return stringBuilder.String(), nil
}

// ToRoxNetworkPolicy converts a k8s network policy to a proto network policy
// This code allows for our tests to call the conversion on k8s network policies
func (np KubernetesNetworkPolicyWrap) ToRoxNetworkPolicy() *storage.NetworkPolicy {
	nps := &storage.NetworkPolicySpec{}
	nps.SetPodSelector(np.convertSelector(&np.Spec.PodSelector))
	nps.SetIngress(np.convertIngressRules(np.Spec.Ingress))
	nps.SetEgress(np.convertEgressRules(np.Spec.Egress))
	nps.SetPolicyTypes(k8sPolicyTypesToRox(&np.Spec))
	np2 := &storage.NetworkPolicy{}
	np2.SetId(string(np.GetUID()))
	np2.SetName(np.GetName())
	np2.SetNamespace(np.GetNamespace())
	np2.SetLabels(np.GetLabels())
	np2.SetAnnotations(np.GetAnnotations())
	np2.SetCreated(protoconv.ConvertTimeToTimestamp(np.GetCreationTimestamp().Time))
	np2.SetApiVersion(np.APIVersion)
	np2.SetSpec(nps)
	return np2
}

func (np KubernetesNetworkPolicyWrap) convertSelector(sel *k8sMetaV1.LabelSelector) *storage.LabelSelector {
	convertedSel, err := k8s.ToRoxLabelSelector(sel)
	if err != nil {
		log.Warnf("Failed to convert label selector: %v", err)
	}
	return convertedSel
}

func (np KubernetesNetworkPolicyWrap) convertProtocol(p *k8sCoreV1.Protocol) storage.Protocol {
	if p == nil {
		return storage.Protocol_UNSET_PROTOCOL
	}
	switch *p {
	case k8sCoreV1.ProtocolUDP:
		return storage.Protocol_UDP_PROTOCOL
	case k8sCoreV1.ProtocolTCP:
		return storage.Protocol_TCP_PROTOCOL
	default:
		log.Warnf("Network protocol %s is not handled", *p)
		return storage.Protocol_UNSET_PROTOCOL
	}
}

func (np KubernetesNetworkPolicyWrap) convertPorts(k8sPorts []networkingV1.NetworkPolicyPort) []*storage.NetworkPolicyPort {
	ports := make([]*storage.NetworkPolicyPort, 0, len(k8sPorts))
	for _, p := range k8sPorts {
		netPolPort := &storage.NetworkPolicyPort{}
		netPolPort.SetProtocol(np.convertProtocol(p.Protocol))
		if p.Port != nil {
			switch p.Port.Type {
			case intstr.Int:
				netPolPort.SetPort(p.Port.IntVal)
			case intstr.String:
				netPolPort.SetPortName(p.Port.StrVal)
			default:
				utils.Should(errors.Errorf(
					"UNEXPECTED: port IntOrStr %+v is neither int nor string, treating as no port spec", p.Port))
			}
		}
		ports = append(ports, netPolPort)
	}
	return ports
}

func (np KubernetesNetworkPolicyWrap) convertIPBlock(ipBlock *networkingV1.IPBlock) *storage.IPBlock {
	if ipBlock == nil {
		return nil
	}
	iPBlock := &storage.IPBlock{}
	iPBlock.SetCidr(ipBlock.CIDR)
	iPBlock.SetExcept(ipBlock.Except)
	return iPBlock
}

func (np KubernetesNetworkPolicyWrap) convertNetworkPolicyPeer(k8sPeers []networkingV1.NetworkPolicyPeer) []*storage.NetworkPolicyPeer {
	peers := make([]*storage.NetworkPolicyPeer, 0, len(k8sPeers))
	for _, peer := range k8sPeers {
		npp := &storage.NetworkPolicyPeer{}
		npp.SetPodSelector(np.convertSelector(peer.PodSelector))
		npp.SetNamespaceSelector(np.convertSelector(peer.NamespaceSelector))
		npp.SetIpBlock(np.convertIPBlock(peer.IPBlock))
		peers = append(peers, npp)
	}
	return peers
}

func (np KubernetesNetworkPolicyWrap) convertIngressRules(k8sIngressRules []networkingV1.NetworkPolicyIngressRule) []*storage.NetworkPolicyIngressRule {
	if k8sIngressRules == nil {
		return nil
	}
	ingressRules := make([]*storage.NetworkPolicyIngressRule, 0, len(k8sIngressRules))
	for _, rule := range k8sIngressRules {
		npir := &storage.NetworkPolicyIngressRule{}
		npir.SetPorts(np.convertPorts(rule.Ports))
		npir.SetFrom(np.convertNetworkPolicyPeer(rule.From))
		ingressRules = append(ingressRules, npir)
	}
	return ingressRules
}

func (np KubernetesNetworkPolicyWrap) convertEgressRules(k8sEgressRules []networkingV1.NetworkPolicyEgressRule) []*storage.NetworkPolicyEgressRule {
	if k8sEgressRules == nil {
		return nil
	}
	egressRules := make([]*storage.NetworkPolicyEgressRule, 0, len(k8sEgressRules))
	for _, rule := range k8sEgressRules {
		nper := &storage.NetworkPolicyEgressRule{}
		nper.SetPorts(np.convertPorts(rule.Ports))
		nper.SetTo(np.convertNetworkPolicyPeer(rule.To))
		egressRules = append(egressRules, nper)
	}
	return egressRules
}

func k8sPolicyTypesToRox(spec *networkingV1.NetworkPolicySpec) []storage.NetworkPolicyType {
	if spec.PolicyTypes == nil {
		return k8sSpectoPolicyTypes(spec)
	}
	types := make([]storage.NetworkPolicyType, 0, len(spec.PolicyTypes))
	for _, t := range spec.PolicyTypes {
		types = append(types, k8sPolicyTypeToRox(t))
	}
	return types
}

func k8sPolicyTypeToRox(t networkingV1.PolicyType) storage.NetworkPolicyType {
	switch t {
	case networkingV1.PolicyTypeIngress:
		return storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE
	case networkingV1.PolicyTypeEgress:
		return storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE
	default:
		log.Warnf("network policy type %s is not handled", t)
		return storage.NetworkPolicyType_UNSET_NETWORK_POLICY_TYPE
	}
}

// https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.11/#networkpolicyspec-v1beta1-extensions
// If not already filled we can imply the type from the rules that are present.
func k8sSpectoPolicyTypes(spec *networkingV1.NetworkPolicySpec) (output []storage.NetworkPolicyType) {
	if spec.Egress != nil {
		output = append(output, storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE)
	}
	output = append(output, storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE)
	return
}
