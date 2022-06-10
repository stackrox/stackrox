package fixtures

import (
	"github.com/stackrox/rox/generated/storage"
)

func GetEmbeddedImageComponentAdduser_3_118() *storage.EmbeddedImageScanComponent {
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

func GetEmbeddedImageComponentAttr_1_2_4_48_4() *storage.EmbeddedImageScanComponent {
	return &storage.EmbeddedImageScanComponent{
		Name:          "attr",
		Version:       "1:2.4.48-4",
		HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 0},
	}
}

func GetEmbeddedImageComponentAudit_1_2_8_4_3() *storage.EmbeddedImageScanComponent {
	return &storage.EmbeddedImageScanComponent{
		Name:          "audit",
		Version:       "1:2.8.4-3",
		HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 0},
	}
}

func GetEmbeddedImageComponentBasePasswd_3_5_46() *storage.EmbeddedImageScanComponent {
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

func GetEmbeddedImageComponentGeoIP_1_6_9_4() *storage.EmbeddedImageScanComponent {
	return &storage.EmbeddedImageScanComponent{
		Name:          "geoip",
		Version:       "1.6.9-4",
		HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 5},
		SetTopCvss:    &storage.EmbeddedImageScanComponent_TopCvss{TopCvss: 0.0},
		RiskScore:     0.0,
		Executables:   []*storage.EmbeddedImageScanComponent_Executable{},
	}
}

func GetEmbeddedImageComponentGLibC_2_24_11() *storage.EmbeddedImageScanComponent {
	return &storage.EmbeddedImageScanComponent{
		Name:    "glibc",
		Version: "2.24-11+deb9u4",
		Vulns: []*storage.EmbeddedVulnerability{
			GetEmbeddedImageCVE_2009_5155(),
			GetEmbeddedImageCVE_2010_4756(),
			GetEmbeddedImageCVE_2015_8985(),
			GetEmbeddedImageCVE_2016_10228(),
			GetEmbeddedImageCVE_2016_10739(),
			GetEmbeddedImageCVE_2017_12132(),
			GetEmbeddedImageCVE_2018_6485(),
			GetEmbeddedImageCVE_2018_6551(),
			GetEmbeddedImageCVE_2018_20796(),
			GetEmbeddedImageCVE_2018_1000001(),
			GetEmbeddedImageCVE_2019_6488(),
			GetEmbeddedImageCVE_2019_7309(),
			GetEmbeddedImageCVE_2019_9169(),
			GetEmbeddedImageCVE_2019_9192(),
			GetEmbeddedImageCVE_2019_19126(),
			GetEmbeddedImageCVE_2019_25013(),
			GetEmbeddedImageCVE_2019_1010022(),
			GetEmbeddedImageCVE_2019_1010023(),
			GetEmbeddedImageCVE_2019_1010024(),
			GetEmbeddedImageCVE_2019_1010025(),
			GetEmbeddedImageCVE_2020_1751(),
			GetEmbeddedImageCVE_2020_1752(),
			GetEmbeddedImageCVE_2020_6096(),
			GetEmbeddedImageCVE_2020_10029(),
			GetEmbeddedImageCVE_2020_27618(),
			GetEmbeddedImageCVE_2021_3326(),
			GetEmbeddedImageCVE_2021_27645(),
			GetEmbeddedImageCVE_2021_33574(),
			GetEmbeddedImageCVE_2021_35942(),
			GetEmbeddedImageCVE_2022_23218(),
			GetEmbeddedImageCVE_2022_23219(),
		},
		HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 0},
		SetTopCvss:    &storage.EmbeddedImageScanComponent_TopCvss{TopCvss: 9.8},
		RiskScore:     4.0,
		Executables: []*storage.EmbeddedImageScanComponent_Executable{
			{
				Path: "/sbin/ldconfig",
				Dependencies: []string{
					"Z2xpYmM:Mi4yNC0xMStkZWI5dTQ",
				},
			},
			{
				Path: "/usr/bin/locale",
				Dependencies: []string{
					"Z2xpYmM:Mi4yNC0xMStkZWI5dTQ",
				},
			},
			{
				Path: "/usr/sbin/iconvconfig",
				Dependencies: []string{
					"Z2xpYmM:Mi4yNC0xMStkZWI5dTQ",
				},
			},
			{
				Path: "/usr/bin/iconv",
				Dependencies: []string{
					"Z2xpYmM:Mi4yNC0xMStkZWI5dTQ",
				},
			},
			{
				Path: "/lib/x86_64-linux-gnu/ld-linux-x86-64.so.2",
				Dependencies: []string{
					"Z2xpYmM:Mi4yNC0xMStkZWI5dTQ",
				},
			},
			{
				Path: "/usr/bin/localedef",
				Dependencies: []string{
					"Z2xpYmM:Mi4yNC0xMStkZWI5dTQ",
				},
			},
			{
				Path: "/lib/x86_64-linux-gnu/libc.so.6",
				Dependencies: []string{
					"Z2xpYmM:Mi4yNC0xMStkZWI5dTQ",
				},
			},
			{
				Path: "/lib/x86_64-linux-gnu/libpthread.so.0",
				Dependencies: []string{
					"Z2xpYmM:Mi4yNC0xMStkZWI5dTQ",
				},
			},
			{
				Path: "/usr/bin/catchsegv",
				Dependencies: []string{
					"Z2xpYmM:Mi4yNC0xMStkZWI5dTQ",
				},
			},
			{
				Path: "/usr/bin/ldd",
				Dependencies: []string{
					"Z2xpYmM:Mi4yNC0xMStkZWI5dTQ",
				},
			},
			{
				Path: "/usr/bin/zdump",
				Dependencies: []string{
					"Z2xpYmM:Mi4yNC0xMStkZWI5dTQ",
				},
			},
			{
				Path: "/usr/bin/getent",
				Dependencies: []string{
					"Z2xpYmM:Mi4yNC0xMStkZWI5dTQ",
				},
			},
			{
				Path: "/lib/x86_64-linux-gnu/ld-2.24.so",
				Dependencies: []string{
					"Z2xpYmM:Mi4yNC0xMStkZWI5dTQ",
				},
			},
			{
				Path: "/lib/x86_64-linux-gnu/libc-2.24.so",
				Dependencies: []string{
					"Z2xpYmM:Mi4yNC0xMStkZWI5dTQ",
				},
			},
			{
				Path: "/usr/bin/pldd",
				Dependencies: []string{
					"Z2xpYmM:Mi4yNC0xMStkZWI5dTQ",
				},
			},
			{
				Path: "/lib/x86_64-linux-gnu/libpthread-2.24.so",
				Dependencies: []string{
					"Z2xpYmM:Mi4yNC0xMStkZWI5dTQ",
				},
			},
			{
				Path: "/lib64/ld-linux-x86-64.so.2",
				Dependencies: []string{
					"Z2xpYmM:Mi4yNC0xMStkZWI5dTQ",
				},
			},
			{
				Path: "/usr/bin/getconf",
				Dependencies: []string{
					"Z2xpYmM:Mi4yNC0xMStkZWI5dTQ",
				},
			},
			{
				Path: "/usr/bin/tzselect",
				Dependencies: []string{
					"Z2xpYmM:Mi4yNC0xMStkZWI5dTQ",
				},
			},
			{
				Path: "/usr/sbin/zic",
				Dependencies: []string{
					"Z2xpYmM:Mi4yNC0xMStkZWI5dTQ",
				},
			},
		},
	}
}

func GetEmbeddedImageComponentKmod_26_1() *storage.EmbeddedImageScanComponent {
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

func GetEmbeddedImageComponentLibftnl_1_1_7_1() *storage.EmbeddedImageScanComponent {
	return &storage.EmbeddedImageScanComponent{
		Name:          "libnftnl",
		Version:       "1.1.7-1~bpo10+1",
		HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 0},
	}
}

func GetEmbeddedImageComponentLibnetfilterConntrack_1_0_7_1() *storage.EmbeddedImageScanComponent {
	return &storage.EmbeddedImageScanComponent{
		Name:          "libnetfilter-conntrack",
		Version:       "1.0.7-1",
		HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 0},
	}
}

func GetEmbeddedImageComponentLibXSLT_1_1_29_2_1() *storage.EmbeddedImageScanComponent {
	return &storage.EmbeddedImageScanComponent{
		Name:    "libxslt",
		Version: "1.1.29-2.1",
		Vulns: []*storage.EmbeddedVulnerability{
			GetEmbeddedImageCVE_2015_9019(),
			GetEmbeddedImageCVE_2019_11068(),
			GetEmbeddedImageCVE_2019_13117(),
			GetEmbeddedImageCVE_2019_13118(),
			GetEmbeddedImageCVE_2019_18197(),
		},
		HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 5},
		SetTopCvss:    &storage.EmbeddedImageScanComponent_TopCvss{TopCvss: 9.8},
		RiskScore:     1.409875,
		Executables:   []*storage.EmbeddedImageScanComponent_Executable{},
		FixedBy:       "1.1.29-2.1+deb9u2",
	}
}

func GetEmbeddedImageComponentLibzstd_1_3_8() *storage.EmbeddedImageScanComponent {
	return &storage.EmbeddedImageScanComponent{
		Name:          "libzstd",
		Version:       "1.3.8+dfsg-3+deb10u2",
		HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 0},
	}
}

func GetEmbeddedImageComponentLsb_10_2019051400() *storage.EmbeddedImageScanComponent {
	return &storage.EmbeddedImageScanComponent{
		Name:          "lsb",
		Version:       "10.2019051400",
		HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 0},
	}
}

func GetEmbeddedImageComponentLz4_0_0() *storage.EmbeddedImageScanComponent {
	return &storage.EmbeddedImageScanComponent{
		Name:    "lz4",
		Version: "0.0~r131-2",
		Vulns: []*storage.EmbeddedVulnerability{
			GetEmbeddedImageCVE_2019_17543(),
			GetEmbeddedImageCVE_2021_3520(),
		},
		HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 0},
		SetTopCvss:    &storage.EmbeddedImageScanComponent_TopCvss{TopCvss: 9.8},
		RiskScore:     1.28275,
		Executables:   []*storage.EmbeddedImageScanComponent_Executable{},
		FixedBy:       "0.0~r131-2+deb9u1",
	}
}

func GetEmbeddedImageComponentNginX_1_14_2_1() *storage.EmbeddedImageScanComponent {
	return &storage.EmbeddedImageScanComponent{
		Name:    "nginx",
		Version: "1.14.2-1~stretch",
		Vulns: []*storage.EmbeddedVulnerability{
			GetEmbeddedImageCVE_2009_4487(),
			GetEmbeddedImageCVE_2013_0337(),
			GetEmbeddedImageCVE_2020_36309(),
			GetEmbeddedImageCVE_2021_3618(),
		},
		HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 5},
		SetTopCvss:    &storage.EmbeddedImageScanComponent_TopCvss{TopCvss: 7.5},
		RiskScore:     1.30675,
		Executables: []*storage.EmbeddedImageScanComponent_Executable{
			{
				Path: "/etc/init.d/nginx",
				Dependencies: []string{
					"bmdpbng:MS4xNC4yLTF-c3RyZXRjaA",
				},
			},
			{
				Path: "/etc/init.d/nginx-debug",
				Dependencies: []string{
					"bmdpbng:MS4xNC4yLTF-c3RyZXRjaA",
				},
			},
			{
				Path: "/usr/sbin/nginx",
				Dependencies: []string{
					"emxpYg:MToxLjIuOC5kZnNnLTU",
					"bmdpbng:MS4xNC4yLTF-c3RyZXRjaA",
					"Z2xpYmM:Mi4yNC0xMStkZWI5dTQ",
					"cGNyZTM:Mjo4LjM5LTM",
					"b3BlbnNzbA:MS4xLjBqLTF-ZGViOXUx",
				},
			},
			{
				Path: "/usr/sbin/nginx-debug",
				Dependencies: []string{
					"emxpYg:MToxLjIuOC5kZnNnLTU",
					"bmdpbng:MS4xNC4yLTF-c3RyZXRjaA",
					"Z2xpYmM:Mi4yNC0xMStkZWI5dTQ",
					"b3BlbnNzbA:MS4xLjBqLTF-ZGViOXUx",
					"cGNyZTM:Mjo4LjM5LTM",
				},
			},
		},
	}
}

func GetEmbeddedImageComponentNginX_Module_GeoIP_1_14_2_1() *storage.EmbeddedImageScanComponent {
	return &storage.EmbeddedImageScanComponent{
		Name:          "nginx-module-geoip",
		Version:       "1.14.2-1~stretch",
		HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 5},
		SetTopCvss:    &storage.EmbeddedImageScanComponent_TopCvss{TopCvss: 0.0},
		RiskScore:     0.0,
		Executables:   []*storage.EmbeddedImageScanComponent_Executable{},
	}
}

func GetEmbeddedImageComponentNginX_Module_Image_Filter_1_14_2_1() *storage.EmbeddedImageScanComponent {
	return &storage.EmbeddedImageScanComponent{
		Name:          "nginx-module-image-filter",
		Version:       "1.14.2-1~stretch",
		HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 5},
		SetTopCvss:    &storage.EmbeddedImageScanComponent_TopCvss{TopCvss: 0.0},
		RiskScore:     0.0,
		Executables:   []*storage.EmbeddedImageScanComponent_Executable{},
	}
}

func GetEmbeddedImageComponentNginX_Module_NJS_1_14_2_0_2_6() *storage.EmbeddedImageScanComponent {
	return &storage.EmbeddedImageScanComponent{
		Name:          "nginx-module-njs",
		Version:       "1.14.2.0.2.6-1~stretch",
		HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 5},
		SetTopCvss:    &storage.EmbeddedImageScanComponent_TopCvss{TopCvss: 0.0},
		RiskScore:     0.0,
		Executables: []*storage.EmbeddedImageScanComponent_Executable{
			{
				Path: "/usr/bin/njs",
				Dependencies: []string{
					"bGliYnNk:MC44LjMtMQ",
					"bmN1cnNlcw:Ni4wKzIwMTYxMTI2LTErZGViOXUy",
					"bmdpbngtbW9kdWxlLW5qcw:MS4xNC4yLjAuMi42LTF-c3RyZXRjaA",
					"cGNyZTM:Mjo4LjM5LTM",
					"Z2xpYmM:Mi4yNC0xMStkZWI5dTQ",
					"bGliZWRpdA:My4xLTIwMTYwOTAzLTM",
				},
			},
		},
	}
}

func GetEmbeddedImageComponentNginX_Module_XSLT_1_14_2_1() *storage.EmbeddedImageScanComponent {
	return &storage.EmbeddedImageScanComponent{
		Name:          "nginx-module-xslt",
		Version:       "1.14.2-1~stretch",
		HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 5},
		SetTopCvss:    &storage.EmbeddedImageScanComponent_TopCvss{TopCvss: 0.0},
		RiskScore:     0.0,
		Executables:   []*storage.EmbeddedImageScanComponent_Executable{},
	}
}

func GetEmbeddedImageComponentOpenSSL_1_1_0j_1() *storage.EmbeddedImageScanComponent {
	return &storage.EmbeddedImageScanComponent{
		Name:    "openssl",
		Version: "1.1.0j-1~deb9u1",
		Vulns: []*storage.EmbeddedVulnerability{
			GetEmbeddedImageCVE_2007_6755(),
			GetEmbeddedImageCVE_2010_0928(),
			GetEmbeddedImageCVE_2019_1543(),
			GetEmbeddedImageCVE_2019_1547(),
			GetEmbeddedImageCVE_2019_1551(),
			GetEmbeddedImageCVE_2019_1563(),
			GetEmbeddedImageCVE_2020_1971(),
			GetEmbeddedImageCVE_2021_3712(),
			GetEmbeddedImageCVE_2021_4160(),
			GetEmbeddedImageCVE_2021_23840(),
			GetEmbeddedImageCVE_2021_23841(),
			GetEmbeddedImageCVE_2022_0778(),
			GetEmbeddedImageCVE_2022_1292(),
		},
		HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 5},
		SetTopCvss:    &storage.EmbeddedImageScanComponent_TopCvss{TopCvss: 9.8},
		RiskScore:     2.2697499,
		FixedBy:       "1.1.0l-1~deb9u6",
	}
}

func GetEmbeddedImageComponentOpenSSL_1_1_1d() *storage.EmbeddedImageScanComponent {
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

func GetEmbeddedImageComponentPcre3_2_8_39_3() *storage.EmbeddedImageScanComponent {
	return &storage.EmbeddedImageScanComponent{
		Name:    "pcre3",
		Version: "2:8.39-3",
		Vulns: []*storage.EmbeddedVulnerability{
			GetEmbeddedImageCVE_2017_7245(),
			GetEmbeddedImageCVE_2017_7246(),
			GetEmbeddedImageCVE_2017_11164(),
			GetEmbeddedImageCVE_2017_16231(),
			GetEmbeddedImageCVE_2019_20838(),
			GetEmbeddedImageCVE_2020_14155(),
		},
		HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 0},
		SetTopCvss:    &storage.EmbeddedImageScanComponent_TopCvss{TopCvss: 7.8},
		RiskScore:     1.15075,
	}
}

func GetEmbeddedImageComponentPerl_5_28_1_6() *storage.EmbeddedImageScanComponent {
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

func GetEmbeddedImageComponentPam_1_1_8_3_6() *storage.EmbeddedImageScanComponent {
	return &storage.EmbeddedImageScanComponent{
		Name:          "pam",
		Version:       "1.1.8-3.6",
		HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 0},
		SetTopCvss:    &storage.EmbeddedImageScanComponent_TopCvss{TopCvss: 0.0},
		RiskScore:     0.0,
		Executables: []*storage.EmbeddedImageScanComponent_Executable{
			{
				Path: "/sbin/mkhomedir_helper",
				Dependencies: []string{
					"cGFt:MS4xLjgtMy42",
					"Z2xpYmM:Mi4yNC0xMStkZWI5dTQ",
					"YXVkaXQ:MToyLjYuNy0y",
					"bGliY2FwLW5n:MC43LjctMw",
				},
			},
			{
				Path: "/sbin/pam_tally",
				Dependencies: []string{
					"cGFt:MS4xLjgtMy42",
					"Z2xpYmM:Mi4yNC0xMStkZWI5dTQ",
				},
			},
			{
				Path: "/sbin/unix_chkpwd",
				Dependencies: []string{
					"YXVkaXQ:MToyLjYuNy0y",
					"bGliY2FwLW5n:MC43LjctMw",
					"cGFt:MS4xLjgtMy42",
					"Z2xpYmM:Mi4yNC0xMStkZWI5dTQ",
					"bGlic2VsaW51eA:Mi42LTM",
					"cGNyZTM:Mjo4LjM5LTM",
				},
			},
			{
				Path: "/etc/security/namespace.init",
				Dependencies: []string{
					"cGFt:MS4xLjgtMy42",
				},
			},
			{
				Path: "/sbin/pam_tally2",
				Dependencies: []string{
					"bGliY2FwLW5n:MC43LjctMw",
					"cGFt:MS4xLjgtMy42",
					"YXVkaXQ:MToyLjYuNy0y",
					"Z2xpYmM:Mi4yNC0xMStkZWI5dTQ",
				},
			},
			{
				Path: "/sbin/unix_update",
				Dependencies: []string{
					"cGFt:MS4xLjgtMy42",
					"cGNyZTM:Mjo4LjM5LTM",
					"bGlic2VsaW51eA:Mi42LTM",
					"Z2xpYmM:Mi4yNC0xMStkZWI5dTQ",
				},
			},
			{
				Path: "/usr/sbin/pam-auth-update",
				Dependencies: []string{
					"cGFt:MS4xLjgtMy42",
				},
			},
			{
				Path: "/usr/sbin/pam_getenv",
				Dependencies: []string{
					"cGFt:MS4xLjgtMy42",
				},
			},
			{
				Path: "/usr/sbin/pam_timestamp_check",
				Dependencies: []string{
					"Z2xpYmM:Mi4yNC0xMStkZWI5dTQ",
					"YXVkaXQ:MToyLjYuNy0y",
					"bGliY2FwLW5n:MC43LjctMw",
					"cGFt:MS4xLjgtMy42",
				},
			},
		},
	}
}

func GetEmbeddedImageComponentZlib_1_1_2_8() *storage.EmbeddedImageScanComponent {
	return &storage.EmbeddedImageScanComponent{
		Name:    "zlib",
		Version: "1:1.2.8.dfsg-5",
		Vulns: []*storage.EmbeddedVulnerability{
			GetEmbeddedImageCVE_2018_25032(),
		},
		HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 0},
		SetTopCvss:    &storage.EmbeddedImageScanComponent_TopCvss{TopCvss: 7.5},
		RiskScore:     1.192,
		Executables:   []*storage.EmbeddedImageScanComponent_Executable{},
		FixedBy:       "1:1.2.8.dfsg-5+deb9u1",
	}
}

func GetEmbeddedImageComponentZlib_1_1_2_11() *storage.EmbeddedImageScanComponent {
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
