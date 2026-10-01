package main

import (
	time "time"

	csaf "github.com/stackrox/rox/pkg/scannerv4/enricher/csaf"
)

// stackrox-rhel-csaf.json.zst native fixture relationships, transcribed from the checked-in CI bundle.
// Kept independent from expected results in scanner/e2etests and backend QA.
// Advisory enrichment for RHEL fixtures used by the TestImage RHEL scenarios.
// Preserve Red Hat advisory descriptions, release dates, severity and vectors;
// the native VEX links connect these records to the corresponding CVEs.
var stackrox_rhel_csafFixtures = []operation{
	{Member: "stackrox-rhel-csaf.json.zst", Updater: "stackrox.rhel-csaf", Enrichments: []enrichmentFixture{
		{Tags: []string{"RHSA-2020:5499"}, Payload: stackrox_rhel_csafValue1},
		{Tags: []string{"RHSA-2021:4409"}, Payload: stackrox_rhel_csafValue3},
		{Tags: []string{"RHSA-2021:0734"}, Payload: stackrox_rhel_csafValue4},
		{Tags: []string{"RHSA-2020:4950"}, Payload: stackrox_rhel_csafValue6},
		{Tags: []string{"RHSA-2026:23262"}, Payload: stackrox_rhel_csafValue8},
		{Tags: []string{"RHSA-2022:9110"}, Payload: stackrox_rhel_csafValue10},
		{Tags: []string{"RHSA-2019:3700"}, Payload: stackrox_rhel_csafValue12},
		{Tags: []string{"RHSA-2026:7261"}, Payload: stackrox_rhel_csafValue14},
		{Tags: []string{"RHSA-2026:1496"}, Payload: stackrox_rhel_csafValue15},
		{Tags: []string{"RHSA-2020:4005"}, Payload: stackrox_rhel_csafValue17},
		{Tags: []string{"RHSA-2026:36651"}, Payload: stackrox_rhel_csafValue19},
		{Tags: []string{"RHBA-2022:5749"}, Payload: stackrox_rhel_csafValue21},
		{Tags: []string{"RHSA-2021:1131"}, Payload: stackrox_rhel_csafValue23},
		{Tags: []string{"RHSA-2023:5453"}, Payload: stackrox_rhel_csafValue25},
		{Tags: []string{"RHSA-2023:3198"}, Payload: stackrox_rhel_csafValue27},
		{Tags: []string{"RHSA-2026:7661"}, Payload: stackrox_rhel_csafValue28},
		{Tags: []string{"RHSA-2026:36207"}, Payload: stackrox_rhel_csafValue30},
		{Tags: []string{"RHSA-2026:1519"}, Payload: stackrox_rhel_csafValue31},
		{Tags: []string{"RHSA-2021:0741"}, Payload: stackrox_rhel_csafValue32},
		{Tags: []string{"RHSA-2026:26546"}, Payload: stackrox_rhel_csafValue33},
		{Tags: []string{"RHSA-2024:0776"}, Payload: stackrox_rhel_csafValue34},
		{Tags: []string{"RHSA-2022:5311"}, Payload: stackrox_rhel_csafValue36},
		{Tags: []string{"RHBA-2022:8582"}, Payload: stackrox_rhel_csafValue38},
		{Tags: []string{"RHSA-2019:0910"}, Payload: stackrox_rhel_csafValue41},
		{Tags: []string{"RHSA-2023:7540"}, Payload: stackrox_rhel_csafValue43},
		{Tags: []string{"RHSA-2022:6224"}, Payload: stackrox_rhel_csafValue44},
		{Tags: []string{"RHSA-2024:3635"}, Payload: stackrox_rhel_csafValue45},
		{Tags: []string{"RHSA-2020:4464"}, Payload: stackrox_rhel_csafValue46},
		{Tags: []string{"RHSA-2020:4272"}, Payload: stackrox_rhel_csafValue47},
		{Tags: []string{"RHSA-2021:2724"}, Payload: stackrox_rhel_csafValue49},
		{Tags: []string{"RHSA-2026:37275"}, Payload: stackrox_rhel_csafValue50},
		{Tags: []string{"RHSA-2020:4907"}, Payload: stackrox_rhel_csafValue51},
		{Tags: []string{"RHSA-2026:1594"}, Payload: stackrox_rhel_csafValue52},
		{Tags: []string{"RHSA-2026:40945"}, Payload: stackrox_rhel_csafValue53},
		{Tags: []string{"RHSA-2026:30651"}, Payload: stackrox_rhel_csafValue55},
		{Tags: []string{"RHSA-2026:41036"}, Payload: stackrox_rhel_csafValue56},
		{Tags: []string{"RHSA-2026:36808"}, Payload: stackrox_rhel_csafValue57},
		{Tags: []string{"RHSA-2024:1601"}, Payload: stackrox_rhel_csafValue59},
		{Tags: []string{"RHSA-2020:1792"}, Payload: stackrox_rhel_csafValue61},
		{Tags: []string{"RHSA-2024:0778"}, Payload: stackrox_rhel_csafValue62},
		{Tags: []string{"RHSA-2021:2721"}, Payload: stackrox_rhel_csafValue63},
		{Tags: []string{"RHSA-2021:1063"}, Payload: stackrox_rhel_csafValue64},
		{Tags: []string{"RHSA-2022:5498"}, Payload: stackrox_rhel_csafValue66},
		{Tags: []string{"RHSA-2026:23264"}, Payload: stackrox_rhel_csafValue67},
		{Tags: []string{"RHSA-2021:2717"}, Payload: stackrox_rhel_csafValue68},
		{Tags: []string{"RHSA-2026:40118"}, Payload: stackrox_rhel_csafValue69},
		{Tags: []string{"RHSA-2026:36648"}, Payload: stackrox_rhel_csafValue70},
		{Tags: []string{"RHSA-2021:0548"}, Payload: stackrox_rhel_csafValue71},
		{Tags: []string{"RHSA-2020:4312"}, Payload: stackrox_rhel_csafValue72},
		{Tags: []string{"RHSA-2026:30650"}, Payload: stackrox_rhel_csafValue73},
		{Tags: []string{"RHSA-2021:0739"}, Payload: stackrox_rhel_csafValue74},
		{Tags: []string{"RHBA-2022:5747"}, Payload: stackrox_rhel_csafValue75},
		{Tags: []string{"RHSA-2026:33531"}, Payload: stackrox_rhel_csafValue76},
		{Tags: []string{"RHSA-2026:37387"}, Payload: stackrox_rhel_csafValue77},
		{Tags: []string{"RHSA-2024:3634"}, Payload: stackrox_rhel_csafValue78},
		{Tags: []string{"RHSA-2026:1473"}, Payload: stackrox_rhel_csafValue79},
		{Tags: []string{"RHSA-2022:6531"}, Payload: stackrox_rhel_csafValue80},
		{Tags: []string{"RHSA-2024:4597"}, Payload: stackrox_rhel_csafValue81},
		{Tags: []string{"RHSA-2021:1024"}, Payload: stackrox_rhel_csafValue83},
		{Tags: []string{"RHSA-2020:4508"}, Payload: stackrox_rhel_csafValue84},
		{Tags: []string{"RHSA-2021:4060"}, Payload: stackrox_rhel_csafValue85},
		{Tags: []string{"RHSA-2019:0483"}, Payload: stackrox_rhel_csafValue86},
		{Tags: []string{"RHSA-2023:6745"}, Payload: stackrox_rhel_csafValue87},
		{Tags: []string{"RHSA-2023:5455"}, Payload: stackrox_rhel_csafValue88},
		{Tags: []string{"RHSA-2021:0735"}, Payload: stackrox_rhel_csafValue89},
		{Tags: []string{"RHSA-2022:7288"}, Payload: stackrox_rhel_csafValue90},
		{Tags: []string{"RHSA-2026:36796"}, Payload: stackrox_rhel_csafValue91},
		{Tags: []string{"RHSA-2026:1733"}, Payload: stackrox_rhel_csafValue92},
		{Tags: []string{"RHSA-2022:5813"}, Payload: stackrox_rhel_csafValue94},
		{Tags: []string{"RHSA-2020:1020"}, Payload: stackrox_rhel_csafValue95},
		{Tags: []string{"RHSA-2020:4949"}, Payload: stackrox_rhel_csafValue96},
		{Tags: []string{"RHSA-2024:3203"}, Payload: stackrox_rhel_csafValue98},
		{Tags: []string{"RHSA-2020:4173"}, Payload: stackrox_rhel_csafValue99},
		{Tags: []string{"RHSA-2024:3636"}, Payload: stackrox_rhel_csafValue100},
		{Tags: []string{"RHSA-2022:6057"}, Payload: stackrox_rhel_csafValue101},
		{Tags: []string{"RHSA-2023:6292"}, Payload: stackrox_rhel_csafValue102},
		{Tags: []string{"RHSA-2026:41019"}, Payload: stackrox_rhel_csafValue103},
		{Tags: []string{"RHSA-2021:0740"}, Payload: stackrox_rhel_csafValue104},
		{Tags: []string{"RHSA-2021:4408"}, Payload: stackrox_rhel_csafValue106},
		{Tags: []string{"RHSA-2021:0744"}, Payload: stackrox_rhel_csafValue107},
		{Tags: []string{"RHSA-2026:36797"}, Payload: stackrox_rhel_csafValue108},
		{Tags: []string{"RHSA-2022:5942"}, Payload: stackrox_rhel_csafValue109},
		{Tags: []string{"RHSA-2026:36820"}, Payload: stackrox_rhel_csafValue111},
		{Tags: []string{"RHSA-2021:0551"}, Payload: stackrox_rhel_csafValue112},
		{Tags: []string{"RHSA-2021:2575"}, Payload: stackrox_rhel_csafValue113},
		{Tags: []string{"RHSA-2023:3299"}, Payload: stackrox_rhel_csafValue114},
		{Tags: []string{"RHSA-2020:2505"}, Payload: stackrox_rhel_csafValue115},
		{Tags: []string{"RHSA-2020:4952"}, Payload: stackrox_rhel_csafValue116},
		{Tags: []string{"RHSA-2026:1503"}, Payload: stackrox_rhel_csafValue117},
		{Tags: []string{"RHSA-2024:2463"}, Payload: stackrox_rhel_csafValue118},
		{Tags: []string{"RHSA-2020:4951"}, Payload: stackrox_rhel_csafValue119},
		{Tags: []string{"RHSA-2022:5818"}, Payload: stackrox_rhel_csafValue121},
		{Tags: []string{"RHSA-2021:0738"}, Payload: stackrox_rhel_csafValue122},
		{Tags: []string{"RHSA-2023:0017"}, Payload: stackrox_rhel_csafValue123},
		{Tags: []string{"RHSA-2023:5454"}, Payload: stackrox_rhel_csafValue124},
		{Tags: []string{"RHSA-2020:4903"}, Payload: stackrox_rhel_csafValue125},
		{Tags: []string{"RHSA-2023:5476"}, Payload: stackrox_rhel_csafValue126},
		{Tags: []string{"RHSA-2026:26547"}, Payload: stackrox_rhel_csafValue127},
		{Tags: []string{"RHSA-2023:5763"}, Payload: stackrox_rhel_csafValue128},
		{Tags: []string{"RHSA-2026:33524"}, Payload: stackrox_rhel_csafValue129},
		{Tags: []string{"RHSA-2021:0549"}, Payload: stackrox_rhel_csafValue130},
		{Tags: []string{"RHSA-2023:5700"}, Payload: stackrox_rhel_csafValue131},
		{Tags: []string{"RHSA-2026:1472"}, Payload: stackrox_rhel_csafValue132},
		{Tags: []string{"RHSA-2022:0246"}, Payload: stackrox_rhel_csafValue133},
	}},
}
var stackrox_rhel_csafValue0 = csaf.CVSS{Score: 7.5, Vector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H"}

var stackrox_rhel_csafValue1 = csaf.Advisory{Name: "RHSA-2020:5499", Description: "Red Hat Security Advisory: nodejs:12 security and bug fix update", ReleaseDate: time.Date(2020, 12, 15, 17, 27, 36, 0, time.UTC), Severity: "Moderate", CVSSv3: stackrox_rhel_csafValue0}

var stackrox_rhel_csafValue2 = csaf.CVSS{Score: 7.5, Vector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:N/A:N"}

var stackrox_rhel_csafValue3 = csaf.Advisory{Name: "RHSA-2021:4409", Description: "Red Hat Security Advisory: libgcrypt security and bug fix update", ReleaseDate: time.Date(2021, 11, 9, 18, 23, 25, 0, time.UTC), Severity: "Moderate", CVSSv3: stackrox_rhel_csafValue2}

var stackrox_rhel_csafValue4 = csaf.Advisory{Name: "RHSA-2021:0734", Description: "Red Hat Security Advisory: nodejs:12 security update", ReleaseDate: time.Date(2021, 3, 4, 16, 3, 58, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue0}

var stackrox_rhel_csafValue5 = csaf.CVSS{Score: 8.6, Vector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:H"}

var stackrox_rhel_csafValue6 = csaf.Advisory{Name: "RHSA-2020:4950", Description: "Red Hat Security Advisory: freetype security update", ReleaseDate: time.Date(2020, 11, 5, 8, 49, 18, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue5}

var stackrox_rhel_csafValue7 = csaf.CVSS{Score: 8.2, Vector: "CVSS:3.1/AV:N/AC:H/PR:L/UI:N/S:C/C:H/I:H/A:N"}

var stackrox_rhel_csafValue8 = csaf.Advisory{Name: "RHSA-2026:23262", Description: "Red Hat Security Advisory: Red Hat Hardened Images RPMs bug fix and enhancement update", ReleaseDate: time.Date(2026, 6, 4, 12, 39, 22, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue7}

var stackrox_rhel_csafValue9 = csaf.CVSS{Score: 7.5, Vector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:H/A:N"}

var stackrox_rhel_csafValue10 = csaf.Advisory{Name: "RHSA-2022:9110", Description: "Red Hat Security Advisory: OpenShift Container Platform 4.9.54 packages and security update", ReleaseDate: time.Date(2023, 1, 6, 8, 12, 18, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue9}

var stackrox_rhel_csafValue11 = csaf.CVSS{Score: 5.1, Vector: "CVSS:3.0/AV:L/AC:H/PR:N/UI:N/S:U/C:H/I:N/A:N"}

var stackrox_rhel_csafValue12 = csaf.Advisory{Name: "RHSA-2019:3700", Description: "Red Hat Security Advisory: openssl security, bug fix, and enhancement update", ReleaseDate: time.Date(2019, 11, 5, 22, 28, 48, 0, time.UTC), Severity: "Low", CVSSv3: stackrox_rhel_csafValue11}

var stackrox_rhel_csafValue13 = csaf.CVSS{Score: 9.8, Vector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H"}

var stackrox_rhel_csafValue14 = csaf.Advisory{Name: "RHSA-2026:7261", Description: "Red Hat Security Advisory: Red Hat Hardened Images RPMs bug fix and enhancement update", ReleaseDate: time.Date(2026, 4, 9, 8, 50, 10, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue13}

var stackrox_rhel_csafValue15 = csaf.Advisory{Name: "RHSA-2026:1496", Description: "Red Hat Security Advisory: openssl security update", ReleaseDate: time.Date(2026, 1, 28, 15, 32, 54, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue13}

var stackrox_rhel_csafValue16 = csaf.CVSS{Score: 7.5, Vector: "CVSS:3.1/AV:N/AC:H/PR:N/UI:R/S:U/C:H/I:H/A:H"}

var stackrox_rhel_csafValue17 = csaf.Advisory{Name: "RHSA-2020:4005", Description: "Red Hat Security Advisory: libxslt security update", ReleaseDate: time.Date(2020, 9, 29, 19, 54, 52, 0, time.UTC), Severity: "Moderate", CVSSv3: stackrox_rhel_csafValue16}

var stackrox_rhel_csafValue18 = csaf.CVSS{Score: 9.1, Vector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:N"}

var stackrox_rhel_csafValue19 = csaf.Advisory{Name: "RHSA-2026:36651", Description: "Red Hat Security Advisory: Red Hat Edge Manager Version 1.0.3 Security Update", ReleaseDate: time.Date(2026, 7, 8, 10, 8, 44, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue18}

var stackrox_rhel_csafValue20 = csaf.CVSS{Score: 9.3, Vector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:C/C:H/I:H/A:N"}

var stackrox_rhel_csafValue21 = csaf.Advisory{Name: "RHBA-2022:5749", Description: "Red Hat Bug Fix Advisory: .NET 6.0 bugfix update", ReleaseDate: time.Date(2022, 7, 28, 10, 19, 2, 0, time.UTC), Severity: "Moderate", CVSSv3: stackrox_rhel_csafValue20}

var stackrox_rhel_csafValue22 = csaf.CVSS{Score: 5.9, Vector: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:N/A:H"}

var stackrox_rhel_csafValue23 = csaf.Advisory{Name: "RHSA-2021:1131", Description: "Red Hat Security Advisory: openssl security update", ReleaseDate: time.Date(2021, 4, 7, 15, 34, 17, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue22}

var stackrox_rhel_csafValue24 = csaf.CVSS{Score: 7.8, Vector: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H"}

var stackrox_rhel_csafValue25 = csaf.Advisory{Name: "RHSA-2023:5453", Description: "Red Hat Security Advisory: glibc security update", ReleaseDate: time.Date(2023, 10, 5, 14, 3, 40, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue24}

var stackrox_rhel_csafValue26 = csaf.CVSS{Score: 9.9, Vector: "CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:C/C:H/I:H/A:H"}

var stackrox_rhel_csafValue27 = csaf.Advisory{Name: "RHSA-2023:3198", Description: "Red Hat Security Advisory: jenkins and jenkins-2-plugins security update", ReleaseDate: time.Date(2023, 5, 17, 17, 53, 4, 0, time.UTC), Severity: "Critical", CVSSv3: stackrox_rhel_csafValue26}

var stackrox_rhel_csafValue28 = csaf.Advisory{Name: "RHSA-2026:7661", Description: "Red Hat Security Advisory: Red Hat Hardened Images RPMs bug fix and enhancement update", ReleaseDate: time.Date(2026, 4, 11, 19, 41, 59, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue0}

var stackrox_rhel_csafValue29 = csaf.CVSS{Score: 8.8, Vector: "CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H"}

var stackrox_rhel_csafValue30 = csaf.Advisory{Name: "RHSA-2026:36207", Description: "Red Hat Security Advisory: RHACS 4.11.1 security and bug fix update", ReleaseDate: time.Date(2026, 7, 7, 11, 9, 24, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue29}

var stackrox_rhel_csafValue31 = csaf.Advisory{Name: "RHSA-2026:1519", Description: "Red Hat Security Advisory: openssl security update", ReleaseDate: time.Date(2026, 1, 29, 0, 24, 19, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue13}

var stackrox_rhel_csafValue32 = csaf.Advisory{Name: "RHSA-2021:0741", Description: "Red Hat Security Advisory: nodejs:10 security update", ReleaseDate: time.Date(2021, 3, 8, 10, 23, 18, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue0}

var stackrox_rhel_csafValue33 = csaf.Advisory{Name: "RHSA-2026:26546", Description: "Red Hat Security Advisory: RHACS 4.9.8 security and bug fix update", ReleaseDate: time.Date(2026, 6, 17, 10, 25, 30, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue29}

var stackrox_rhel_csafValue34 = csaf.Advisory{Name: "RHSA-2024:0776", Description: "Red Hat Security Advisory: jenkins and jenkins-2-plugins security update", ReleaseDate: time.Date(2024, 2, 12, 10, 26, 48, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue13}

var stackrox_rhel_csafValue35 = csaf.CVSS{Score: 5.9, Vector: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:N/A:N"}

var stackrox_rhel_csafValue36 = csaf.Advisory{Name: "RHSA-2022:5311", Description: "Red Hat Security Advisory: libgcrypt security update", ReleaseDate: time.Date(2022, 6, 30, 21, 5, 49, 0, time.UTC), Severity: "Moderate", CVSSv3: stackrox_rhel_csafValue35}

var stackrox_rhel_csafValue37 = csaf.CVSS{Score: 8.1, Vector: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:H"}

var stackrox_rhel_csafValue38 = csaf.Advisory{Name: "RHBA-2022:8582", Description: "Red Hat Bug Fix Advisory: OpenShift Container Platform 4.9.52 packages update", ReleaseDate: time.Date(2022, 11, 23, 17, 59, 1, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue37}

var stackrox_rhel_csafValue39 = csaf.CVSS{Score: 9.8, Vector: "CVSS:3.0/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H"}

var stackrox_rhel_csafValue40 = csaf.CVSS{Score: 7.5, Vector: "AV:N/AC:L/Au:N/C:P/I:P/A:P"}

var stackrox_rhel_csafValue41 = csaf.Advisory{Name: "RHSA-2019:0910", Description: "Red Hat Security Advisory: Red Hat Fuse 7.3 security update", ReleaseDate: time.Date(2019, 4, 30, 15, 18, 16, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue39, CVSSv2: stackrox_rhel_csafValue40}

var stackrox_rhel_csafValue42 = csaf.CVSS{Score: 3.7, Vector: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:L/A:N"}

var stackrox_rhel_csafValue43 = csaf.Advisory{Name: "RHSA-2023:7540", Description: "Red Hat Security Advisory: curl security and bug fix update", ReleaseDate: time.Date(2023, 11, 28, 15, 39, 2, 0, time.UTC), Severity: "Low", CVSSv3: stackrox_rhel_csafValue42}

var stackrox_rhel_csafValue44 = csaf.Advisory{Name: "RHSA-2022:6224", Description: "Red Hat Security Advisory: openssl security and bug fix update", ReleaseDate: time.Date(2022, 8, 30, 16, 7, 21, 0, time.UTC), Severity: "Moderate", CVSSv3: stackrox_rhel_csafValue0}

var stackrox_rhel_csafValue45 = csaf.Advisory{Name: "RHSA-2024:3635", Description: "Red Hat Security Advisory: Red Hat Product OCP Tools 4.12 Openshift Jenkins security update", ReleaseDate: time.Date(2024, 6, 5, 14, 47, 22, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue29}

var stackrox_rhel_csafValue46 = csaf.Advisory{Name: "RHSA-2020:4464", Description: "Red Hat Security Advisory: libxslt security update", ReleaseDate: time.Date(2020, 11, 4, 1, 47, 26, 0, time.UTC), Severity: "Moderate", CVSSv3: stackrox_rhel_csafValue16}

var stackrox_rhel_csafValue47 = csaf.Advisory{Name: "RHSA-2020:4272", Description: "Red Hat Security Advisory: nodejs:12 security and bug fix update", ReleaseDate: time.Date(2020, 10, 19, 14, 37, 38, 0, time.UTC), Severity: "Moderate", CVSSv3: stackrox_rhel_csafValue24}

var stackrox_rhel_csafValue48 = csaf.CVSS{Score: 5.5, Vector: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:N/I:N/A:H"}

var stackrox_rhel_csafValue49 = csaf.Advisory{Name: "RHSA-2021:2724", Description: "Red Hat Security Advisory: systemd security update", ReleaseDate: time.Date(2021, 7, 20, 22, 33, 56, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue48}

var stackrox_rhel_csafValue50 = csaf.Advisory{Name: "RHSA-2026:37275", Description: "Red Hat Security Advisory: RHOAI 3.3.5 - Red Hat OpenShift AI", ReleaseDate: time.Date(2026, 7, 9, 11, 57, 58, 0, time.UTC), Severity: "Critical", CVSSv3: stackrox_rhel_csafValue18}

var stackrox_rhel_csafValue51 = csaf.Advisory{Name: "RHSA-2020:4907", Description: "Red Hat Security Advisory: freetype security update", ReleaseDate: time.Date(2020, 11, 4, 14, 38, 53, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue5}

var stackrox_rhel_csafValue52 = csaf.Advisory{Name: "RHSA-2026:1594", Description: "Red Hat Security Advisory: openssl security update", ReleaseDate: time.Date(2026, 1, 29, 17, 22, 14, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue13}

var stackrox_rhel_csafValue53 = csaf.Advisory{Name: "RHSA-2026:40945", Description: "Red Hat Security Advisory: Red Hat Edge Manager Version 1.1.3 Security Update", ReleaseDate: time.Date(2026, 7, 16, 10, 53, 15, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue18}

var stackrox_rhel_csafValue54 = csaf.CVSS{Score: 8.7, Vector: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:C/C:H/I:H/A:N"}

var stackrox_rhel_csafValue55 = csaf.Advisory{Name: "RHSA-2026:30651", Description: "Red Hat Security Advisory: Red Hat Advanced Cluster Management for Kubernetes v2.13.9 security update", ReleaseDate: time.Date(2026, 6, 28, 15, 14, 48, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue54}

var stackrox_rhel_csafValue56 = csaf.Advisory{Name: "RHSA-2026:41036", Description: "Red Hat Security Advisory: Red Hat OpenShift Builds 1.8.1", ReleaseDate: time.Date(2026, 7, 16, 15, 12, 39, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue29}

var stackrox_rhel_csafValue57 = csaf.Advisory{Name: "RHSA-2026:36808", Description: "Red Hat Security Advisory: DevWorkspace Operator 0.42.0 release.", ReleaseDate: time.Date(2026, 7, 8, 16, 6, 37, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue29}

var stackrox_rhel_csafValue58 = csaf.CVSS{Score: 5.3, Vector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:N/A:N"}

var stackrox_rhel_csafValue59 = csaf.Advisory{Name: "RHSA-2024:1601", Description: "Red Hat Security Advisory: curl security and bug fix update", ReleaseDate: time.Date(2024, 4, 2, 16, 2, 18, 0, time.UTC), Severity: "Moderate", CVSSv3: stackrox_rhel_csafValue58}

var stackrox_rhel_csafValue60 = csaf.CVSS{Score: 7, Vector: "CVSS:3.1/AV:L/AC:H/PR:L/UI:N/S:U/C:H/I:H/A:H"}

var stackrox_rhel_csafValue61 = csaf.Advisory{Name: "RHSA-2020:1792", Description: "Red Hat Security Advisory: curl security update", ReleaseDate: time.Date(2020, 4, 28, 15, 45, 59, 0, time.UTC), Severity: "Moderate", CVSSv3: stackrox_rhel_csafValue60}

var stackrox_rhel_csafValue62 = csaf.Advisory{Name: "RHSA-2024:0778", Description: "Red Hat Security Advisory: Jenkins and Jenkins-2-plugins security update", ReleaseDate: time.Date(2024, 2, 12, 10, 38, 58, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue13}

var stackrox_rhel_csafValue63 = csaf.Advisory{Name: "RHSA-2021:2721", Description: "Red Hat Security Advisory: systemd security update", ReleaseDate: time.Date(2021, 7, 20, 22, 40, 45, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue48}

var stackrox_rhel_csafValue64 = csaf.Advisory{Name: "RHSA-2021:1063", Description: "Red Hat Security Advisory: openssl security update", ReleaseDate: time.Date(2021, 4, 5, 13, 48, 26, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue22}

var stackrox_rhel_csafValue65 = csaf.CVSS{Score: 9.4, Vector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:L"}

var stackrox_rhel_csafValue66 = csaf.Advisory{Name: "RHSA-2022:5498", Description: "Red Hat Security Advisory: Satellite 6.11 Release", ReleaseDate: time.Date(2022, 7, 5, 14, 41, 16, 0, time.UTC), Severity: "Moderate", CVSSv3: stackrox_rhel_csafValue65}

var stackrox_rhel_csafValue67 = csaf.Advisory{Name: "RHSA-2026:23264", Description: "Red Hat Security Advisory: Red Hat Hardened Images RPMs bug fix and enhancement update", ReleaseDate: time.Date(2026, 6, 4, 12, 43, 59, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue7}

var stackrox_rhel_csafValue68 = csaf.Advisory{Name: "RHSA-2021:2717", Description: "Red Hat Security Advisory: systemd security update", ReleaseDate: time.Date(2021, 7, 21, 0, 41, 41, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue48}

var stackrox_rhel_csafValue69 = csaf.Advisory{Name: "RHSA-2026:40118", Description: "Red Hat Security Advisory: Red Hat Edge Manager Version 1.1.3 Security Update", ReleaseDate: time.Date(2026, 7, 15, 16, 22, 53, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue18}

var stackrox_rhel_csafValue70 = csaf.Advisory{Name: "RHSA-2026:36648", Description: "Red Hat Security Advisory: Red Hat OpenShift Builds 1.7.4", ReleaseDate: time.Date(2026, 7, 8, 9, 20, 18, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue29}

var stackrox_rhel_csafValue71 = csaf.Advisory{Name: "RHSA-2021:0548", Description: "Red Hat Security Advisory: nodejs:10 security update", ReleaseDate: time.Date(2021, 2, 16, 14, 25, 46, 0, time.UTC), Severity: "Moderate", CVSSv3: stackrox_rhel_csafValue37}

var stackrox_rhel_csafValue72 = csaf.Advisory{Name: "RHSA-2020:4312", Description: "Red Hat Security Advisory: rh-maven35-jackson-databind security update", ReleaseDate: time.Date(2020, 10, 22, 16, 48, 27, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue9}

var stackrox_rhel_csafValue73 = csaf.Advisory{Name: "RHSA-2026:30650", Description: "Red Hat Security Advisory: multicluster engine for Kubernetes v2.8.8 security update", ReleaseDate: time.Date(2026, 6, 28, 14, 38, 37, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue54}

var stackrox_rhel_csafValue74 = csaf.Advisory{Name: "RHSA-2021:0739", Description: "Red Hat Security Advisory: nodejs:12 security update", ReleaseDate: time.Date(2021, 3, 8, 10, 18, 18, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue0}

var stackrox_rhel_csafValue75 = csaf.Advisory{Name: "RHBA-2022:5747", Description: "Red Hat Bug Fix Advisory: .NET 6.0 bugfix update", ReleaseDate: time.Date(2022, 7, 28, 10, 19, 12, 0, time.UTC), Severity: "Moderate", CVSSv3: stackrox_rhel_csafValue20}

var stackrox_rhel_csafValue76 = csaf.Advisory{Name: "RHSA-2026:33531", Description: "Red Hat Security Advisory: Red Hat Enterprise Linux AI 3.4.1 enhancement update", ReleaseDate: time.Date(2026, 6, 30, 14, 5, 56, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue7}

var stackrox_rhel_csafValue77 = csaf.Advisory{Name: "RHSA-2026:37387", Description: "Red Hat Security Advisory: Red Hat OpenShift Data Foundation 4.22.0 security, enhancement & bug fix update", ReleaseDate: time.Date(2026, 7, 9, 14, 48, 40, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue29}

var stackrox_rhel_csafValue78 = csaf.Advisory{Name: "RHSA-2024:3634", Description: "Red Hat Security Advisory: Red Hat Product OCP Tools 4.14 OpenShift Jenkins security update", ReleaseDate: time.Date(2024, 6, 5, 14, 47, 2, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue29}

var stackrox_rhel_csafValue79 = csaf.Advisory{Name: "RHSA-2026:1473", Description: "Red Hat Security Advisory: openssl security update", ReleaseDate: time.Date(2026, 1, 28, 10, 8, 56, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue13}

var stackrox_rhel_csafValue80 = csaf.Advisory{Name: "RHSA-2022:6531", Description: "Red Hat Security Advisory: OpenShift Container Platform 4.10.33 packages and security update", ReleaseDate: time.Date(2022, 9, 21, 14, 3, 24, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue9}

var stackrox_rhel_csafValue81 = csaf.Advisory{Name: "RHSA-2024:4597", Description: "Red Hat Security Advisory: Red Hat Product OCP Tools 4.15 OpenShift Jenkins security update", ReleaseDate: time.Date(2024, 7, 17, 18, 49, 17, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue29}

var stackrox_rhel_csafValue82 = csaf.CVSS{Score: 7.4, Vector: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:N"}

var stackrox_rhel_csafValue83 = csaf.Advisory{Name: "RHSA-2021:1024", Description: "Red Hat Security Advisory: openssl security update", ReleaseDate: time.Date(2021, 3, 30, 14, 40, 51, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue82}

var stackrox_rhel_csafValue84 = csaf.Advisory{Name: "RHSA-2020:4508", Description: "Red Hat Security Advisory: libsolv security, bug fix, and enhancement update", ReleaseDate: time.Date(2020, 11, 4, 2, 15, 19, 0, time.UTC), Severity: "Moderate", CVSSv3: stackrox_rhel_csafValue0}

var stackrox_rhel_csafValue85 = csaf.Advisory{Name: "RHSA-2021:4060", Description: "Red Hat Security Advisory: libsolv security update", ReleaseDate: time.Date(2021, 11, 2, 9, 9, 39, 0, time.UTC), Severity: "Moderate", CVSSv3: stackrox_rhel_csafValue0}

var stackrox_rhel_csafValue86 = csaf.Advisory{Name: "RHSA-2019:0483", Description: "Red Hat Security Advisory: openssl security and bug fix update", ReleaseDate: time.Date(2019, 3, 13, 13, 0, 21, 0, time.UTC), Severity: "Moderate", CVSSv3: stackrox_rhel_csafValue11}

var stackrox_rhel_csafValue87 = csaf.Advisory{Name: "RHSA-2023:6745", Description: "Red Hat Security Advisory: curl security update", ReleaseDate: time.Date(2023, 11, 7, 10, 27, 3, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue37}

var stackrox_rhel_csafValue88 = csaf.Advisory{Name: "RHSA-2023:5455", Description: "Red Hat Security Advisory: glibc security update", ReleaseDate: time.Date(2023, 10, 5, 14, 14, 13, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue24}

var stackrox_rhel_csafValue89 = csaf.Advisory{Name: "RHSA-2021:0735", Description: "Red Hat Security Advisory: nodejs:10 security update", ReleaseDate: time.Date(2021, 3, 4, 16, 8, 40, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue0}

var stackrox_rhel_csafValue90 = csaf.Advisory{Name: "RHSA-2022:7288", Description: "Red Hat Security Advisory: openssl security update", ReleaseDate: time.Date(2022, 11, 1, 18, 40, 16, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue0}

var stackrox_rhel_csafValue91 = csaf.Advisory{Name: "RHSA-2026:36796", Description: "Red Hat Security Advisory: Red Hat Edge Manager Version 1.0.3 Security Update", ReleaseDate: time.Date(2026, 7, 8, 15, 52, 29, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue18}

var stackrox_rhel_csafValue92 = csaf.Advisory{Name: "RHSA-2026:1733", Description: "Red Hat Security Advisory: openssl security update", ReleaseDate: time.Date(2026, 2, 2, 17, 33, 59, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue13}

var stackrox_rhel_csafValue93 = csaf.CVSS{Score: 7.8, Vector: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H"}

var stackrox_rhel_csafValue94 = csaf.Advisory{Name: "RHSA-2022:5813", Description: "Red Hat Security Advisory: vim security update", ReleaseDate: time.Date(2022, 8, 3, 13, 51, 5, 0, time.UTC), Severity: "Moderate", CVSSv3: stackrox_rhel_csafValue93}

var stackrox_rhel_csafValue95 = csaf.Advisory{Name: "RHSA-2020:1020", Description: "Red Hat Security Advisory: curl security and bug fix update", ReleaseDate: time.Date(2020, 3, 31, 20, 48, 11, 0, time.UTC), Severity: "Low", CVSSv3: stackrox_rhel_csafValue60}

var stackrox_rhel_csafValue96 = csaf.Advisory{Name: "RHSA-2020:4949", Description: "Red Hat Security Advisory: freetype security update", ReleaseDate: time.Date(2020, 11, 5, 8, 40, 36, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue5}

var stackrox_rhel_csafValue97 = csaf.CVSS{Score: 5.9, Vector: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:H/A:N"}

var stackrox_rhel_csafValue98 = csaf.Advisory{Name: "RHSA-2024:3203", Description: "Red Hat Security Advisory: systemd security update", ReleaseDate: time.Date(2024, 5, 22, 10, 4, 25, 0, time.UTC), Severity: "Moderate", CVSSv3: stackrox_rhel_csafValue97}

var stackrox_rhel_csafValue99 = csaf.Advisory{Name: "RHSA-2020:4173", Description: "Red Hat Security Advisory: rh-maven35-jackson-databind security update", ReleaseDate: time.Date(2020, 10, 5, 15, 14, 20, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue37}

var stackrox_rhel_csafValue100 = csaf.Advisory{Name: "RHSA-2024:3636", Description: "Red Hat Security Advisory: Red Hat Product OCP Tools 4.13 OpenShift Jenkins security update", ReleaseDate: time.Date(2024, 6, 5, 14, 46, 12, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue29}

var stackrox_rhel_csafValue101 = csaf.Advisory{Name: "RHSA-2022:6057", Description: "Red Hat Security Advisory: .NET Core 3.1 security, bug fix, and enhancement update", ReleaseDate: time.Date(2022, 8, 15, 9, 4, 46, 0, time.UTC), Severity: "Moderate", CVSSv3: stackrox_rhel_csafValue20}

var stackrox_rhel_csafValue102 = csaf.Advisory{Name: "RHSA-2023:6292", Description: "Red Hat Security Advisory: curl security update", ReleaseDate: time.Date(2023, 11, 2, 16, 9, 3, 0, time.UTC), Severity: "Moderate", CVSSv3: stackrox_rhel_csafValue97}

var stackrox_rhel_csafValue103 = csaf.Advisory{Name: "RHSA-2026:41019", Description: "Red Hat Security Advisory: Red Hat Edge Manager Version 1.1.3 Security Update", ReleaseDate: time.Date(2026, 7, 16, 14, 35, 59, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue18}

var stackrox_rhel_csafValue104 = csaf.Advisory{Name: "RHSA-2021:0740", Description: "Red Hat Security Advisory: nodejs:12 security update", ReleaseDate: time.Date(2021, 3, 8, 10, 31, 43, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue0}

var stackrox_rhel_csafValue105 = csaf.CVSS{Score: 3.3, Vector: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:N/I:N/A:L"}

var stackrox_rhel_csafValue106 = csaf.Advisory{Name: "RHSA-2021:4408", Description: "Red Hat Security Advisory: libsolv security and bug fix update", ReleaseDate: time.Date(2021, 11, 9, 18, 19, 51, 0, time.UTC), Severity: "Low", CVSSv3: stackrox_rhel_csafValue105}

var stackrox_rhel_csafValue107 = csaf.Advisory{Name: "RHSA-2021:0744", Description: "Red Hat Security Advisory: nodejs:14 security and bug fix update", ReleaseDate: time.Date(2021, 3, 8, 10, 36, 40, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue0}

var stackrox_rhel_csafValue108 = csaf.Advisory{Name: "RHSA-2026:36797", Description: "Red Hat Security Advisory: RHTAS 1.4.2 - CLI Stack Release", ReleaseDate: time.Date(2026, 7, 8, 15, 46, 32, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue29}

var stackrox_rhel_csafValue109 = csaf.Advisory{Name: "RHSA-2022:5942", Description: "Red Hat Security Advisory: vim security update", ReleaseDate: time.Date(2022, 8, 9, 10, 32, 11, 0, time.UTC), Severity: "Moderate", CVSSv3: stackrox_rhel_csafValue93}

var stackrox_rhel_csafValue110 = csaf.CVSS{Score: 8.8, Vector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H"}

var stackrox_rhel_csafValue111 = csaf.Advisory{Name: "RHSA-2026:36820", Description: "Red Hat Security Advisory: Red Hat OpenShift Dev Spaces 3.29.0 Release.", ReleaseDate: time.Date(2026, 7, 8, 17, 12, 17, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue110}

var stackrox_rhel_csafValue112 = csaf.Advisory{Name: "RHSA-2021:0551", Description: "Red Hat Security Advisory: nodejs:14 security and bug fix update", ReleaseDate: time.Date(2021, 2, 16, 14, 28, 3, 0, time.UTC), Severity: "Moderate", CVSSv3: stackrox_rhel_csafValue37}

var stackrox_rhel_csafValue113 = csaf.Advisory{Name: "RHSA-2021:2575", Description: "Red Hat Security Advisory: lz4 security update", ReleaseDate: time.Date(2021, 6, 29, 16, 36, 36, 0, time.UTC), Severity: "Moderate", CVSSv3: stackrox_rhel_csafValue5}

var stackrox_rhel_csafValue114 = csaf.Advisory{Name: "RHSA-2023:3299", Description: "Red Hat Security Advisory: jenkins and jenkins-2-plugins security update", ReleaseDate: time.Date(2023, 5, 24, 17, 13, 53, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue13}

var stackrox_rhel_csafValue115 = csaf.Advisory{Name: "RHSA-2020:2505", Description: "Red Hat Security Advisory: curl security update", ReleaseDate: time.Date(2020, 6, 12, 5, 40, 14, 0, time.UTC), Severity: "Low", CVSSv3: stackrox_rhel_csafValue60}

var stackrox_rhel_csafValue116 = csaf.Advisory{Name: "RHSA-2020:4952", Description: "Red Hat Security Advisory: freetype security update", ReleaseDate: time.Date(2020, 11, 5, 8, 53, 38, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue5}

var stackrox_rhel_csafValue117 = csaf.Advisory{Name: "RHSA-2026:1503", Description: "Red Hat Security Advisory: openssl security update", ReleaseDate: time.Date(2026, 1, 28, 17, 17, 47, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue13}

var stackrox_rhel_csafValue118 = csaf.Advisory{Name: "RHSA-2024:2463", Description: "Red Hat Security Advisory: systemd security update", ReleaseDate: time.Date(2024, 4, 30, 10, 4, 59, 0, time.UTC), Severity: "Moderate", CVSSv3: stackrox_rhel_csafValue97}

var stackrox_rhel_csafValue119 = csaf.Advisory{Name: "RHSA-2020:4951", Description: "Red Hat Security Advisory: freetype security update", ReleaseDate: time.Date(2020, 11, 5, 9, 0, 7, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue5}

var stackrox_rhel_csafValue120 = csaf.CVSS{Score: 6.7, Vector: "CVSS:3.1/AV:L/AC:L/PR:H/UI:N/S:U/C:H/I:H/A:H"}

var stackrox_rhel_csafValue121 = csaf.Advisory{Name: "RHSA-2022:5818", Description: "Red Hat Security Advisory: openssl security update", ReleaseDate: time.Date(2022, 8, 3, 12, 50, 24, 0, time.UTC), Severity: "Moderate", CVSSv3: stackrox_rhel_csafValue120}

var stackrox_rhel_csafValue122 = csaf.Advisory{Name: "RHSA-2021:0738", Description: "Red Hat Security Advisory: nodejs:10 security update", ReleaseDate: time.Date(2021, 3, 8, 10, 27, 55, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue0}

var stackrox_rhel_csafValue123 = csaf.Advisory{Name: "RHSA-2023:0017", Description: "Red Hat Security Advisory: OpenShift Container Platform 4.8.56 packages and security update", ReleaseDate: time.Date(2023, 1, 12, 16, 49, 54, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue110}

var stackrox_rhel_csafValue124 = csaf.Advisory{Name: "RHSA-2023:5454", Description: "Red Hat Security Advisory: glibc security update", ReleaseDate: time.Date(2023, 10, 5, 13, 11, 35, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue24}

var stackrox_rhel_csafValue125 = csaf.Advisory{Name: "RHSA-2020:4903", Description: "Red Hat Security Advisory: nodejs:12 security and bug fix update", ReleaseDate: time.Date(2020, 11, 4, 12, 35, 47, 0, time.UTC), Severity: "Moderate", CVSSv3: stackrox_rhel_csafValue24}

var stackrox_rhel_csafValue126 = csaf.Advisory{Name: "RHSA-2023:5476", Description: "Red Hat Security Advisory: glibc security update", ReleaseDate: time.Date(2023, 10, 5, 15, 41, 31, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue24}

var stackrox_rhel_csafValue127 = csaf.Advisory{Name: "RHSA-2026:26547", Description: "Red Hat Security Advisory: RHACS 4.10.4 security and bug fix update", ReleaseDate: time.Date(2026, 6, 17, 10, 26, 32, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue29}

var stackrox_rhel_csafValue128 = csaf.Advisory{Name: "RHSA-2023:5763", Description: "Red Hat Security Advisory: curl security update", ReleaseDate: time.Date(2023, 10, 17, 9, 4, 53, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue37}

var stackrox_rhel_csafValue129 = csaf.Advisory{Name: "RHSA-2026:33524", Description: "Red Hat Security Advisory: Red Hat Enterprise Linux AI 3.4.1 enhancement update", ReleaseDate: time.Date(2026, 6, 30, 13, 53, 11, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue7}

var stackrox_rhel_csafValue130 = csaf.Advisory{Name: "RHSA-2021:0549", Description: "Red Hat Security Advisory: nodejs:12 security update", ReleaseDate: time.Date(2021, 2, 16, 14, 25, 52, 0, time.UTC), Severity: "Moderate", CVSSv3: stackrox_rhel_csafValue37}

var stackrox_rhel_csafValue131 = csaf.Advisory{Name: "RHSA-2023:5700", Description: "Red Hat Security Advisory: curl security update", ReleaseDate: time.Date(2023, 10, 13, 21, 51, 56, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue37}

var stackrox_rhel_csafValue132 = csaf.Advisory{Name: "RHSA-2026:1472", Description: "Red Hat Security Advisory: openssl security update", ReleaseDate: time.Date(2026, 1, 28, 9, 6, 6, 0, time.UTC), Severity: "Important", CVSSv3: stackrox_rhel_csafValue13}

var stackrox_rhel_csafValue133 = csaf.Advisory{Name: "RHSA-2022:0246", Description: "Red Hat Security Advisory: nodejs:14 security, bug fix, and enhancement update", ReleaseDate: time.Date(2022, 1, 25, 9, 28, 51, 0, time.UTC), Severity: "Moderate", CVSSv3: stackrox_rhel_csafValue13}
