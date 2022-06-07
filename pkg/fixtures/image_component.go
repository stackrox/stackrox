package fixtures

import (
	"github.com/stackrox/rox/generated/storage"
)

func getEmbeddedLibzstd_1_3_8() *storage.EmbeddedImageScanComponent {
	return &storage.EmbeddedImageScanComponent{
		Name:          "libzstd",
		Version:       "1.3.8+dfsg-3+deb10u2",
		HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 0},
	}
}

func getEmbeddedLsb_10_2019051400() *storage.EmbeddedImageScanComponent {
	return &storage.EmbeddedImageScanComponent{
		Name:          "lsb",
		Version:       "10.2019051400",
		HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 0},
	}
}

func getEmbeddedLibnetfilterConntrack_1_0_7_1() *storage.EmbeddedImageScanComponent {
	return &storage.EmbeddedImageScanComponent{
		Name:          "libnetfilter-conntrack",
		Version:       "1.0.7-1",
		HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 0},
	}
}

func getEmbeddedBasePasswd_3_5_46() *storage.EmbeddedImageScanComponent {
	return &storage.EmbeddedImageScanComponent{
		Name:          "base-passwd",
		Version:       "3.5.46",
		HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 0},
		Executables: []*storage.EmbeddedImageScanComponent_Executable{
			{
				Path: "/usr/sbin/update-passwd",
				Dependencies: []string{
					"YmFzZS1wYXNzd2Q:My41LjQ2",
					"Z2xpYmM:Mi4yOC0xMA",
					"Y2RlYmNvbmY:MC4yNDk",
				},
			},
		},
	}
}

func getEmbeddedAdduser_3_118() *storage.EmbeddedImageScanComponent {
	return &storage.EmbeddedImageScanComponent{
		Name:          "adduser",
		Version:       "3.118",
		HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 0},
		Executables: []*storage.EmbeddedImageScanComponent_Executable{
			{
				Path: "/usr/sbin/adduser",
				Dependencies: []string{
					"YWRkdXNlcg:My4xMTg",
				},
			},
			{
				Path: "/usr/sbin/deluser",
				Dependencies: []string{
					"YWRkdXNlcg:My4xMTg",
				},
			},
		},
	}
}
