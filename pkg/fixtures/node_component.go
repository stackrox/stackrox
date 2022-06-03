package fixtures

import (
	"github.com/stackrox/rox/generated/storage"
)

func GetEmbeddedLinux_5_10_47_linuxkit() *storage.EmbeddedNodeScanComponent {
	return &storage.EmbeddedNodeScanComponent{
		Name:    "kernel",
		Version: "5.10.47-linuxkit",
		Vulns: []*storage.EmbeddedVulnerability{
			GetEmbeddedNodeCVE_2018_16880(),
			GetEmbeddedNodeCVE_2018_1000026(),
			GetEmbeddedNodeCVE_2019_3016(),
			GetEmbeddedNodeCVE_2019_3819(),
			GetEmbeddedNodeCVE_2020_16120(),
			GetEmbeddedNodeCVE_2020_35508(),
			GetEmbeddedNodeCVE_2021_20194(),
			GetEmbeddedNodeCVE_2021_46283(),
			GetEmbeddedNodeCVE_2022_0185(),
			GetEmbeddedNodeCVE_2022_30594(),
		},
		SetTopCvss: &storage.EmbeddedNodeScanComponent_TopCvss{TopCvss: 8.4},
	}
}

func GetEmbeddedKubelet_v1_21_5() *storage.EmbeddedNodeScanComponent {
	return &storage.EmbeddedNodeScanComponent{
		Name:    "kubelet",
		Version: "v1.21.5",
	}
}

func GetEmbeddedKube_Proxy_v1_21_5() *storage.EmbeddedNodeScanComponent {
	return &storage.EmbeddedNodeScanComponent{
		Name:    "kube-proxy",
		Version: "v1.21.5",
	}
}

func GetEmbeddedDocker_20_10_10() *storage.EmbeddedNodeScanComponent {
	return &storage.EmbeddedNodeScanComponent{
		Name:    "docker",
		Version: "20.10.10",
	}
}

