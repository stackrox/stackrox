package fixtures

import (
	"github.com/stackrox/rox/generated/storage"
)

func GetEmbeddedLibzstd_1_3_8() *storage.EmbeddedImageScanComponent {
	return &storage.EmbeddedImageScanComponent{
		Name:          "libzstd",
		Version:       "1.3.8+dfsg-3+deb10u2",
		HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 0},
	}
}

func GetEmbeddedLsb_10_2019051400() *storage.EmbeddedImageScanComponent {
	return &storage.EmbeddedImageScanComponent{
		Name:          "lsb",
		Version:       "10.2019051400",
		HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 0},
	}
}

func GetEmbeddedLibnetfilterConntrack_1_0_7_1() *storage.EmbeddedImageScanComponent {
	return &storage.EmbeddedImageScanComponent{
		Name:          "libnetfilter-conntrack",
		Version:       "1.0.7-1",
		HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 0},
	}
}

func GetEmbeddedBasePasswd_3_5_46() *storage.EmbeddedImageScanComponent {
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

func GetEmbeddedAdduser_3_118() *storage.EmbeddedImageScanComponent {
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

func GetEmbeddedOpenSSL_1_1_1d() *storage.EmbeddedImageScanComponent {
	return &storage.EmbeddedImageScanComponent{
		Name:    "openssl",
		Version: "1.1.1d-0+deb10u6",
		Vulns: []*storage.EmbeddedVulnerability{
			GetEmbeddedImageCVE_2007_6755(),
			GetEmbeddedImageCVE_2010_0928(),
			GetEmbeddedImageCVE_2021_3711(),
			GetEmbeddedImageCVE_2021_3712(),
			GetEmbeddedImageCVE_2021_4160(),
			GetEmbeddedImageCVE_2022_0778(),
			GetEmbeddedImageCVE_2022_1292(),
		},
		HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 0},
		SetTopCvss:    &storage.EmbeddedImageScanComponent_TopCvss{TopCvss: 9.8},
		RiskScore:     1.7815,
		FixedBy:       "1.1.1n-0+deb10u2",
	}
}

func GetEmbeddedKmod_26_1() *storage.EmbeddedImageScanComponent {
	return &storage.EmbeddedImageScanComponent{
		Name:          "kmod",
		Version:       "26-1",
		HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 0},
		Executables: []*storage.EmbeddedImageScanComponent_Executable{
			{
				Path: "/etc/init.d/kmod",
				Dependencies: []string{
					"a21vZA:MjYtMQ",
				},
			},
			{
				Path: "/usr/share/initramfs-tools/hooks/kmod",
				Dependencies: []string{
					"a21vZA:MjYtMQ",
				},
			},
			{
				Path: "/bin/kmod",
				Dependencies: []string{
					"a21vZA:MjYtMQ",
					"Z2xpYmM:Mi4yOC0xMA",
					"b3BlbnNzbA:MS4xLjFkLTArZGViMTB1Ng",
					"eHotdXRpbHM:NS4yLjQtMQ",
				},
			},
		},
	}
}

func GetEmbeddedZlib_1_1_2_11() *storage.EmbeddedImageScanComponent {
	return &storage.EmbeddedImageScanComponent{
		Name:    "zlib",
		Version: "1:1.2.11.dfsg-1",
		Vulns: []*storage.EmbeddedVulnerability{
			GetEmbeddedImageCVE_2018_25032(),
		},
		HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 0},
		SetTopCvss:    &storage.EmbeddedImageScanComponent_TopCvss{TopCvss: 7.5},
		RiskScore:     1.192,
		FixedBy:       "1:1.2.11.dfsg-1+deb10u1",
	}
}

func GetEmbeddedLibftnl_1_1_7_1() *storage.EmbeddedImageScanComponent {
	return &storage.EmbeddedImageScanComponent{
		Name:          "libnftnl",
		Version:       "1.1.7-1~bpo10+1",
		HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 0},
	}
}

func GetEmbeddedAttr_1_2_4_48_4() *storage.EmbeddedImageScanComponent {
	return &storage.EmbeddedImageScanComponent{
		Name:          "attr",
		Version:       "1:2.4.48-4",
		HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 0},
	}
}

func GetEmbeddedAudit_1_2_8_4_3() *storage.EmbeddedImageScanComponent {
	return &storage.EmbeddedImageScanComponent{
		Name:          "audit",
		Version:       "1:2.8.4-3",
		HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 0},
	}
}

func GetEmbeddedPerl_5_28_1_6() *storage.EmbeddedImageScanComponent {
	return &storage.EmbeddedImageScanComponent{
		Name:    "perl",
		Version: "5.28.1-6+deb10u1",
		Vulns: []*storage.EmbeddedVulnerability{
			GetEmbeddedImageCVE_2011_4116(),
			GetEmbeddedImageCVE_2020_16156(),
		},
		HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 0},
		SetTopCvss:    &storage.EmbeddedImageScanComponent_TopCvss{TopCvss: 7.8},
		RiskScore:     1.204,
		Executables: []*storage.EmbeddedImageScanComponent_Executable{
			{
				Path: "/usr/bin/perl5.28.1",
				Dependencies: []string{
					"cGVybA:NS4yOC4xLTYrZGViMTB1MQ",
				},
			},
			{
				Path: "/usr/bin/perl",
				Dependencies: []string{
					"cGVybA:NS4yOC4xLTYrZGViMTB1MQ",
					"Z2xpYmM:Mi4yOC0xMA",
				},
			},
		},
	}
}
