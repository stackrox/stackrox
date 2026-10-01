package main

import (
	unique "unique"

	claircore "github.com/quay/claircore"
	cpe "github.com/quay/claircore/toolkit/types/cpe"
)

// rhel-vex.json.zst native fixture relationships, transcribed from the checked-in CI bundle.
// Kept independent from expected results in scanner/e2etests and backend QA.
// Scenarios: TestImage sandbox-jenkins-agent-maven-35-rhel7[-chown],
// sandbox-nodejs-10, ubi9/ubi:9.0.0-1576, sandbox-dotnet-60-runtime,
// spring-CVE-2022-22978, ose-jenkins, and scanner:4.3.0. Backend QA also uses
// ubi9-minimal-9.6-1760515502 (openssl-libs) and ubi9-9.7-1769417801 (python3).
// Keep the native repository CPE/module/architecture constraints. Shared
// identities below are also used by fixtures_rhel_unaffected.go.
var rhel_vexFixtures = []operation{
	{Member: "rhel-vex.json.zst", Updater: "rhel-vex", Vulnerabilities: []*claircore.Vulnerability{
		{Updater: "rhel-vex", Name: "CVE-2022-30065", Description: rhel_vexValue0, Links: rhel_vexValue1, Severity: "CVSS:3.1/AV:L/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue2, Repo: &rhel_vexValue9, Self: rhel_vexValue10, Aliases: []claircore.Alias{rhel_vexValue11}},
		{Updater: "rhel-vex", Name: "CVE-2020-15366", Description: rhel_vexValue12, Links: rhel_vexValue13, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:L/I:L/A:L", NormalizedSeverity: 3, Package: &rhel_vexValue14, Repo: &rhel_vexValue20, FixedInVersion: "0:12.19.1-1.module+el8.3.0+8851+b7b41ca0", Self: rhel_vexValue21, Aliases: []claircore.Alias{rhel_vexValue22}},
		{Updater: "rhel-vex", Name: "CVE-2020-15366", Description: rhel_vexValue12, Links: rhel_vexValue23, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:L/I:L/A:L", NormalizedSeverity: 3, Package: &rhel_vexValue14, Repo: &rhel_vexValue20, FixedInVersion: "0:10.23.1-1.module+el8.3.0+9502+012d8a97", Self: rhel_vexValue21, Aliases: []claircore.Alias{rhel_vexValue22}},
		{Updater: "rhel-vex", Name: "CVE-2020-15366", Description: rhel_vexValue12, Links: rhel_vexValue24, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:L/I:L/A:L", NormalizedSeverity: 3, Package: &rhel_vexValue14, Repo: &rhel_vexValue20, FixedInVersion: "0:14.15.4-2.module+el8.3.0+9635+ffdf8381", Self: rhel_vexValue21, Aliases: []claircore.Alias{rhel_vexValue22}},
		{Updater: "rhel-vex", Name: "CVE-2020-15366", Description: rhel_vexValue12, Links: rhel_vexValue25, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:L/I:L/A:L", NormalizedSeverity: 3, Package: &rhel_vexValue26, Repo: &rhel_vexValue28, Self: rhel_vexValue21, Aliases: []claircore.Alias{rhel_vexValue22}},
		{Updater: "rhel-vex", Name: "CVE-2020-15366", Description: rhel_vexValue12, Links: rhel_vexValue25, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:L/I:L/A:L", NormalizedSeverity: 3, Package: &rhel_vexValue29, Repo: &rhel_vexValue28, Self: rhel_vexValue21, Aliases: []claircore.Alias{rhel_vexValue22}},
		{Updater: "rhel-vex", Name: "CVE-2020-15366", Description: rhel_vexValue12, Links: rhel_vexValue25, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:L/I:L/A:L", NormalizedSeverity: 3, Package: &rhel_vexValue30, Repo: &rhel_vexValue28, Self: rhel_vexValue21, Aliases: []claircore.Alias{rhel_vexValue22}},
		{Updater: "rhel-vex", Name: "CVE-2021-33928", Description: rhel_vexValue31, Links: rhel_vexValue32, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue33, Repo: &rhel_vexValue38, FixedInVersion: "0:0.7.16-3.el8_4", Self: rhel_vexValue39, Aliases: []claircore.Alias{rhel_vexValue40}},
		{Updater: "rhel-vex", Name: "CVE-2021-33928", Description: rhel_vexValue31, Links: rhel_vexValue32, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue33, Repo: &rhel_vexValue43, Self: rhel_vexValue39, Aliases: []claircore.Alias{rhel_vexValue40}},
		{Updater: "rhel-vex", Name: "CVE-2021-33928", Description: rhel_vexValue31, Links: rhel_vexValue32, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue33, Repo: &rhel_vexValue45, FixedInVersion: "0:0.7.16-3.el8_4", Self: rhel_vexValue39, Aliases: []claircore.Alias{rhel_vexValue40}},
		{Updater: "rhel-vex", Name: "CVE-2021-33928", Description: rhel_vexValue31, Links: rhel_vexValue32, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue33, Repo: &rhel_vexValue43, FixedInVersion: "0:0.7.16-3.el8_4", Self: rhel_vexValue39, Aliases: []claircore.Alias{rhel_vexValue40}},
		{Updater: "rhel-vex", Name: "CVE-2021-33928", Description: rhel_vexValue31, Links: rhel_vexValue46, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue33, Repo: &rhel_vexValue51, FixedInVersion: "0:0.7.22-1.el7pc", Self: rhel_vexValue39, Aliases: []claircore.Alias{rhel_vexValue40}},
		{Updater: "rhel-vex", Name: "CVE-2021-33928", Description: rhel_vexValue31, Links: rhel_vexValue46, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue33, Repo: &rhel_vexValue54, FixedInVersion: "0:0.7.22-1.el8pc", Self: rhel_vexValue39, Aliases: []claircore.Alias{rhel_vexValue40}},
		{Updater: "rhel-vex", Name: "CVE-2021-33928", Description: rhel_vexValue31, Links: rhel_vexValue46, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue33, Repo: &rhel_vexValue57, FixedInVersion: "0:0.7.22-1.el7pc", Self: rhel_vexValue39, Aliases: []claircore.Alias{rhel_vexValue40}},
		{Updater: "rhel-vex", Name: "CVE-2021-33928", Description: rhel_vexValue31, Links: rhel_vexValue46, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue33, Repo: &rhel_vexValue59, FixedInVersion: "0:0.7.22-1.el8pc", Self: rhel_vexValue39, Aliases: []claircore.Alias{rhel_vexValue40}},
		{Updater: "rhel-vex", Name: "CVE-2021-33928", Description: rhel_vexValue31, Links: rhel_vexValue60, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue61, Repo: &rhel_vexValue64, Self: rhel_vexValue39, Aliases: []claircore.Alias{rhel_vexValue40}},
		{Updater: "rhel-vex", Name: "CVE-2021-33928", Description: rhel_vexValue31, Links: rhel_vexValue60, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue61, Repo: &rhel_vexValue28, Self: rhel_vexValue39, Aliases: []claircore.Alias{rhel_vexValue40}},
		{Updater: "rhel-vex", Name: "CVE-2021-33928", Description: rhel_vexValue31, Links: rhel_vexValue60, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue61, Repo: &rhel_vexValue66, Self: rhel_vexValue39, Aliases: []claircore.Alias{rhel_vexValue40}},
		{Updater: "rhel-vex", Name: "CVE-2021-33928", Description: rhel_vexValue31, Links: rhel_vexValue60, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue61, Repo: &rhel_vexValue70, Self: rhel_vexValue39, Aliases: []claircore.Alias{rhel_vexValue40}},
		{Updater: "rhel-vex", Name: "CVE-2019-20387", Description: rhel_vexValue71, Links: rhel_vexValue72, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue33, Repo: &rhel_vexValue45, FixedInVersion: "0:0.7.11-1.el8", Self: rhel_vexValue73, Aliases: []claircore.Alias{rhel_vexValue74}},
		{Updater: "rhel-vex", Name: "CVE-2019-20387", Description: rhel_vexValue71, Links: rhel_vexValue75, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue61, Repo: &rhel_vexValue64, Self: rhel_vexValue73, Aliases: []claircore.Alias{rhel_vexValue74}},
		{Updater: "rhel-vex", Name: "CVE-2019-20387", Description: rhel_vexValue71, Links: rhel_vexValue75, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue61, Repo: &rhel_vexValue28, Self: rhel_vexValue73, Aliases: []claircore.Alias{rhel_vexValue74}},
		{Updater: "rhel-vex", Name: "CVE-2019-20387", Description: rhel_vexValue71, Links: rhel_vexValue75, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue61, Repo: &rhel_vexValue66, Self: rhel_vexValue73, Aliases: []claircore.Alias{rhel_vexValue74}},
		{Updater: "rhel-vex", Name: "CVE-2019-20387", Description: rhel_vexValue71, Links: rhel_vexValue75, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue61, Repo: &rhel_vexValue70, Self: rhel_vexValue73, Aliases: []claircore.Alias{rhel_vexValue74}},
		{Updater: "rhel-vex", Name: "CVE-2020-14060", Description: rhel_vexValue76, Links: rhel_vexValue77, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 2, Package: &rhel_vexValue78, Repo: &rhel_vexValue81, Self: rhel_vexValue82, Aliases: []claircore.Alias{rhel_vexValue83}},
		{Updater: "rhel-vex", Name: "CVE-2018-0735", Description: rhel_vexValue84, Links: rhel_vexValue85, Severity: "CVSS:3.0/AV:L/AC:H/PR:N/UI:N/S:U/C:H/I:N/A:N", NormalizedSeverity: 2, Package: &rhel_vexValue86, Repo: &rhel_vexValue89, FixedInVersion: "1:1.0.2k-16.el7_6.1", Self: rhel_vexValue90, Aliases: []claircore.Alias{rhel_vexValue91}},
		{Updater: "rhel-vex", Name: "CVE-2018-0735", Description: rhel_vexValue84, Links: rhel_vexValue85, Severity: "CVSS:3.0/AV:L/AC:H/PR:N/UI:N/S:U/C:H/I:N/A:N", NormalizedSeverity: 2, Package: &rhel_vexValue92, Repo: &rhel_vexValue89, FixedInVersion: "1:1.0.2k-16.el7_6.1", Self: rhel_vexValue90, Aliases: []claircore.Alias{rhel_vexValue91}},
		{Updater: "rhel-vex", Name: "CVE-2018-0735", Description: rhel_vexValue84, Links: rhel_vexValue85, Severity: "CVSS:3.0/AV:L/AC:H/PR:N/UI:N/S:U/C:H/I:N/A:N", NormalizedSeverity: 2, Package: &rhel_vexValue86, Repo: &rhel_vexValue95, FixedInVersion: "1:1.0.2k-16.el7_6.1", Self: rhel_vexValue90, Aliases: []claircore.Alias{rhel_vexValue91}},
		{Updater: "rhel-vex", Name: "CVE-2018-0735", Description: rhel_vexValue84, Links: rhel_vexValue85, Severity: "CVSS:3.0/AV:L/AC:H/PR:N/UI:N/S:U/C:H/I:N/A:N", NormalizedSeverity: 2, Package: &rhel_vexValue92, Repo: &rhel_vexValue95, FixedInVersion: "1:1.0.2k-16.el7_6.1", Self: rhel_vexValue90, Aliases: []claircore.Alias{rhel_vexValue91}},
		{Updater: "rhel-vex", Name: "CVE-2018-0735", Description: rhel_vexValue84, Links: rhel_vexValue85, Severity: "CVSS:3.0/AV:L/AC:H/PR:N/UI:N/S:U/C:H/I:N/A:N", NormalizedSeverity: 2, Package: &rhel_vexValue86, Repo: &rhel_vexValue98, FixedInVersion: "1:1.0.2k-16.el7_6.1", Self: rhel_vexValue90, Aliases: []claircore.Alias{rhel_vexValue91}},
		{Updater: "rhel-vex", Name: "CVE-2018-0735", Description: rhel_vexValue84, Links: rhel_vexValue85, Severity: "CVSS:3.0/AV:L/AC:H/PR:N/UI:N/S:U/C:H/I:N/A:N", NormalizedSeverity: 2, Package: &rhel_vexValue92, Repo: &rhel_vexValue98, FixedInVersion: "1:1.0.2k-16.el7_6.1", Self: rhel_vexValue90, Aliases: []claircore.Alias{rhel_vexValue91}},
		{Updater: "rhel-vex", Name: "CVE-2018-0735", Description: rhel_vexValue84, Links: rhel_vexValue85, Severity: "CVSS:3.0/AV:L/AC:H/PR:N/UI:N/S:U/C:H/I:N/A:N", NormalizedSeverity: 2, Package: &rhel_vexValue86, Repo: &rhel_vexValue101, FixedInVersion: "1:1.0.2k-16.el7_6.1", Self: rhel_vexValue90, Aliases: []claircore.Alias{rhel_vexValue91}},
		{Updater: "rhel-vex", Name: "CVE-2018-0735", Description: rhel_vexValue84, Links: rhel_vexValue85, Severity: "CVSS:3.0/AV:L/AC:H/PR:N/UI:N/S:U/C:H/I:N/A:N", NormalizedSeverity: 2, Package: &rhel_vexValue92, Repo: &rhel_vexValue101, FixedInVersion: "1:1.0.2k-16.el7_6.1", Self: rhel_vexValue90, Aliases: []claircore.Alias{rhel_vexValue91}},
		{Updater: "rhel-vex", Name: "CVE-2018-0735", Description: rhel_vexValue84, Links: rhel_vexValue102, Severity: "CVSS:3.0/AV:L/AC:H/PR:N/UI:N/S:U/C:H/I:N/A:N", NormalizedSeverity: 2, Package: &rhel_vexValue86, Repo: &rhel_vexValue45, FixedInVersion: "1:1.1.1c-2.el8", Self: rhel_vexValue90, Aliases: []claircore.Alias{rhel_vexValue91}},
		{Updater: "rhel-vex", Name: "CVE-2018-0735", Description: rhel_vexValue84, Links: rhel_vexValue102, Severity: "CVSS:3.0/AV:L/AC:H/PR:N/UI:N/S:U/C:H/I:N/A:N", NormalizedSeverity: 2, Package: &rhel_vexValue92, Repo: &rhel_vexValue45, FixedInVersion: "1:1.1.1c-2.el8", Self: rhel_vexValue90, Aliases: []claircore.Alias{rhel_vexValue91}},
		{Updater: "rhel-vex", Name: "CVE-2018-0735", Description: rhel_vexValue84, Links: rhel_vexValue103, Severity: "CVSS:3.0/AV:L/AC:H/PR:N/UI:N/S:U/C:H/I:N/A:N", NormalizedSeverity: 2, Package: &rhel_vexValue104, Repo: &rhel_vexValue9, Self: rhel_vexValue90, Aliases: []claircore.Alias{rhel_vexValue91}},
		{Updater: "rhel-vex", Name: "CVE-2018-0735", Description: rhel_vexValue84, Links: rhel_vexValue103, Severity: "CVSS:3.0/AV:L/AC:H/PR:N/UI:N/S:U/C:H/I:N/A:N", NormalizedSeverity: 2, Package: &rhel_vexValue104, Repo: &rhel_vexValue64, Self: rhel_vexValue90, Aliases: []claircore.Alias{rhel_vexValue91}},
		{Updater: "rhel-vex", Name: "CVE-2018-0735", Description: rhel_vexValue84, Links: rhel_vexValue103, Severity: "CVSS:3.0/AV:L/AC:H/PR:N/UI:N/S:U/C:H/I:N/A:N", NormalizedSeverity: 2, Package: &rhel_vexValue105, Repo: &rhel_vexValue64, Self: rhel_vexValue90, Aliases: []claircore.Alias{rhel_vexValue91}},
		{Updater: "rhel-vex", Name: "CVE-2018-0735", Description: rhel_vexValue84, Links: rhel_vexValue103, Severity: "CVSS:3.0/AV:L/AC:H/PR:N/UI:N/S:U/C:H/I:N/A:N", NormalizedSeverity: 2, Package: &rhel_vexValue104, Repo: &rhel_vexValue28, Self: rhel_vexValue90, Aliases: []claircore.Alias{rhel_vexValue91}},
		{Updater: "rhel-vex", Name: "CVE-2021-33930", Description: rhel_vexValue106, Links: rhel_vexValue107, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue33, Repo: &rhel_vexValue38, FixedInVersion: "0:0.7.16-3.el8_4", Self: rhel_vexValue108, Aliases: []claircore.Alias{rhel_vexValue109}},
		{Updater: "rhel-vex", Name: "CVE-2021-33930", Description: rhel_vexValue106, Links: rhel_vexValue107, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue33, Repo: &rhel_vexValue43, Self: rhel_vexValue108, Aliases: []claircore.Alias{rhel_vexValue109}},
		{Updater: "rhel-vex", Name: "CVE-2021-33930", Description: rhel_vexValue106, Links: rhel_vexValue107, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue33, Repo: &rhel_vexValue45, FixedInVersion: "0:0.7.16-3.el8_4", Self: rhel_vexValue108, Aliases: []claircore.Alias{rhel_vexValue109}},
		{Updater: "rhel-vex", Name: "CVE-2021-33930", Description: rhel_vexValue106, Links: rhel_vexValue107, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue33, Repo: &rhel_vexValue43, FixedInVersion: "0:0.7.16-3.el8_4", Self: rhel_vexValue108, Aliases: []claircore.Alias{rhel_vexValue109}},
		{Updater: "rhel-vex", Name: "CVE-2021-33930", Description: rhel_vexValue106, Links: rhel_vexValue110, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue33, Repo: &rhel_vexValue51, FixedInVersion: "0:0.7.22-1.el7pc", Self: rhel_vexValue108, Aliases: []claircore.Alias{rhel_vexValue109}},
		{Updater: "rhel-vex", Name: "CVE-2021-33930", Description: rhel_vexValue106, Links: rhel_vexValue110, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue33, Repo: &rhel_vexValue54, FixedInVersion: "0:0.7.22-1.el8pc", Self: rhel_vexValue108, Aliases: []claircore.Alias{rhel_vexValue109}},
		{Updater: "rhel-vex", Name: "CVE-2021-33930", Description: rhel_vexValue106, Links: rhel_vexValue110, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue33, Repo: &rhel_vexValue57, FixedInVersion: "0:0.7.22-1.el7pc", Self: rhel_vexValue108, Aliases: []claircore.Alias{rhel_vexValue109}},
		{Updater: "rhel-vex", Name: "CVE-2021-33930", Description: rhel_vexValue106, Links: rhel_vexValue110, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue33, Repo: &rhel_vexValue59, FixedInVersion: "0:0.7.22-1.el8pc", Self: rhel_vexValue108, Aliases: []claircore.Alias{rhel_vexValue109}},
		{Updater: "rhel-vex", Name: "CVE-2021-33930", Description: rhel_vexValue106, Links: rhel_vexValue111, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue61, Repo: &rhel_vexValue64, Self: rhel_vexValue108, Aliases: []claircore.Alias{rhel_vexValue109}},
		{Updater: "rhel-vex", Name: "CVE-2021-33930", Description: rhel_vexValue106, Links: rhel_vexValue111, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue61, Repo: &rhel_vexValue28, Self: rhel_vexValue108, Aliases: []claircore.Alias{rhel_vexValue109}},
		{Updater: "rhel-vex", Name: "CVE-2021-33930", Description: rhel_vexValue106, Links: rhel_vexValue111, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue61, Repo: &rhel_vexValue66, Self: rhel_vexValue108, Aliases: []claircore.Alias{rhel_vexValue109}},
		{Updater: "rhel-vex", Name: "CVE-2021-33930", Description: rhel_vexValue106, Links: rhel_vexValue111, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue61, Repo: &rhel_vexValue70, Self: rhel_vexValue108, Aliases: []claircore.Alias{rhel_vexValue109}},
		{Updater: "rhel-vex", Name: "CVE-2021-22883", Description: rhel_vexValue112, Links: rhel_vexValue113, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue14, Repo: &rhel_vexValue20, FixedInVersion: "0:12.21.0-1.module+el8.3.0+10191+34fb5a07", Self: rhel_vexValue114, Aliases: []claircore.Alias{rhel_vexValue115}},
		{Updater: "rhel-vex", Name: "CVE-2021-22883", Description: rhel_vexValue112, Links: rhel_vexValue116, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue14, Repo: &rhel_vexValue20, FixedInVersion: "0:10.24.0-1.module+el8.3.0+10166+b07ac28e", Self: rhel_vexValue114, Aliases: []claircore.Alias{rhel_vexValue115}},
		{Updater: "rhel-vex", Name: "CVE-2021-22883", Description: rhel_vexValue112, Links: rhel_vexValue117, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue14, Repo: &rhel_vexValue120, FixedInVersion: "0:10.24.0-1.module+el8.2.0+10165+019e8570", Self: rhel_vexValue114, Aliases: []claircore.Alias{rhel_vexValue115}},
		{Updater: "rhel-vex", Name: "CVE-2021-22883", Description: rhel_vexValue112, Links: rhel_vexValue121, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue14, Repo: &rhel_vexValue124, FixedInVersion: "0:12.21.0-1.module+el8.1.0+10194+d5e49c90", Self: rhel_vexValue114, Aliases: []claircore.Alias{rhel_vexValue115}},
		{Updater: "rhel-vex", Name: "CVE-2021-22883", Description: rhel_vexValue112, Links: rhel_vexValue125, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue14, Repo: &rhel_vexValue120, FixedInVersion: "0:12.21.0-1.module+el8.2.0+10192+8959c43b", Self: rhel_vexValue114, Aliases: []claircore.Alias{rhel_vexValue115}},
		{Updater: "rhel-vex", Name: "CVE-2021-22883", Description: rhel_vexValue112, Links: rhel_vexValue126, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue14, Repo: &rhel_vexValue124, FixedInVersion: "0:10.24.0-1.module+el8.1.0+10161+5cffdac6", Self: rhel_vexValue114, Aliases: []claircore.Alias{rhel_vexValue115}},
		{Updater: "rhel-vex", Name: "CVE-2021-22883", Description: rhel_vexValue112, Links: rhel_vexValue127, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue14, Repo: &rhel_vexValue20, FixedInVersion: "0:14.16.0-2.module+el8.3.0+10180+b92e1eb6", Self: rhel_vexValue114, Aliases: []claircore.Alias{rhel_vexValue115}},
		{Updater: "rhel-vex", Name: "CVE-2021-22883", Description: rhel_vexValue112, Links: rhel_vexValue128, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue26, Repo: &rhel_vexValue28, Self: rhel_vexValue114, Aliases: []claircore.Alias{rhel_vexValue115}},
		{Updater: "rhel-vex", Name: "CVE-2021-22883", Description: rhel_vexValue112, Links: rhel_vexValue128, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue29, Repo: &rhel_vexValue28, Self: rhel_vexValue114, Aliases: []claircore.Alias{rhel_vexValue115}},
		{Updater: "rhel-vex", Name: "CVE-2021-22883", Description: rhel_vexValue112, Links: rhel_vexValue128, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue26, Repo: &rhel_vexValue131, Self: rhel_vexValue114, Aliases: []claircore.Alias{rhel_vexValue115}},
		{Updater: "rhel-vex", Name: "CVE-2021-22883", Description: rhel_vexValue112, Links: rhel_vexValue128, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue29, Repo: &rhel_vexValue131, Self: rhel_vexValue114, Aliases: []claircore.Alias{rhel_vexValue115}},
		{Updater: "rhel-vex", Name: "CVE-2021-22883", Description: rhel_vexValue112, Links: rhel_vexValue128, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue30, Repo: &rhel_vexValue28, Self: rhel_vexValue114, Aliases: []claircore.Alias{rhel_vexValue115}},
		{Updater: "rhel-vex", Name: "CVE-2021-22883", Description: rhel_vexValue112, Links: rhel_vexValue128, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue132, Repo: &rhel_vexValue135, Self: rhel_vexValue114, Aliases: []claircore.Alias{rhel_vexValue115}},
		{Updater: "rhel-vex", Name: "CVE-2019-5436", Description: rhel_vexValue136, Links: rhel_vexValue137, Severity: "CVSS:3.1/AV:L/AC:H/PR:L/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 2, Package: &rhel_vexValue138, Repo: &rhel_vexValue101, FixedInVersion: "0:7.29.0-57.el7", Self: rhel_vexValue139, Aliases: []claircore.Alias{rhel_vexValue140}},
		{Updater: "rhel-vex", Name: "CVE-2019-5436", Description: rhel_vexValue136, Links: rhel_vexValue137, Severity: "CVSS:3.1/AV:L/AC:H/PR:L/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 2, Package: &rhel_vexValue138, Repo: &rhel_vexValue95, FixedInVersion: "0:7.29.0-57.el7", Self: rhel_vexValue139, Aliases: []claircore.Alias{rhel_vexValue140}},
		{Updater: "rhel-vex", Name: "CVE-2019-5436", Description: rhel_vexValue136, Links: rhel_vexValue137, Severity: "CVSS:3.1/AV:L/AC:H/PR:L/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 2, Package: &rhel_vexValue138, Repo: &rhel_vexValue98, FixedInVersion: "0:7.29.0-57.el7", Self: rhel_vexValue139, Aliases: []claircore.Alias{rhel_vexValue140}},
		{Updater: "rhel-vex", Name: "CVE-2019-5436", Description: rhel_vexValue136, Links: rhel_vexValue137, Severity: "CVSS:3.1/AV:L/AC:H/PR:L/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 2, Package: &rhel_vexValue138, Repo: &rhel_vexValue89, FixedInVersion: "0:7.29.0-57.el7", Self: rhel_vexValue139, Aliases: []claircore.Alias{rhel_vexValue140}},
		{Updater: "rhel-vex", Name: "CVE-2019-5436", Description: rhel_vexValue136, Links: rhel_vexValue141, Severity: "CVSS:3.1/AV:L/AC:H/PR:L/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 2, Package: &rhel_vexValue138, Repo: &rhel_vexValue45, FixedInVersion: "0:7.61.1-12.el8", Self: rhel_vexValue139, Aliases: []claircore.Alias{rhel_vexValue140}},
		{Updater: "rhel-vex", Name: "CVE-2019-5436", Description: rhel_vexValue136, Links: rhel_vexValue142, Severity: "CVSS:3.1/AV:L/AC:H/PR:L/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 2, Package: &rhel_vexValue138, Repo: &rhel_vexValue145, FixedInVersion: "0:7.29.0-54.el7_7.3", Self: rhel_vexValue139, Aliases: []claircore.Alias{rhel_vexValue140}},
		{Updater: "rhel-vex", Name: "CVE-2019-5436", Description: rhel_vexValue136, Links: rhel_vexValue142, Severity: "CVSS:3.1/AV:L/AC:H/PR:L/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 2, Package: &rhel_vexValue138, Repo: &rhel_vexValue147, FixedInVersion: "0:7.29.0-54.el7_7.3", Self: rhel_vexValue139, Aliases: []claircore.Alias{rhel_vexValue140}},
		{Updater: "rhel-vex", Name: "CVE-2019-5436", Description: rhel_vexValue136, Links: rhel_vexValue148, Severity: "CVSS:3.1/AV:L/AC:H/PR:L/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 2, Package: &rhel_vexValue149, Repo: &rhel_vexValue9, Self: rhel_vexValue139, Aliases: []claircore.Alias{rhel_vexValue140}},
		{Updater: "rhel-vex", Name: "CVE-2019-5436", Description: rhel_vexValue136, Links: rhel_vexValue148, Severity: "CVSS:3.1/AV:L/AC:H/PR:L/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 2, Package: &rhel_vexValue149, Repo: &rhel_vexValue28, Self: rhel_vexValue139, Aliases: []claircore.Alias{rhel_vexValue140}},
		{Updater: "rhel-vex", Name: "CVE-2020-7608", Description: rhel_vexValue150, Links: rhel_vexValue151, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:L/I:L/A:L", NormalizedSeverity: 2, Package: &rhel_vexValue14, Repo: &rhel_vexValue20, FixedInVersion: "0:12.19.1-1.module+el8.3.0+8851+b7b41ca0", Self: rhel_vexValue152, Aliases: []claircore.Alias{rhel_vexValue153}},
		{Updater: "rhel-vex", Name: "CVE-2020-7608", Description: rhel_vexValue150, Links: rhel_vexValue154, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:L/I:L/A:L", NormalizedSeverity: 2, Package: &rhel_vexValue14, Repo: &rhel_vexValue20, FixedInVersion: "0:10.23.1-1.module+el8.3.0+9502+012d8a97", Self: rhel_vexValue152, Aliases: []claircore.Alias{rhel_vexValue153}},
		{Updater: "rhel-vex", Name: "CVE-2020-7608", Description: rhel_vexValue150, Links: rhel_vexValue155, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:L/I:L/A:L", NormalizedSeverity: 2, Package: &rhel_vexValue26, Repo: &rhel_vexValue28, Self: rhel_vexValue152, Aliases: []claircore.Alias{rhel_vexValue153}},
		{Updater: "rhel-vex", Name: "CVE-2020-7608", Description: rhel_vexValue150, Links: rhel_vexValue155, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:L/I:L/A:L", NormalizedSeverity: 2, Package: &rhel_vexValue29, Repo: &rhel_vexValue28, Self: rhel_vexValue152, Aliases: []claircore.Alias{rhel_vexValue153}},
		{Updater: "rhel-vex", Name: "CVE-2020-15999", Description: rhel_vexValue156, Links: rhel_vexValue157, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue158, Repo: &rhel_vexValue95, FixedInVersion: "0:2.8-14.el7_9.1", Self: rhel_vexValue159, Aliases: []claircore.Alias{rhel_vexValue160}},
		{Updater: "rhel-vex", Name: "CVE-2020-15999", Description: rhel_vexValue156, Links: rhel_vexValue157, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue158, Repo: &rhel_vexValue98, FixedInVersion: "0:2.8-14.el7_9.1", Self: rhel_vexValue159, Aliases: []claircore.Alias{rhel_vexValue160}},
		{Updater: "rhel-vex", Name: "CVE-2020-15999", Description: rhel_vexValue156, Links: rhel_vexValue157, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue158, Repo: &rhel_vexValue89, FixedInVersion: "0:2.8-14.el7_9.1", Self: rhel_vexValue159, Aliases: []claircore.Alias{rhel_vexValue160}},
		{Updater: "rhel-vex", Name: "CVE-2020-15999", Description: rhel_vexValue156, Links: rhel_vexValue157, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue158, Repo: &rhel_vexValue101, FixedInVersion: "0:2.8-14.el7_9.1", Self: rhel_vexValue159, Aliases: []claircore.Alias{rhel_vexValue160}},
		{Updater: "rhel-vex", Name: "CVE-2020-15999", Description: rhel_vexValue156, Links: rhel_vexValue161, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue158, Repo: &rhel_vexValue165, FixedInVersion: "0:2.9.1-4.el8_0.1", Self: rhel_vexValue159, Aliases: []claircore.Alias{rhel_vexValue160}},
		{Updater: "rhel-vex", Name: "CVE-2020-15999", Description: rhel_vexValue156, Links: rhel_vexValue166, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue158, Repo: &rhel_vexValue168, FixedInVersion: "0:2.9.1-4.el8_1.1", Self: rhel_vexValue159, Aliases: []claircore.Alias{rhel_vexValue160}},
		{Updater: "rhel-vex", Name: "CVE-2020-15999", Description: rhel_vexValue156, Links: rhel_vexValue169, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue158, Repo: &rhel_vexValue171, FixedInVersion: "0:2.9.1-4.el8_2.1", Self: rhel_vexValue159, Aliases: []claircore.Alias{rhel_vexValue160}},
		{Updater: "rhel-vex", Name: "CVE-2020-15999", Description: rhel_vexValue156, Links: rhel_vexValue172, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue158, Repo: &rhel_vexValue45, FixedInVersion: "0:2.9.1-4.el8_3.1", Self: rhel_vexValue159, Aliases: []claircore.Alias{rhel_vexValue160}},
		{Updater: "rhel-vex", Name: "CVE-2020-15999", Description: rhel_vexValue156, Links: rhel_vexValue173, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue174, Repo: &rhel_vexValue64, Self: rhel_vexValue159, Aliases: []claircore.Alias{rhel_vexValue160}},
		{Updater: "rhel-vex", Name: "CVE-2020-15999", Description: rhel_vexValue156, Links: rhel_vexValue173, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue174, Repo: &rhel_vexValue28, Self: rhel_vexValue159, Aliases: []claircore.Alias{rhel_vexValue160}},
		{Updater: "rhel-vex", Name: "CVE-2018-5407", Description: rhel_vexValue175, Links: rhel_vexValue176, Severity: "CVSS:3.0/AV:P/AC:H/PR:L/UI:N/S:C/C:H/I:N/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue86, Repo: &rhel_vexValue95, FixedInVersion: "1:1.0.2k-16.el7_6.1", Self: rhel_vexValue177, Aliases: []claircore.Alias{rhel_vexValue178}},
		{Updater: "rhel-vex", Name: "CVE-2018-5407", Description: rhel_vexValue175, Links: rhel_vexValue176, Severity: "CVSS:3.0/AV:P/AC:H/PR:L/UI:N/S:C/C:H/I:N/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue92, Repo: &rhel_vexValue95, FixedInVersion: "1:1.0.2k-16.el7_6.1", Self: rhel_vexValue177, Aliases: []claircore.Alias{rhel_vexValue178}},
		{Updater: "rhel-vex", Name: "CVE-2018-5407", Description: rhel_vexValue175, Links: rhel_vexValue176, Severity: "CVSS:3.0/AV:P/AC:H/PR:L/UI:N/S:C/C:H/I:N/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue86, Repo: &rhel_vexValue98, FixedInVersion: "1:1.0.2k-16.el7_6.1", Self: rhel_vexValue177, Aliases: []claircore.Alias{rhel_vexValue178}},
		{Updater: "rhel-vex", Name: "CVE-2018-5407", Description: rhel_vexValue175, Links: rhel_vexValue176, Severity: "CVSS:3.0/AV:P/AC:H/PR:L/UI:N/S:C/C:H/I:N/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue92, Repo: &rhel_vexValue98, FixedInVersion: "1:1.0.2k-16.el7_6.1", Self: rhel_vexValue177, Aliases: []claircore.Alias{rhel_vexValue178}},
		{Updater: "rhel-vex", Name: "CVE-2018-5407", Description: rhel_vexValue175, Links: rhel_vexValue176, Severity: "CVSS:3.0/AV:P/AC:H/PR:L/UI:N/S:C/C:H/I:N/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue86, Repo: &rhel_vexValue89, FixedInVersion: "1:1.0.2k-16.el7_6.1", Self: rhel_vexValue177, Aliases: []claircore.Alias{rhel_vexValue178}},
		{Updater: "rhel-vex", Name: "CVE-2018-5407", Description: rhel_vexValue175, Links: rhel_vexValue176, Severity: "CVSS:3.0/AV:P/AC:H/PR:L/UI:N/S:C/C:H/I:N/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue86, Repo: &rhel_vexValue101, FixedInVersion: "1:1.0.2k-16.el7_6.1", Self: rhel_vexValue177, Aliases: []claircore.Alias{rhel_vexValue178}},
		{Updater: "rhel-vex", Name: "CVE-2018-5407", Description: rhel_vexValue175, Links: rhel_vexValue176, Severity: "CVSS:3.0/AV:P/AC:H/PR:L/UI:N/S:C/C:H/I:N/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue92, Repo: &rhel_vexValue101, FixedInVersion: "1:1.0.2k-16.el7_6.1", Self: rhel_vexValue177, Aliases: []claircore.Alias{rhel_vexValue178}},
		{Updater: "rhel-vex", Name: "CVE-2018-5407", Description: rhel_vexValue175, Links: rhel_vexValue179, Severity: "CVSS:3.0/AV:P/AC:H/PR:L/UI:N/S:C/C:H/I:N/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue104, Repo: &rhel_vexValue182, Self: rhel_vexValue177, Aliases: []claircore.Alias{rhel_vexValue178}},
		{Updater: "rhel-vex", Name: "CVE-2018-5407", Description: rhel_vexValue175, Links: rhel_vexValue179, Severity: "CVSS:3.0/AV:P/AC:H/PR:L/UI:N/S:C/C:H/I:N/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue104, Repo: &rhel_vexValue9, Self: rhel_vexValue177, Aliases: []claircore.Alias{rhel_vexValue178}},
		{Updater: "rhel-vex", Name: "CVE-2020-24750", Description: rhel_vexValue183, Links: rhel_vexValue184, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue185, Repo: &rhel_vexValue187, FixedInVersion: "0:2.7.6-2.11.el7", Self: rhel_vexValue188, Aliases: []claircore.Alias{rhel_vexValue189}},
		{Updater: "rhel-vex", Name: "CVE-2020-24750", Description: rhel_vexValue183, Links: rhel_vexValue190, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue78, Repo: &rhel_vexValue81, Self: rhel_vexValue188, Aliases: []claircore.Alias{rhel_vexValue189}},
		{Updater: "rhel-vex", Name: "CVE-2020-24616", Description: rhel_vexValue191, Links: rhel_vexValue192, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue78, Repo: &rhel_vexValue81, Self: rhel_vexValue193, Aliases: []claircore.Alias{rhel_vexValue194}},
		{Updater: "rhel-vex", Name: "CVE-2020-14061", Description: rhel_vexValue195, Links: rhel_vexValue196, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 2, Package: &rhel_vexValue78, Repo: &rhel_vexValue81, Self: rhel_vexValue197, Aliases: []claircore.Alias{rhel_vexValue198}},
		{Updater: "rhel-vex", Name: "CVE-2021-3200", Description: rhel_vexValue199, Links: rhel_vexValue200, Severity: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:N/I:N/A:L", NormalizedSeverity: 2, Package: &rhel_vexValue33, Repo: &rhel_vexValue45, FixedInVersion: "0:0.7.19-1.el8", Self: rhel_vexValue201, Aliases: []claircore.Alias{rhel_vexValue202}},
		{Updater: "rhel-vex", Name: "CVE-2021-3200", Description: rhel_vexValue199, Links: rhel_vexValue200, Severity: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:N/I:N/A:L", NormalizedSeverity: 2, Package: &rhel_vexValue33, Repo: &rhel_vexValue43, FixedInVersion: "0:0.7.19-1.el8", Self: rhel_vexValue201, Aliases: []claircore.Alias{rhel_vexValue202}},
		{Updater: "rhel-vex", Name: "CVE-2021-3200", Description: rhel_vexValue199, Links: rhel_vexValue203, Severity: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:N/I:N/A:L", NormalizedSeverity: 2, Package: &rhel_vexValue33, Repo: &rhel_vexValue51, FixedInVersion: "0:0.7.22-1.el7pc", Self: rhel_vexValue201, Aliases: []claircore.Alias{rhel_vexValue202}},
		{Updater: "rhel-vex", Name: "CVE-2021-3200", Description: rhel_vexValue199, Links: rhel_vexValue203, Severity: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:N/I:N/A:L", NormalizedSeverity: 2, Package: &rhel_vexValue33, Repo: &rhel_vexValue54, FixedInVersion: "0:0.7.22-1.el8pc", Self: rhel_vexValue201, Aliases: []claircore.Alias{rhel_vexValue202}},
		{Updater: "rhel-vex", Name: "CVE-2021-3200", Description: rhel_vexValue199, Links: rhel_vexValue203, Severity: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:N/I:N/A:L", NormalizedSeverity: 2, Package: &rhel_vexValue33, Repo: &rhel_vexValue57, FixedInVersion: "0:0.7.22-1.el7pc", Self: rhel_vexValue201, Aliases: []claircore.Alias{rhel_vexValue202}},
		{Updater: "rhel-vex", Name: "CVE-2021-3200", Description: rhel_vexValue199, Links: rhel_vexValue203, Severity: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:N/I:N/A:L", NormalizedSeverity: 2, Package: &rhel_vexValue33, Repo: &rhel_vexValue59, FixedInVersion: "0:0.7.22-1.el8pc", Self: rhel_vexValue201, Aliases: []claircore.Alias{rhel_vexValue202}},
		{Updater: "rhel-vex", Name: "CVE-2021-3200", Description: rhel_vexValue199, Links: rhel_vexValue204, Severity: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:N/I:N/A:L", NormalizedSeverity: 2, Package: &rhel_vexValue61, Repo: &rhel_vexValue207, Self: rhel_vexValue201, Aliases: []claircore.Alias{rhel_vexValue202}},
		{Updater: "rhel-vex", Name: "CVE-2021-3200", Description: rhel_vexValue199, Links: rhel_vexValue204, Severity: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:N/I:N/A:L", NormalizedSeverity: 2, Package: &rhel_vexValue61, Repo: &rhel_vexValue64, Self: rhel_vexValue201, Aliases: []claircore.Alias{rhel_vexValue202}},
		{Updater: "rhel-vex", Name: "CVE-2021-3200", Description: rhel_vexValue199, Links: rhel_vexValue204, Severity: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:N/I:N/A:L", NormalizedSeverity: 2, Package: &rhel_vexValue61, Repo: &rhel_vexValue28, Self: rhel_vexValue201, Aliases: []claircore.Alias{rhel_vexValue202}},
		{Updater: "rhel-vex", Name: "CVE-2021-3200", Description: rhel_vexValue199, Links: rhel_vexValue204, Severity: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:N/I:N/A:L", NormalizedSeverity: 2, Package: &rhel_vexValue61, Repo: &rhel_vexValue66, Self: rhel_vexValue201, Aliases: []claircore.Alias{rhel_vexValue202}},
		{Updater: "rhel-vex", Name: "CVE-2021-3200", Description: rhel_vexValue199, Links: rhel_vexValue204, Severity: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:N/I:N/A:L", NormalizedSeverity: 2, Package: &rhel_vexValue61, Repo: &rhel_vexValue70, Self: rhel_vexValue201, Aliases: []claircore.Alias{rhel_vexValue202}},
		{Updater: "rhel-vex", Name: "CVE-2020-8265", Description: rhel_vexValue208, Links: rhel_vexValue209, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue14, Repo: &rhel_vexValue20, FixedInVersion: "0:10.23.1-1.module+el8.3.0+9502+012d8a97", Self: rhel_vexValue210, Aliases: []claircore.Alias{rhel_vexValue211}},
		{Updater: "rhel-vex", Name: "CVE-2020-8265", Description: rhel_vexValue208, Links: rhel_vexValue212, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue14, Repo: &rhel_vexValue20, FixedInVersion: "0:12.20.1-1.module+el8.3.0+9503+19cb079c", Self: rhel_vexValue210, Aliases: []claircore.Alias{rhel_vexValue211}},
		{Updater: "rhel-vex", Name: "CVE-2020-8265", Description: rhel_vexValue208, Links: rhel_vexValue213, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue14, Repo: &rhel_vexValue20, FixedInVersion: "0:14.15.4-2.module+el8.3.0+9635+ffdf8381", Self: rhel_vexValue210, Aliases: []claircore.Alias{rhel_vexValue211}},
		{Updater: "rhel-vex", Name: "CVE-2020-8265", Description: rhel_vexValue208, Links: rhel_vexValue214, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue26, Repo: &rhel_vexValue28, Self: rhel_vexValue210, Aliases: []claircore.Alias{rhel_vexValue211}},
		{Updater: "rhel-vex", Name: "CVE-2020-8265", Description: rhel_vexValue208, Links: rhel_vexValue214, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue29, Repo: &rhel_vexValue28, Self: rhel_vexValue210, Aliases: []claircore.Alias{rhel_vexValue211}},
		{Updater: "rhel-vex", Name: "CVE-2020-8265", Description: rhel_vexValue208, Links: rhel_vexValue214, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue30, Repo: &rhel_vexValue28, Self: rhel_vexValue210, Aliases: []claircore.Alias{rhel_vexValue211}},
		{Updater: "rhel-vex", Name: "CVE-2020-8265", Description: rhel_vexValue208, Links: rhel_vexValue214, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue132, Repo: &rhel_vexValue135, Self: rhel_vexValue210, Aliases: []claircore.Alias{rhel_vexValue211}},
		{Updater: "rhel-vex", Name: "CVE-2021-40528", Description: rhel_vexValue215, Links: rhel_vexValue216, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:N/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue217, Repo: &rhel_vexValue220, FixedInVersion: "0:1.8.5-7.el8_6", Self: rhel_vexValue221, Aliases: []claircore.Alias{rhel_vexValue222}},
		{Updater: "rhel-vex", Name: "CVE-2021-40528", Description: rhel_vexValue215, Links: rhel_vexValue216, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:N/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue217, Repo: &rhel_vexValue45, FixedInVersion: "0:1.8.5-7.el8_6", Self: rhel_vexValue221, Aliases: []claircore.Alias{rhel_vexValue222}},
		{Updater: "rhel-vex", Name: "CVE-2021-40528", Description: rhel_vexValue215, Links: rhel_vexValue223, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:N/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue224, Repo: &rhel_vexValue9, Self: rhel_vexValue221, Aliases: []claircore.Alias{rhel_vexValue222}},
		{Updater: "rhel-vex", Name: "CVE-2021-40528", Description: rhel_vexValue215, Links: rhel_vexValue223, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:N/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue224, Repo: &rhel_vexValue64, Self: rhel_vexValue221, Aliases: []claircore.Alias{rhel_vexValue222}},
		{Updater: "rhel-vex", Name: "CVE-2021-40528", Description: rhel_vexValue215, Links: rhel_vexValue223, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:N/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue224, Repo: &rhel_vexValue28, Self: rhel_vexValue221, Aliases: []claircore.Alias{rhel_vexValue222}},
		{Updater: "rhel-vex", Name: "CVE-2021-40528", Description: rhel_vexValue215, Links: rhel_vexValue223, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:N/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue224, Repo: &rhel_vexValue135, Self: rhel_vexValue221, Aliases: []claircore.Alias{rhel_vexValue222}},
		{Updater: "rhel-vex", Name: "CVE-2021-26291", Description: rhel_vexValue225, Links: rhel_vexValue226, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:N", NormalizedSeverity: 4, Package: &rhel_vexValue227, Repo: &rhel_vexValue231, Self: rhel_vexValue232, Aliases: []claircore.Alias{rhel_vexValue233}},
		{Updater: "rhel-vex", Name: "CVE-2021-26291", Description: rhel_vexValue225, Links: rhel_vexValue226, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:N", NormalizedSeverity: 4, Package: &rhel_vexValue234, Repo: &rhel_vexValue231, FixedInVersion: "0:4.11.1683009941-1.el8", Self: rhel_vexValue232, Aliases: []claircore.Alias{rhel_vexValue233}},
		{Updater: "rhel-vex", Name: "CVE-2021-26291", Description: rhel_vexValue225, Links: rhel_vexValue235, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:N", NormalizedSeverity: 4, Package: &rhel_vexValue227, Repo: &rhel_vexValue238, Self: rhel_vexValue232, Aliases: []claircore.Alias{rhel_vexValue233}},
		{Updater: "rhel-vex", Name: "CVE-2021-26291", Description: rhel_vexValue225, Links: rhel_vexValue235, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:N", NormalizedSeverity: 4, Package: &rhel_vexValue234, Repo: &rhel_vexValue238, FixedInVersion: "0:4.13.1706516346-1.el8", Self: rhel_vexValue232, Aliases: []claircore.Alias{rhel_vexValue233}},
		{Updater: "rhel-vex", Name: "CVE-2021-26291", Description: rhel_vexValue225, Links: rhel_vexValue239, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:N", NormalizedSeverity: 4, Package: &rhel_vexValue227, Repo: &rhel_vexValue242, Self: rhel_vexValue232, Aliases: []claircore.Alias{rhel_vexValue233}},
		{Updater: "rhel-vex", Name: "CVE-2021-26291", Description: rhel_vexValue225, Links: rhel_vexValue239, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:N", NormalizedSeverity: 4, Package: &rhel_vexValue234, Repo: &rhel_vexValue242, FixedInVersion: "0:4.12.1706515741-1.el8", Self: rhel_vexValue232, Aliases: []claircore.Alias{rhel_vexValue233}},
		{Updater: "rhel-vex", Name: "CVE-2021-26291", Description: rhel_vexValue225, Links: rhel_vexValue243, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue244, Repo: &rhel_vexValue246, Self: rhel_vexValue232, Aliases: []claircore.Alias{rhel_vexValue233}},
		{Updater: "rhel-vex", Name: "CVE-2021-26291", Description: rhel_vexValue225, Links: rhel_vexValue243, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue244, Repo: &rhel_vexValue248, Self: rhel_vexValue232, Aliases: []claircore.Alias{rhel_vexValue233}},
		{Updater: "rhel-vex", Name: "CVE-2021-26291", Description: rhel_vexValue225, Links: rhel_vexValue243, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue244, Repo: &rhel_vexValue250, Self: rhel_vexValue232, Aliases: []claircore.Alias{rhel_vexValue233}},
		{Updater: "rhel-vex", Name: "CVE-2021-26291", Description: rhel_vexValue225, Links: rhel_vexValue243, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue244, Repo: &rhel_vexValue254, Self: rhel_vexValue232, Aliases: []claircore.Alias{rhel_vexValue233}},
		{Updater: "rhel-vex", Name: "CVE-2021-26291", Description: rhel_vexValue225, Links: rhel_vexValue243, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue244, Repo: &rhel_vexValue256, Self: rhel_vexValue232, Aliases: []claircore.Alias{rhel_vexValue233}},
		{Updater: "rhel-vex", Name: "CVE-2021-26291", Description: rhel_vexValue225, Links: rhel_vexValue243, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue244, Repo: &rhel_vexValue258, Self: rhel_vexValue232, Aliases: []claircore.Alias{rhel_vexValue233}},
		{Updater: "rhel-vex", Name: "CVE-2021-26291", Description: rhel_vexValue225, Links: rhel_vexValue243, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue244, Repo: &rhel_vexValue261, Self: rhel_vexValue232, Aliases: []claircore.Alias{rhel_vexValue233}},
		{Updater: "rhel-vex", Name: "CVE-2021-3520", Description: rhel_vexValue262, Links: rhel_vexValue263, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue264, Repo: &rhel_vexValue38, FixedInVersion: "0:1.8.3-3.el8_4", Self: rhel_vexValue265, Aliases: []claircore.Alias{rhel_vexValue266}},
		{Updater: "rhel-vex", Name: "CVE-2021-3520", Description: rhel_vexValue262, Links: rhel_vexValue263, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue264, Repo: &rhel_vexValue45, FixedInVersion: "0:1.8.3-3.el8_4", Self: rhel_vexValue265, Aliases: []claircore.Alias{rhel_vexValue266}},
		{Updater: "rhel-vex", Name: "CVE-2021-3520", Description: rhel_vexValue262, Links: rhel_vexValue267, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue268, Repo: &rhel_vexValue64, Self: rhel_vexValue265, Aliases: []claircore.Alias{rhel_vexValue266}},
		{Updater: "rhel-vex", Name: "CVE-2021-3520", Description: rhel_vexValue262, Links: rhel_vexValue267, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue268, Repo: &rhel_vexValue28, Self: rhel_vexValue265, Aliases: []claircore.Alias{rhel_vexValue266}},
		{Updater: "rhel-vex", Name: "CVE-2021-3520", Description: rhel_vexValue262, Links: rhel_vexValue267, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue268, Repo: &rhel_vexValue135, Self: rhel_vexValue265, Aliases: []claircore.Alias{rhel_vexValue266}},
		{Updater: "rhel-vex", Name: "CVE-2023-38545", Description: rhel_vexValue269, Links: rhel_vexValue270, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue138, Repo: &rhel_vexValue273, FixedInVersion: "0:7.76.1-14.el9_0.9", Self: rhel_vexValue274, Aliases: []claircore.Alias{rhel_vexValue275}},
		{Updater: "rhel-vex", Name: "CVE-2023-38545", Description: rhel_vexValue269, Links: rhel_vexValue270, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue138, Repo: &rhel_vexValue277, FixedInVersion: "0:7.76.1-14.el9_0.9", Self: rhel_vexValue274, Aliases: []claircore.Alias{rhel_vexValue275}},
		{Updater: "rhel-vex", Name: "CVE-2023-38545", Description: rhel_vexValue269, Links: rhel_vexValue270, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue138, Repo: &rhel_vexValue273, Self: rhel_vexValue274, Aliases: []claircore.Alias{rhel_vexValue275}},
		{Updater: "rhel-vex", Name: "CVE-2023-38545", Description: rhel_vexValue269, Links: rhel_vexValue278, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue138, Repo: &rhel_vexValue281, Self: rhel_vexValue274, Aliases: []claircore.Alias{rhel_vexValue275}},
		{Updater: "rhel-vex", Name: "CVE-2023-38545", Description: rhel_vexValue269, Links: rhel_vexValue278, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue138, Repo: &rhel_vexValue283, FixedInVersion: "0:7.76.1-23.el9_2.4", Self: rhel_vexValue274, Aliases: []claircore.Alias{rhel_vexValue275}},
		{Updater: "rhel-vex", Name: "CVE-2023-38545", Description: rhel_vexValue269, Links: rhel_vexValue278, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue138, Repo: &rhel_vexValue285, FixedInVersion: "0:7.76.1-23.el9_2.4", Self: rhel_vexValue274, Aliases: []claircore.Alias{rhel_vexValue275}},
		{Updater: "rhel-vex", Name: "CVE-2023-38545", Description: rhel_vexValue269, Links: rhel_vexValue286, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue138, Repo: &rhel_vexValue288, Self: rhel_vexValue274, Aliases: []claircore.Alias{rhel_vexValue275}},
		{Updater: "rhel-vex", Name: "CVE-2023-38545", Description: rhel_vexValue269, Links: rhel_vexValue286, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue138, Repo: &rhel_vexValue285, FixedInVersion: "0:7.76.1-26.el9_3.2", Self: rhel_vexValue274, Aliases: []claircore.Alias{rhel_vexValue275}},
		{Updater: "rhel-vex", Name: "CVE-2023-38545", Description: rhel_vexValue269, Links: rhel_vexValue289, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue149, Repo: &rhel_vexValue135, Self: rhel_vexValue274, Aliases: []claircore.Alias{rhel_vexValue275}},
		{Updater: "rhel-vex", Name: "CVE-2020-25649", Description: rhel_vexValue290, Links: rhel_vexValue291, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:H/A:N", NormalizedSeverity: 4, Package: &rhel_vexValue185, Repo: &rhel_vexValue187, FixedInVersion: "0:2.7.6-2.12.el7", Self: rhel_vexValue292, Aliases: []claircore.Alias{rhel_vexValue293}},
		{Updater: "rhel-vex", Name: "CVE-2022-1927", Description: rhel_vexValue294, Links: rhel_vexValue295, Severity: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue296, Repo: &rhel_vexValue20, Self: rhel_vexValue297, Aliases: []claircore.Alias{rhel_vexValue298}},
		{Updater: "rhel-vex", Name: "CVE-2022-1927", Description: rhel_vexValue294, Links: rhel_vexValue295, Severity: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue299, Repo: &rhel_vexValue301, Self: rhel_vexValue297, Aliases: []claircore.Alias{rhel_vexValue298}},
		{Updater: "rhel-vex", Name: "CVE-2022-1927", Description: rhel_vexValue294, Links: rhel_vexValue295, Severity: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue296, Repo: &rhel_vexValue301, Self: rhel_vexValue297, Aliases: []claircore.Alias{rhel_vexValue298}},
		{Updater: "rhel-vex", Name: "CVE-2022-1927", Description: rhel_vexValue294, Links: rhel_vexValue295, Severity: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue299, Repo: &rhel_vexValue220, Self: rhel_vexValue297, Aliases: []claircore.Alias{rhel_vexValue298}},
		{Updater: "rhel-vex", Name: "CVE-2022-1927", Description: rhel_vexValue294, Links: rhel_vexValue295, Severity: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue296, Repo: &rhel_vexValue220, FixedInVersion: "2:8.0.1763-19.el8_6.4", Self: rhel_vexValue297, Aliases: []claircore.Alias{rhel_vexValue298}},
		{Updater: "rhel-vex", Name: "CVE-2022-1927", Description: rhel_vexValue294, Links: rhel_vexValue295, Severity: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue299, Repo: &rhel_vexValue20, Self: rhel_vexValue297, Aliases: []claircore.Alias{rhel_vexValue298}},
		{Updater: "rhel-vex", Name: "CVE-2022-1927", Description: rhel_vexValue294, Links: rhel_vexValue295, Severity: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue296, Repo: &rhel_vexValue45, FixedInVersion: "2:8.0.1763-19.el8_6.4", Self: rhel_vexValue297, Aliases: []claircore.Alias{rhel_vexValue298}},
		{Updater: "rhel-vex", Name: "CVE-2022-1927", Description: rhel_vexValue294, Links: rhel_vexValue302, Severity: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue296, Repo: &rhel_vexValue277, FixedInVersion: "2:8.2.2637-16.el9_0.3", Self: rhel_vexValue297, Aliases: []claircore.Alias{rhel_vexValue298}},
		{Updater: "rhel-vex", Name: "CVE-2022-1927", Description: rhel_vexValue294, Links: rhel_vexValue302, Severity: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue296, Repo: &rhel_vexValue273, Self: rhel_vexValue297, Aliases: []claircore.Alias{rhel_vexValue298}},
		{Updater: "rhel-vex", Name: "CVE-2022-1927", Description: rhel_vexValue294, Links: rhel_vexValue302, Severity: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue296, Repo: &rhel_vexValue285, FixedInVersion: "2:8.2.2637-16.el9_0.3", Self: rhel_vexValue297, Aliases: []claircore.Alias{rhel_vexValue298}},
		{Updater: "rhel-vex", Name: "CVE-2022-1927", Description: rhel_vexValue294, Links: rhel_vexValue303, Severity: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue304, Repo: &rhel_vexValue28, Self: rhel_vexValue297, Aliases: []claircore.Alias{rhel_vexValue298}},
		{Updater: "rhel-vex", Name: "CVE-2022-1927", Description: rhel_vexValue294, Links: rhel_vexValue303, Severity: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue305, Repo: &rhel_vexValue28, Self: rhel_vexValue297, Aliases: []claircore.Alias{rhel_vexValue298}},
		{Updater: "rhel-vex", Name: "CVE-2022-1927", Description: rhel_vexValue294, Links: rhel_vexValue303, Severity: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue304, Repo: &rhel_vexValue135, Self: rhel_vexValue297, Aliases: []claircore.Alias{rhel_vexValue298}},
		{Updater: "rhel-vex", Name: "CVE-2022-1927", Description: rhel_vexValue294, Links: rhel_vexValue303, Severity: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue305, Repo: &rhel_vexValue135, Self: rhel_vexValue297, Aliases: []claircore.Alias{rhel_vexValue298}},
		{Updater: "rhel-vex", Name: "CVE-2020-8252", Description: rhel_vexValue306, Links: rhel_vexValue307, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue14, Repo: &rhel_vexValue20, FixedInVersion: "0:12.18.4-2.module+el8.2.0+8361+192e434e", Self: rhel_vexValue308, Aliases: []claircore.Alias{rhel_vexValue309}},
		{Updater: "rhel-vex", Name: "CVE-2020-8252", Description: rhel_vexValue306, Links: rhel_vexValue310, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue14, Repo: &rhel_vexValue124, FixedInVersion: "0:12.18.4-2.module+el8.1.0+8360+14141500", Self: rhel_vexValue308, Aliases: []claircore.Alias{rhel_vexValue309}},
		{Updater: "rhel-vex", Name: "CVE-2020-8252", Description: rhel_vexValue306, Links: rhel_vexValue311, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue14, Repo: &rhel_vexValue20, FixedInVersion: "0:10.23.1-1.module+el8.3.0+9502+012d8a97", Self: rhel_vexValue308, Aliases: []claircore.Alias{rhel_vexValue309}},
		{Updater: "rhel-vex", Name: "CVE-2020-8252", Description: rhel_vexValue306, Links: rhel_vexValue312, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 2, Package: &rhel_vexValue29, Repo: &rhel_vexValue28, Self: rhel_vexValue308, Aliases: []claircore.Alias{rhel_vexValue309}},
		{Updater: "rhel-vex", Name: "CVE-2020-8252", Description: rhel_vexValue306, Links: rhel_vexValue312, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 2, Package: &rhel_vexValue29, Repo: &rhel_vexValue131, Self: rhel_vexValue308, Aliases: []claircore.Alias{rhel_vexValue309}},
		{Updater: "rhel-vex", Name: "CVE-2020-8252", Description: rhel_vexValue306, Links: rhel_vexValue312, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 2, Package: &rhel_vexValue26, Repo: &rhel_vexValue28, Self: rhel_vexValue308, Aliases: []claircore.Alias{rhel_vexValue309}},
		{Updater: "rhel-vex", Name: "CVE-2022-1897", Description: rhel_vexValue313, Links: rhel_vexValue314, Severity: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue296, Repo: &rhel_vexValue45, FixedInVersion: "2:8.0.1763-19.el8_6.4", Self: rhel_vexValue315, Aliases: []claircore.Alias{rhel_vexValue316}},
		{Updater: "rhel-vex", Name: "CVE-2022-1897", Description: rhel_vexValue313, Links: rhel_vexValue314, Severity: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue296, Repo: &rhel_vexValue20, Self: rhel_vexValue315, Aliases: []claircore.Alias{rhel_vexValue316}},
		{Updater: "rhel-vex", Name: "CVE-2022-1897", Description: rhel_vexValue313, Links: rhel_vexValue314, Severity: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue299, Repo: &rhel_vexValue301, Self: rhel_vexValue315, Aliases: []claircore.Alias{rhel_vexValue316}},
		{Updater: "rhel-vex", Name: "CVE-2022-1897", Description: rhel_vexValue313, Links: rhel_vexValue314, Severity: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue296, Repo: &rhel_vexValue301, Self: rhel_vexValue315, Aliases: []claircore.Alias{rhel_vexValue316}},
		{Updater: "rhel-vex", Name: "CVE-2022-1897", Description: rhel_vexValue313, Links: rhel_vexValue314, Severity: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue299, Repo: &rhel_vexValue220, Self: rhel_vexValue315, Aliases: []claircore.Alias{rhel_vexValue316}},
		{Updater: "rhel-vex", Name: "CVE-2022-1897", Description: rhel_vexValue313, Links: rhel_vexValue314, Severity: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue296, Repo: &rhel_vexValue220, FixedInVersion: "2:8.0.1763-19.el8_6.4", Self: rhel_vexValue315, Aliases: []claircore.Alias{rhel_vexValue316}},
		{Updater: "rhel-vex", Name: "CVE-2022-1897", Description: rhel_vexValue313, Links: rhel_vexValue314, Severity: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue296, Repo: &rhel_vexValue320, FixedInVersion: "2:8.0.1763-19.el8_6.4", Self: rhel_vexValue315, Aliases: []claircore.Alias{rhel_vexValue316}},
		{Updater: "rhel-vex", Name: "CVE-2022-1897", Description: rhel_vexValue313, Links: rhel_vexValue314, Severity: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue299, Repo: &rhel_vexValue20, Self: rhel_vexValue315, Aliases: []claircore.Alias{rhel_vexValue316}},
		{Updater: "rhel-vex", Name: "CVE-2022-1897", Description: rhel_vexValue313, Links: rhel_vexValue321, Severity: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue296, Repo: &rhel_vexValue277, FixedInVersion: "2:8.2.2637-16.el9_0.3", Self: rhel_vexValue315, Aliases: []claircore.Alias{rhel_vexValue316}},
		{Updater: "rhel-vex", Name: "CVE-2022-1897", Description: rhel_vexValue313, Links: rhel_vexValue321, Severity: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue296, Repo: &rhel_vexValue285, FixedInVersion: "2:8.2.2637-16.el9_0.3", Self: rhel_vexValue315, Aliases: []claircore.Alias{rhel_vexValue316}},
		{Updater: "rhel-vex", Name: "CVE-2022-1897", Description: rhel_vexValue313, Links: rhel_vexValue321, Severity: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue296, Repo: &rhel_vexValue273, Self: rhel_vexValue315, Aliases: []claircore.Alias{rhel_vexValue316}},
		{Updater: "rhel-vex", Name: "CVE-2022-1897", Description: rhel_vexValue313, Links: rhel_vexValue322, Severity: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue304, Repo: &rhel_vexValue28, Self: rhel_vexValue315, Aliases: []claircore.Alias{rhel_vexValue316}},
		{Updater: "rhel-vex", Name: "CVE-2022-1897", Description: rhel_vexValue313, Links: rhel_vexValue322, Severity: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue305, Repo: &rhel_vexValue28, Self: rhel_vexValue315, Aliases: []claircore.Alias{rhel_vexValue316}},
		{Updater: "rhel-vex", Name: "CVE-2022-1897", Description: rhel_vexValue313, Links: rhel_vexValue322, Severity: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue304, Repo: &rhel_vexValue135, Self: rhel_vexValue315, Aliases: []claircore.Alias{rhel_vexValue316}},
		{Updater: "rhel-vex", Name: "CVE-2022-1897", Description: rhel_vexValue313, Links: rhel_vexValue322, Severity: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue305, Repo: &rhel_vexValue135, Self: rhel_vexValue315, Aliases: []claircore.Alias{rhel_vexValue316}},
		{Updater: "rhel-vex", Name: "CVE-2020-8116", Description: rhel_vexValue323, Links: rhel_vexValue324, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:L", NormalizedSeverity: 3, Package: &rhel_vexValue14, Repo: &rhel_vexValue20, FixedInVersion: "0:12.18.4-2.module+el8.2.0+8361+192e434e", Self: rhel_vexValue325, Aliases: []claircore.Alias{rhel_vexValue326}},
		{Updater: "rhel-vex", Name: "CVE-2020-8116", Description: rhel_vexValue323, Links: rhel_vexValue327, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:L", NormalizedSeverity: 3, Package: &rhel_vexValue14, Repo: &rhel_vexValue124, FixedInVersion: "0:12.18.4-2.module+el8.1.0+8360+14141500", Self: rhel_vexValue325, Aliases: []claircore.Alias{rhel_vexValue326}},
		{Updater: "rhel-vex", Name: "CVE-2020-8116", Description: rhel_vexValue323, Links: rhel_vexValue328, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:L", NormalizedSeverity: 3, Package: &rhel_vexValue14, Repo: &rhel_vexValue20, FixedInVersion: "0:10.23.1-1.module+el8.3.0+9502+012d8a97", Self: rhel_vexValue325, Aliases: []claircore.Alias{rhel_vexValue326}},
		{Updater: "rhel-vex", Name: "CVE-2020-8116", Description: rhel_vexValue323, Links: rhel_vexValue329, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:L", NormalizedSeverity: 3, Package: &rhel_vexValue29, Repo: &rhel_vexValue28, Self: rhel_vexValue325, Aliases: []claircore.Alias{rhel_vexValue326}},
		{Updater: "rhel-vex", Name: "CVE-2020-8116", Description: rhel_vexValue323, Links: rhel_vexValue329, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:L", NormalizedSeverity: 3, Package: &rhel_vexValue29, Repo: &rhel_vexValue131, Self: rhel_vexValue325, Aliases: []claircore.Alias{rhel_vexValue326}},
		{Updater: "rhel-vex", Name: "CVE-2020-8116", Description: rhel_vexValue323, Links: rhel_vexValue329, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:L", NormalizedSeverity: 3, Package: &rhel_vexValue26, Repo: &rhel_vexValue28, Self: rhel_vexValue325, Aliases: []claircore.Alias{rhel_vexValue326}},
		{Updater: "rhel-vex", Name: "CVE-2020-14195", Description: rhel_vexValue351, Links: rhel_vexValue352, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 2, Package: &rhel_vexValue78, Repo: &rhel_vexValue81, Self: rhel_vexValue353, Aliases: []claircore.Alias{rhel_vexValue354}},
		{Updater: "rhel-vex", Name: "CVE-2017-1000382", Description: rhel_vexValue355, Links: rhel_vexValue356, Severity: "CVSS:3.0/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:N/A:N", NormalizedSeverity: 2, Package: &rhel_vexValue304, Repo: &rhel_vexValue182, Self: rhel_vexValue357, Aliases: []claircore.Alias{rhel_vexValue358}},
		{Updater: "rhel-vex", Name: "CVE-2017-1000382", Description: rhel_vexValue355, Links: rhel_vexValue356, Severity: "CVSS:3.0/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:N/A:N", NormalizedSeverity: 2, Package: &rhel_vexValue304, Repo: &rhel_vexValue9, Self: rhel_vexValue357, Aliases: []claircore.Alias{rhel_vexValue358}},
		{Updater: "rhel-vex", Name: "CVE-2017-1000382", Description: rhel_vexValue355, Links: rhel_vexValue356, Severity: "CVSS:3.0/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:N/A:N", NormalizedSeverity: 2, Package: &rhel_vexValue305, Repo: &rhel_vexValue9, Self: rhel_vexValue357, Aliases: []claircore.Alias{rhel_vexValue358}},
		{Updater: "rhel-vex", Name: "CVE-2017-1000382", Description: rhel_vexValue355, Links: rhel_vexValue356, Severity: "CVSS:3.0/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:N/A:N", NormalizedSeverity: 2, Package: &rhel_vexValue304, Repo: &rhel_vexValue64, Self: rhel_vexValue357, Aliases: []claircore.Alias{rhel_vexValue358}},
		{Updater: "rhel-vex", Name: "CVE-2017-1000382", Description: rhel_vexValue355, Links: rhel_vexValue356, Severity: "CVSS:3.0/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:N/A:N", NormalizedSeverity: 2, Package: &rhel_vexValue305, Repo: &rhel_vexValue64, Self: rhel_vexValue357, Aliases: []claircore.Alias{rhel_vexValue358}},
		{Updater: "rhel-vex", Name: "CVE-2021-3450", Description: rhel_vexValue359, Links: rhel_vexValue360, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:N", NormalizedSeverity: 4, Package: &rhel_vexValue86, Repo: &rhel_vexValue45, FixedInVersion: "1:1.1.1g-15.el8_3", Self: rhel_vexValue361, Aliases: []claircore.Alias{rhel_vexValue362}},
		{Updater: "rhel-vex", Name: "CVE-2021-3450", Description: rhel_vexValue359, Links: rhel_vexValue360, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:N", NormalizedSeverity: 4, Package: &rhel_vexValue92, Repo: &rhel_vexValue45, FixedInVersion: "1:1.1.1g-15.el8_3", Self: rhel_vexValue361, Aliases: []claircore.Alias{rhel_vexValue362}},
		{Updater: "rhel-vex", Name: "CVE-2021-3450", Description: rhel_vexValue359, Links: rhel_vexValue363, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:N", NormalizedSeverity: 4, Package: &rhel_vexValue104, Repo: &rhel_vexValue28, Self: rhel_vexValue361, Aliases: []claircore.Alias{rhel_vexValue362}},
		{Updater: "rhel-vex", Name: "CVE-2021-3450", Description: rhel_vexValue359, Links: rhel_vexValue363, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:N", NormalizedSeverity: 4, Package: &rhel_vexValue105, Repo: &rhel_vexValue28, Self: rhel_vexValue361, Aliases: []claircore.Alias{rhel_vexValue362}},
		{Updater: "rhel-vex", Name: "CVE-2021-3450", Description: rhel_vexValue359, Links: rhel_vexValue363, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:N", NormalizedSeverity: 4, Package: &rhel_vexValue104, Repo: &rhel_vexValue135, Self: rhel_vexValue361, Aliases: []claircore.Alias{rhel_vexValue362}},
		{Updater: "rhel-vex", Name: "CVE-2021-33560", Description: rhel_vexValue526, Links: rhel_vexValue527, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:N/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue217, Repo: &rhel_vexValue45, FixedInVersion: "0:1.8.5-6.el8", Self: rhel_vexValue528, Aliases: []claircore.Alias{rhel_vexValue529}},
		{Updater: "rhel-vex", Name: "CVE-2021-33560", Description: rhel_vexValue526, Links: rhel_vexValue530, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:N/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue224, Repo: &rhel_vexValue9, Self: rhel_vexValue528, Aliases: []claircore.Alias{rhel_vexValue529}},
		{Updater: "rhel-vex", Name: "CVE-2021-33560", Description: rhel_vexValue526, Links: rhel_vexValue530, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:N/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue224, Repo: &rhel_vexValue64, Self: rhel_vexValue528, Aliases: []claircore.Alias{rhel_vexValue529}},
		{Updater: "rhel-vex", Name: "CVE-2021-33560", Description: rhel_vexValue526, Links: rhel_vexValue530, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:N/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue224, Repo: &rhel_vexValue532, Self: rhel_vexValue528, Aliases: []claircore.Alias{rhel_vexValue529}},
		{Updater: "rhel-vex", Name: "CVE-2021-33560", Description: rhel_vexValue526, Links: rhel_vexValue530, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:N/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue224, Repo: &rhel_vexValue135, Self: rhel_vexValue528, Aliases: []claircore.Alias{rhel_vexValue529}},
		{Updater: "rhel-vex", Name: "CVE-2022-3602", Description: rhel_vexValue533, Links: rhel_vexValue534, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue86, Repo: &rhel_vexValue273, FixedInVersion: "1:3.0.1-43.el9_0", Self: rhel_vexValue535, Aliases: []claircore.Alias{rhel_vexValue536}},
		{Updater: "rhel-vex", Name: "CVE-2022-3602", Description: rhel_vexValue533, Links: rhel_vexValue534, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue86, Repo: &rhel_vexValue277, FixedInVersion: "1:3.0.1-43.el9_0", Self: rhel_vexValue535, Aliases: []claircore.Alias{rhel_vexValue536}},
		{Updater: "rhel-vex", Name: "CVE-2022-3602", Description: rhel_vexValue533, Links: rhel_vexValue534, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue92, Repo: &rhel_vexValue277, FixedInVersion: "1:3.0.1-43.el9_0", Self: rhel_vexValue535, Aliases: []claircore.Alias{rhel_vexValue536}},
		{Updater: "rhel-vex", Name: "CVE-2022-3602", Description: rhel_vexValue533, Links: rhel_vexValue534, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue86, Repo: &rhel_vexValue285, FixedInVersion: "1:3.0.1-43.el9_0", Self: rhel_vexValue535, Aliases: []claircore.Alias{rhel_vexValue536}},
		{Updater: "rhel-vex", Name: "CVE-2022-3602", Description: rhel_vexValue533, Links: rhel_vexValue534, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue92, Repo: &rhel_vexValue285, FixedInVersion: "1:3.0.1-43.el9_0", Self: rhel_vexValue535, Aliases: []claircore.Alias{rhel_vexValue536}},
		{Updater: "rhel-vex", Name: "CVE-2022-3602", Description: rhel_vexValue533, Links: rhel_vexValue537, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue104, Repo: &rhel_vexValue135, Self: rhel_vexValue535, Aliases: []claircore.Alias{rhel_vexValue536}},
		{Updater: "rhel-vex", Name: "CVE-2022-3602", Description: rhel_vexValue533, Links: rhel_vexValue537, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue105, Repo: &rhel_vexValue135, Self: rhel_vexValue535, Aliases: []claircore.Alias{rhel_vexValue536}},
		{Updater: "rhel-vex", Name: "CVE-2023-7008", Description: rhel_vexValue551, Links: rhel_vexValue552, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:H/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue553, Repo: &rhel_vexValue285, FixedInVersion: "0:252-32.el9_4", Self: rhel_vexValue554, Aliases: []claircore.Alias{rhel_vexValue555}},
		{Updater: "rhel-vex", Name: "CVE-2023-7008", Description: rhel_vexValue551, Links: rhel_vexValue552, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:H/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue553, Repo: &rhel_vexValue288, Self: rhel_vexValue554, Aliases: []claircore.Alias{rhel_vexValue555}},
		{Updater: "rhel-vex", Name: "CVE-2023-7008", Description: rhel_vexValue551, Links: rhel_vexValue552, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:H/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue553, Repo: &rhel_vexValue557, Self: rhel_vexValue554, Aliases: []claircore.Alias{rhel_vexValue555}},
		{Updater: "rhel-vex", Name: "CVE-2023-7008", Description: rhel_vexValue551, Links: rhel_vexValue558, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:H/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue553, Repo: &rhel_vexValue45, FixedInVersion: "0:239-82.el8", Self: rhel_vexValue554, Aliases: []claircore.Alias{rhel_vexValue555}},
		{Updater: "rhel-vex", Name: "CVE-2023-7008", Description: rhel_vexValue551, Links: rhel_vexValue559, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:H/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue560, Repo: &rhel_vexValue28, Self: rhel_vexValue554, Aliases: []claircore.Alias{rhel_vexValue555}},
		{Updater: "rhel-vex", Name: "CVE-2023-7008", Description: rhel_vexValue551, Links: rhel_vexValue559, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:H/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue560, Repo: &rhel_vexValue135, Self: rhel_vexValue554, Aliases: []claircore.Alias{rhel_vexValue555}},
		{Updater: "rhel-vex", Name: "CVE-2020-9488", Description: rhel_vexValue568, Links: rhel_vexValue569, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:L/I:N/A:N", NormalizedSeverity: 2, Package: &rhel_vexValue570, Repo: &rhel_vexValue81, Self: rhel_vexValue571, Aliases: []claircore.Alias{rhel_vexValue572}},
		{Updater: "rhel-vex", Name: "CVE-2021-3449", Description: rhel_vexValue573, Links: rhel_vexValue574, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue86, Repo: &rhel_vexValue45, FixedInVersion: "1:1.1.1g-15.el8_3", Self: rhel_vexValue575, Aliases: []claircore.Alias{rhel_vexValue576}},
		{Updater: "rhel-vex", Name: "CVE-2021-3449", Description: rhel_vexValue573, Links: rhel_vexValue574, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue92, Repo: &rhel_vexValue45, FixedInVersion: "1:1.1.1g-15.el8_3", Self: rhel_vexValue575, Aliases: []claircore.Alias{rhel_vexValue576}},
		{Updater: "rhel-vex", Name: "CVE-2021-3449", Description: rhel_vexValue573, Links: rhel_vexValue577, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue86, Repo: &rhel_vexValue171, FixedInVersion: "1:1.1.1c-18.el8_2", Self: rhel_vexValue575, Aliases: []claircore.Alias{rhel_vexValue576}},
		{Updater: "rhel-vex", Name: "CVE-2021-3449", Description: rhel_vexValue573, Links: rhel_vexValue577, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue92, Repo: &rhel_vexValue171, FixedInVersion: "1:1.1.1c-18.el8_2", Self: rhel_vexValue575, Aliases: []claircore.Alias{rhel_vexValue576}},
		{Updater: "rhel-vex", Name: "CVE-2021-3449", Description: rhel_vexValue573, Links: rhel_vexValue578, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue86, Repo: &rhel_vexValue168, FixedInVersion: "1:1.1.1c-5.el8_1", Self: rhel_vexValue575, Aliases: []claircore.Alias{rhel_vexValue576}},
		{Updater: "rhel-vex", Name: "CVE-2021-3449", Description: rhel_vexValue573, Links: rhel_vexValue578, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue92, Repo: &rhel_vexValue168, FixedInVersion: "1:1.1.1c-5.el8_1", Self: rhel_vexValue575, Aliases: []claircore.Alias{rhel_vexValue576}},
		{Updater: "rhel-vex", Name: "CVE-2021-3449", Description: rhel_vexValue573, Links: rhel_vexValue579, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue104, Repo: &rhel_vexValue28, Self: rhel_vexValue575, Aliases: []claircore.Alias{rhel_vexValue576}},
		{Updater: "rhel-vex", Name: "CVE-2021-3449", Description: rhel_vexValue573, Links: rhel_vexValue579, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue105, Repo: &rhel_vexValue28, Self: rhel_vexValue575, Aliases: []claircore.Alias{rhel_vexValue576}},
		{Updater: "rhel-vex", Name: "CVE-2021-3449", Description: rhel_vexValue573, Links: rhel_vexValue579, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue104, Repo: &rhel_vexValue135, Self: rhel_vexValue575, Aliases: []claircore.Alias{rhel_vexValue576}},
		{Updater: "rhel-vex", Name: "CVE-2020-14062", Description: rhel_vexValue195, Links: rhel_vexValue580, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 2, Package: &rhel_vexValue78, Repo: &rhel_vexValue81, Self: rhel_vexValue581, Aliases: []claircore.Alias{rhel_vexValue582}},
		{Updater: "rhel-vex", Name: "CVE-2023-4911", Description: rhel_vexValue583, Links: rhel_vexValue584, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue585, Repo: &rhel_vexValue285, FixedInVersion: "0:2.34-60.el9_2.7", Self: rhel_vexValue586, Aliases: []claircore.Alias{rhel_vexValue587}},
		{Updater: "rhel-vex", Name: "CVE-2023-4911", Description: rhel_vexValue583, Links: rhel_vexValue584, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue585, Repo: &rhel_vexValue589, FixedInVersion: "0:2.34-60.el9_2.7", Self: rhel_vexValue586, Aliases: []claircore.Alias{rhel_vexValue587}},
		{Updater: "rhel-vex", Name: "CVE-2023-4911", Description: rhel_vexValue583, Links: rhel_vexValue584, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue585, Repo: &rhel_vexValue557, FixedInVersion: "0:2.34-60.el9_2.7", Self: rhel_vexValue586, Aliases: []claircore.Alias{rhel_vexValue587}},
		{Updater: "rhel-vex", Name: "CVE-2023-4911", Description: rhel_vexValue583, Links: rhel_vexValue584, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue585, Repo: &rhel_vexValue281, FixedInVersion: "0:2.34-60.el9_2.7", Self: rhel_vexValue586, Aliases: []claircore.Alias{rhel_vexValue587}},
		{Updater: "rhel-vex", Name: "CVE-2023-4911", Description: rhel_vexValue583, Links: rhel_vexValue584, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue585, Repo: &rhel_vexValue283, FixedInVersion: "0:2.34-60.el9_2.7", Self: rhel_vexValue586, Aliases: []claircore.Alias{rhel_vexValue587}},
		{Updater: "rhel-vex", Name: "CVE-2023-4911", Description: rhel_vexValue583, Links: rhel_vexValue590, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue585, Repo: &rhel_vexValue273, FixedInVersion: "0:2.34-28.el9_0.4", Self: rhel_vexValue586, Aliases: []claircore.Alias{rhel_vexValue587}},
		{Updater: "rhel-vex", Name: "CVE-2023-4911", Description: rhel_vexValue583, Links: rhel_vexValue590, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue585, Repo: &rhel_vexValue592, FixedInVersion: "0:2.34-28.el9_0.4", Self: rhel_vexValue586, Aliases: []claircore.Alias{rhel_vexValue587}},
		{Updater: "rhel-vex", Name: "CVE-2023-4911", Description: rhel_vexValue583, Links: rhel_vexValue590, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue585, Repo: &rhel_vexValue277, FixedInVersion: "0:2.34-28.el9_0.4", Self: rhel_vexValue586, Aliases: []claircore.Alias{rhel_vexValue587}},
		{Updater: "rhel-vex", Name: "CVE-2023-4911", Description: rhel_vexValue583, Links: rhel_vexValue593, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue585, Repo: &rhel_vexValue596, Self: rhel_vexValue586, Aliases: []claircore.Alias{rhel_vexValue587}},
		{Updater: "rhel-vex", Name: "CVE-2023-4911", Description: rhel_vexValue583, Links: rhel_vexValue593, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue585, Repo: &rhel_vexValue598, FixedInVersion: "0:2.28-225.el8_8.6", Self: rhel_vexValue586, Aliases: []claircore.Alias{rhel_vexValue587}},
		{Updater: "rhel-vex", Name: "CVE-2023-4911", Description: rhel_vexValue583, Links: rhel_vexValue593, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue585, Repo: &rhel_vexValue600, Self: rhel_vexValue586, Aliases: []claircore.Alias{rhel_vexValue587}},
		{Updater: "rhel-vex", Name: "CVE-2023-4911", Description: rhel_vexValue583, Links: rhel_vexValue593, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue585, Repo: &rhel_vexValue20, Self: rhel_vexValue586, Aliases: []claircore.Alias{rhel_vexValue587}},
		{Updater: "rhel-vex", Name: "CVE-2023-4911", Description: rhel_vexValue583, Links: rhel_vexValue593, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue585, Repo: &rhel_vexValue20, FixedInVersion: "0:2.28-225.el8_8.6", Self: rhel_vexValue586, Aliases: []claircore.Alias{rhel_vexValue587}},
		{Updater: "rhel-vex", Name: "CVE-2023-4911", Description: rhel_vexValue583, Links: rhel_vexValue593, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue585, Repo: &rhel_vexValue45, FixedInVersion: "0:2.28-225.el8_8.6", Self: rhel_vexValue586, Aliases: []claircore.Alias{rhel_vexValue587}},
		{Updater: "rhel-vex", Name: "CVE-2023-4911", Description: rhel_vexValue583, Links: rhel_vexValue593, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue585, Repo: &rhel_vexValue43, FixedInVersion: "0:2.28-225.el8_8.6", Self: rhel_vexValue586, Aliases: []claircore.Alias{rhel_vexValue587}},
		{Updater: "rhel-vex", Name: "CVE-2023-4911", Description: rhel_vexValue583, Links: rhel_vexValue601, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue585, Repo: &rhel_vexValue301, FixedInVersion: "0:2.28-189.6.el8_6", Self: rhel_vexValue586, Aliases: []claircore.Alias{rhel_vexValue587}},
		{Updater: "rhel-vex", Name: "CVE-2023-4911", Description: rhel_vexValue583, Links: rhel_vexValue601, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue585, Repo: &rhel_vexValue220, FixedInVersion: "0:2.28-189.6.el8_6", Self: rhel_vexValue586, Aliases: []claircore.Alias{rhel_vexValue587}},
		{Updater: "rhel-vex", Name: "CVE-2023-4911", Description: rhel_vexValue583, Links: rhel_vexValue601, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue585, Repo: &rhel_vexValue603, FixedInVersion: "0:2.28-189.6.el8_6", Self: rhel_vexValue586, Aliases: []claircore.Alias{rhel_vexValue587}},
		{Updater: "rhel-vex", Name: "CVE-2023-4911", Description: rhel_vexValue583, Links: rhel_vexValue601, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue585, Repo: &rhel_vexValue320, FixedInVersion: "0:2.28-189.6.el8_6", Self: rhel_vexValue586, Aliases: []claircore.Alias{rhel_vexValue587}},
		{Updater: "rhel-vex", Name: "CVE-2023-4911", Description: rhel_vexValue583, Links: rhel_vexValue604, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue605, Repo: &rhel_vexValue28, Self: rhel_vexValue586, Aliases: []claircore.Alias{rhel_vexValue587}},
		{Updater: "rhel-vex", Name: "CVE-2023-4911", Description: rhel_vexValue583, Links: rhel_vexValue604, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue605, Repo: &rhel_vexValue135, Self: rhel_vexValue586, Aliases: []claircore.Alias{rhel_vexValue587}},
		{Updater: "rhel-vex", Name: "CVE-2020-15095", Description: rhel_vexValue606, Links: rhel_vexValue607, Severity: "CVSS:3.1/AV:L/AC:H/PR:L/UI:R/S:U/C:H/I:N/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue14, Repo: &rhel_vexValue20, FixedInVersion: "0:12.18.4-2.module+el8.2.0+8361+192e434e", Self: rhel_vexValue608, Aliases: []claircore.Alias{rhel_vexValue609}},
		{Updater: "rhel-vex", Name: "CVE-2020-15095", Description: rhel_vexValue606, Links: rhel_vexValue610, Severity: "CVSS:3.1/AV:L/AC:H/PR:L/UI:R/S:U/C:H/I:N/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue14, Repo: &rhel_vexValue124, FixedInVersion: "0:12.18.4-2.module+el8.1.0+8360+14141500", Self: rhel_vexValue608, Aliases: []claircore.Alias{rhel_vexValue609}},
		{Updater: "rhel-vex", Name: "CVE-2020-15095", Description: rhel_vexValue606, Links: rhel_vexValue611, Severity: "CVSS:3.1/AV:L/AC:H/PR:L/UI:R/S:U/C:H/I:N/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue14, Repo: &rhel_vexValue20, FixedInVersion: "0:10.23.1-1.module+el8.3.0+9502+012d8a97", Self: rhel_vexValue608, Aliases: []claircore.Alias{rhel_vexValue609}},
		{Updater: "rhel-vex", Name: "CVE-2020-15095", Description: rhel_vexValue606, Links: rhel_vexValue612, Severity: "CVSS:3.1/AV:L/AC:H/PR:L/UI:R/S:U/C:H/I:N/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue29, Repo: &rhel_vexValue28, Self: rhel_vexValue608, Aliases: []claircore.Alias{rhel_vexValue609}},
		{Updater: "rhel-vex", Name: "CVE-2020-15095", Description: rhel_vexValue606, Links: rhel_vexValue612, Severity: "CVSS:3.1/AV:L/AC:H/PR:L/UI:R/S:U/C:H/I:N/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue29, Repo: &rhel_vexValue131, Self: rhel_vexValue608, Aliases: []claircore.Alias{rhel_vexValue609}},
		{Updater: "rhel-vex", Name: "CVE-2020-15095", Description: rhel_vexValue606, Links: rhel_vexValue612, Severity: "CVSS:3.1/AV:L/AC:H/PR:L/UI:R/S:U/C:H/I:N/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue26, Repo: &rhel_vexValue28, Self: rhel_vexValue608, Aliases: []claircore.Alias{rhel_vexValue609}},
		{Updater: "rhel-vex", Name: "CVE-2022-34177", Description: rhel_vexValue613, Links: rhel_vexValue614, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:H/A:N", NormalizedSeverity: 4, Package: &rhel_vexValue227, Repo: &rhel_vexValue616, Self: rhel_vexValue617, Aliases: []claircore.Alias{rhel_vexValue618}},
		{Updater: "rhel-vex", Name: "CVE-2022-34177", Description: rhel_vexValue613, Links: rhel_vexValue614, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:H/A:N", NormalizedSeverity: 4, Package: &rhel_vexValue234, Repo: &rhel_vexValue616, FixedInVersion: "0:4.10.1663147786-1.el8", Self: rhel_vexValue617, Aliases: []claircore.Alias{rhel_vexValue618}},
		{Updater: "rhel-vex", Name: "CVE-2022-34177", Description: rhel_vexValue613, Links: rhel_vexValue619, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:H/A:N", NormalizedSeverity: 4, Package: &rhel_vexValue227, Repo: &rhel_vexValue621, Self: rhel_vexValue617, Aliases: []claircore.Alias{rhel_vexValue618}},
		{Updater: "rhel-vex", Name: "CVE-2022-34177", Description: rhel_vexValue613, Links: rhel_vexValue619, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:H/A:N", NormalizedSeverity: 4, Package: &rhel_vexValue234, Repo: &rhel_vexValue621, FixedInVersion: "0:4.9.1669894222-1.el8", Self: rhel_vexValue617, Aliases: []claircore.Alias{rhel_vexValue618}},
		{Updater: "rhel-vex", Name: "CVE-2022-34177", Description: rhel_vexValue613, Links: rhel_vexValue622, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:H/A:N", NormalizedSeverity: 4, Package: &rhel_vexValue227, Repo: &rhel_vexValue624, Self: rhel_vexValue617, Aliases: []claircore.Alias{rhel_vexValue618}},
		{Updater: "rhel-vex", Name: "CVE-2022-34177", Description: rhel_vexValue613, Links: rhel_vexValue622, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:H/A:N", NormalizedSeverity: 4, Package: &rhel_vexValue234, Repo: &rhel_vexValue624, FixedInVersion: "0:4.8.1672842762-1.el8", Self: rhel_vexValue617, Aliases: []claircore.Alias{rhel_vexValue618}},
		{Updater: "rhel-vex", Name: "CVE-2022-34177", Description: rhel_vexValue613, Links: rhel_vexValue625, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:H/A:N", NormalizedSeverity: 4, Package: &rhel_vexValue244, Repo: &rhel_vexValue254, Self: rhel_vexValue617, Aliases: []claircore.Alias{rhel_vexValue618}},
		{Updater: "rhel-vex", Name: "CVE-2022-34177", Description: rhel_vexValue613, Links: rhel_vexValue625, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:H/A:N", NormalizedSeverity: 4, Package: &rhel_vexValue244, Repo: &rhel_vexValue256, Self: rhel_vexValue617, Aliases: []claircore.Alias{rhel_vexValue618}},
		{Updater: "rhel-vex", Name: "CVE-2022-34177", Description: rhel_vexValue613, Links: rhel_vexValue625, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:H/A:N", NormalizedSeverity: 4, Package: &rhel_vexValue244, Repo: &rhel_vexValue628, Self: rhel_vexValue617, Aliases: []claircore.Alias{rhel_vexValue618}},
		{Updater: "rhel-vex", Name: "CVE-2022-34177", Description: rhel_vexValue613, Links: rhel_vexValue625, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:H/A:N", NormalizedSeverity: 4, Package: &rhel_vexValue244, Repo: &rhel_vexValue631, Self: rhel_vexValue617, Aliases: []claircore.Alias{rhel_vexValue618}},
		{Updater: "rhel-vex", Name: "CVE-2022-34177", Description: rhel_vexValue613, Links: rhel_vexValue625, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:H/A:N", NormalizedSeverity: 4, Package: &rhel_vexValue244, Repo: &rhel_vexValue632, Self: rhel_vexValue617, Aliases: []claircore.Alias{rhel_vexValue618}},
		{Updater: "rhel-vex", Name: "CVE-2022-34177", Description: rhel_vexValue613, Links: rhel_vexValue625, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:H/A:N", NormalizedSeverity: 4, Package: &rhel_vexValue244, Repo: &rhel_vexValue261, Self: rhel_vexValue617, Aliases: []claircore.Alias{rhel_vexValue618}},
		{Updater: "rhel-vex", Name: "CVE-2022-34177", Description: rhel_vexValue613, Links: rhel_vexValue625, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:H/A:N", NormalizedSeverity: 4, Package: &rhel_vexValue244, Repo: &rhel_vexValue635, Self: rhel_vexValue617, Aliases: []claircore.Alias{rhel_vexValue618}},
		{Updater: "rhel-vex", Name: "CVE-2022-3786", Description: rhel_vexValue643, Links: rhel_vexValue644, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue86, Repo: &rhel_vexValue273, FixedInVersion: "1:3.0.1-43.el9_0", Self: rhel_vexValue645, Aliases: []claircore.Alias{rhel_vexValue646}},
		{Updater: "rhel-vex", Name: "CVE-2022-3786", Description: rhel_vexValue643, Links: rhel_vexValue644, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue86, Repo: &rhel_vexValue277, FixedInVersion: "1:3.0.1-43.el9_0", Self: rhel_vexValue645, Aliases: []claircore.Alias{rhel_vexValue646}},
		{Updater: "rhel-vex", Name: "CVE-2022-3786", Description: rhel_vexValue643, Links: rhel_vexValue644, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue92, Repo: &rhel_vexValue277, FixedInVersion: "1:3.0.1-43.el9_0", Self: rhel_vexValue645, Aliases: []claircore.Alias{rhel_vexValue646}},
		{Updater: "rhel-vex", Name: "CVE-2022-3786", Description: rhel_vexValue643, Links: rhel_vexValue644, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue86, Repo: &rhel_vexValue285, FixedInVersion: "1:3.0.1-43.el9_0", Self: rhel_vexValue645, Aliases: []claircore.Alias{rhel_vexValue646}},
		{Updater: "rhel-vex", Name: "CVE-2022-3786", Description: rhel_vexValue643, Links: rhel_vexValue644, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue92, Repo: &rhel_vexValue285, FixedInVersion: "1:3.0.1-43.el9_0", Self: rhel_vexValue645, Aliases: []claircore.Alias{rhel_vexValue646}},
		{Updater: "rhel-vex", Name: "CVE-2022-3786", Description: rhel_vexValue643, Links: rhel_vexValue647, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue104, Repo: &rhel_vexValue135, Self: rhel_vexValue645, Aliases: []claircore.Alias{rhel_vexValue646}},
		{Updater: "rhel-vex", Name: "CVE-2022-3786", Description: rhel_vexValue643, Links: rhel_vexValue647, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue105, Repo: &rhel_vexValue135, Self: rhel_vexValue645, Aliases: []claircore.Alias{rhel_vexValue646}},
		{Updater: "rhel-vex", Name: "CVE-2022-2097", Description: rhel_vexValue648, Links: rhel_vexValue649, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:N/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue86, Repo: &rhel_vexValue45, FixedInVersion: "1:1.1.1k-7.el8_6", Self: rhel_vexValue650, Aliases: []claircore.Alias{rhel_vexValue651}},
		{Updater: "rhel-vex", Name: "CVE-2022-2097", Description: rhel_vexValue648, Links: rhel_vexValue649, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:N/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue92, Repo: &rhel_vexValue220, FixedInVersion: "1:1.1.1k-7.el8_6", Self: rhel_vexValue650, Aliases: []claircore.Alias{rhel_vexValue651}},
		{Updater: "rhel-vex", Name: "CVE-2022-2097", Description: rhel_vexValue648, Links: rhel_vexValue649, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:N/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue86, Repo: &rhel_vexValue220, FixedInVersion: "1:1.1.1k-7.el8_6", Self: rhel_vexValue650, Aliases: []claircore.Alias{rhel_vexValue651}},
		{Updater: "rhel-vex", Name: "CVE-2022-2097", Description: rhel_vexValue648, Links: rhel_vexValue649, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:N/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue92, Repo: &rhel_vexValue45, FixedInVersion: "1:1.1.1k-7.el8_6", Self: rhel_vexValue650, Aliases: []claircore.Alias{rhel_vexValue651}},
		{Updater: "rhel-vex", Name: "CVE-2022-2097", Description: rhel_vexValue648, Links: rhel_vexValue652, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:N/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue86, Repo: &rhel_vexValue273, Self: rhel_vexValue650, Aliases: []claircore.Alias{rhel_vexValue651}},
		{Updater: "rhel-vex", Name: "CVE-2022-2097", Description: rhel_vexValue648, Links: rhel_vexValue652, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:N/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue92, Repo: &rhel_vexValue277, FixedInVersion: "1:3.0.1-41.el9_0", Self: rhel_vexValue650, Aliases: []claircore.Alias{rhel_vexValue651}},
		{Updater: "rhel-vex", Name: "CVE-2022-2097", Description: rhel_vexValue648, Links: rhel_vexValue652, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:N/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue86, Repo: &rhel_vexValue277, FixedInVersion: "1:3.0.1-41.el9_0", Self: rhel_vexValue650, Aliases: []claircore.Alias{rhel_vexValue651}},
		{Updater: "rhel-vex", Name: "CVE-2022-2097", Description: rhel_vexValue648, Links: rhel_vexValue652, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:N/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue92, Repo: &rhel_vexValue273, Self: rhel_vexValue650, Aliases: []claircore.Alias{rhel_vexValue651}},
		{Updater: "rhel-vex", Name: "CVE-2022-2097", Description: rhel_vexValue648, Links: rhel_vexValue652, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:N/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue86, Repo: &rhel_vexValue285, FixedInVersion: "1:3.0.1-41.el9_0", Self: rhel_vexValue650, Aliases: []claircore.Alias{rhel_vexValue651}},
		{Updater: "rhel-vex", Name: "CVE-2022-2097", Description: rhel_vexValue648, Links: rhel_vexValue652, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:N/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue92, Repo: &rhel_vexValue285, FixedInVersion: "1:3.0.1-41.el9_0", Self: rhel_vexValue650, Aliases: []claircore.Alias{rhel_vexValue651}},
		{Updater: "rhel-vex", Name: "CVE-2022-2097", Description: rhel_vexValue648, Links: rhel_vexValue653, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:N/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue104, Repo: &rhel_vexValue28, Self: rhel_vexValue650, Aliases: []claircore.Alias{rhel_vexValue651}},
		{Updater: "rhel-vex", Name: "CVE-2022-2097", Description: rhel_vexValue648, Links: rhel_vexValue653, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:N/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue105, Repo: &rhel_vexValue28, Self: rhel_vexValue650, Aliases: []claircore.Alias{rhel_vexValue651}},
		{Updater: "rhel-vex", Name: "CVE-2022-2097", Description: rhel_vexValue648, Links: rhel_vexValue653, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:N/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue104, Repo: &rhel_vexValue135, Self: rhel_vexValue650, Aliases: []claircore.Alias{rhel_vexValue651}},
		{Updater: "rhel-vex", Name: "CVE-2022-2097", Description: rhel_vexValue648, Links: rhel_vexValue653, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:N/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue105, Repo: &rhel_vexValue135, Self: rhel_vexValue650, Aliases: []claircore.Alias{rhel_vexValue651}},
		{Updater: "rhel-vex", Name: "CVE-2021-33910", Description: rhel_vexValue654, Links: rhel_vexValue655, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue553, Repo: &rhel_vexValue45, FixedInVersion: "0:239-45.el8_4.2", Self: rhel_vexValue656, Aliases: []claircore.Alias{rhel_vexValue657}},
		{Updater: "rhel-vex", Name: "CVE-2021-33910", Description: rhel_vexValue654, Links: rhel_vexValue655, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue553, Repo: &rhel_vexValue38, FixedInVersion: "0:239-45.el8_4.2", Self: rhel_vexValue656, Aliases: []claircore.Alias{rhel_vexValue657}},
		{Updater: "rhel-vex", Name: "CVE-2021-33910", Description: rhel_vexValue654, Links: rhel_vexValue658, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue553, Repo: &rhel_vexValue171, FixedInVersion: "0:239-31.el8_2.4", Self: rhel_vexValue656, Aliases: []claircore.Alias{rhel_vexValue657}},
		{Updater: "rhel-vex", Name: "CVE-2021-33910", Description: rhel_vexValue654, Links: rhel_vexValue659, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue553, Repo: &rhel_vexValue168, FixedInVersion: "0:239-18.el8_1.8", Self: rhel_vexValue656, Aliases: []claircore.Alias{rhel_vexValue657}},
		{Updater: "rhel-vex", Name: "CVE-2021-33910", Description: rhel_vexValue654, Links: rhel_vexValue660, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue560, Repo: &rhel_vexValue28, Self: rhel_vexValue656, Aliases: []claircore.Alias{rhel_vexValue657}},
		{Updater: "rhel-vex", Name: "CVE-2021-33910", Description: rhel_vexValue654, Links: rhel_vexValue660, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue560, Repo: &rhel_vexValue135, Self: rhel_vexValue656, Aliases: []claircore.Alias{rhel_vexValue657}},
		{Updater: "rhel-vex", Name: "CVE-2020-8287", Description: rhel_vexValue735, Links: rhel_vexValue736, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:U/C:L/I:L/A:N", NormalizedSeverity: 2, Package: &rhel_vexValue14, Repo: &rhel_vexValue20, FixedInVersion: "0:10.23.1-1.module+el8.3.0+9502+012d8a97", Self: rhel_vexValue737, Aliases: []claircore.Alias{rhel_vexValue738}},
		{Updater: "rhel-vex", Name: "CVE-2020-8287", Description: rhel_vexValue735, Links: rhel_vexValue739, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:U/C:L/I:L/A:N", NormalizedSeverity: 2, Package: &rhel_vexValue14, Repo: &rhel_vexValue20, FixedInVersion: "0:12.20.1-1.module+el8.3.0+9503+19cb079c", Self: rhel_vexValue737, Aliases: []claircore.Alias{rhel_vexValue738}},
		{Updater: "rhel-vex", Name: "CVE-2020-8287", Description: rhel_vexValue735, Links: rhel_vexValue740, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:U/C:L/I:L/A:N", NormalizedSeverity: 2, Package: &rhel_vexValue14, Repo: &rhel_vexValue20, FixedInVersion: "0:14.15.4-2.module+el8.3.0+9635+ffdf8381", Self: rhel_vexValue737, Aliases: []claircore.Alias{rhel_vexValue738}},
		{Updater: "rhel-vex", Name: "CVE-2020-8287", Description: rhel_vexValue735, Links: rhel_vexValue741, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:U/C:L/I:L/A:N", NormalizedSeverity: 2, Package: &rhel_vexValue26, Repo: &rhel_vexValue28, Self: rhel_vexValue737, Aliases: []claircore.Alias{rhel_vexValue738}},
		{Updater: "rhel-vex", Name: "CVE-2020-8287", Description: rhel_vexValue735, Links: rhel_vexValue741, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:U/C:L/I:L/A:N", NormalizedSeverity: 2, Package: &rhel_vexValue29, Repo: &rhel_vexValue28, Self: rhel_vexValue737, Aliases: []claircore.Alias{rhel_vexValue738}},
		{Updater: "rhel-vex", Name: "CVE-2020-8287", Description: rhel_vexValue735, Links: rhel_vexValue741, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:U/C:L/I:L/A:N", NormalizedSeverity: 2, Package: &rhel_vexValue30, Repo: &rhel_vexValue28, Self: rhel_vexValue737, Aliases: []claircore.Alias{rhel_vexValue738}},
		{Updater: "rhel-vex", Name: "CVE-2020-8287", Description: rhel_vexValue735, Links: rhel_vexValue741, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:U/C:L/I:L/A:N", NormalizedSeverity: 2, Package: &rhel_vexValue132, Repo: &rhel_vexValue135, Self: rhel_vexValue737, Aliases: []claircore.Alias{rhel_vexValue738}},
		{Updater: "rhel-vex", Name: "CVE-2017-9614", Description: rhel_vexValue742, Links: rhel_vexValue743, Severity: "CVSS:3.0/AV:L/AC:L/PR:N/UI:R/S:U/C:N/I:N/A:L", NormalizedSeverity: 2, Package: &rhel_vexValue744, Repo: &rhel_vexValue9, Self: rhel_vexValue745, Aliases: []claircore.Alias{rhel_vexValue746}},
		{Updater: "rhel-vex", Name: "CVE-2017-9614", Description: rhel_vexValue742, Links: rhel_vexValue743, Severity: "CVSS:3.0/AV:L/AC:L/PR:N/UI:R/S:U/C:N/I:N/A:L", NormalizedSeverity: 2, Package: &rhel_vexValue744, Repo: &rhel_vexValue64, Self: rhel_vexValue745, Aliases: []claircore.Alias{rhel_vexValue746}},
		{Updater: "rhel-vex", Name: "CVE-2020-7788", Description: rhel_vexValue747, Links: rhel_vexValue748, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:L", NormalizedSeverity: 3, Package: &rhel_vexValue14, Repo: &rhel_vexValue20, FixedInVersion: "0:10.23.1-1.module+el8.3.0+9502+012d8a97", Self: rhel_vexValue749, Aliases: []claircore.Alias{rhel_vexValue750}},
		{Updater: "rhel-vex", Name: "CVE-2020-7788", Description: rhel_vexValue747, Links: rhel_vexValue751, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:L", NormalizedSeverity: 3, Package: &rhel_vexValue14, Repo: &rhel_vexValue20, FixedInVersion: "0:12.20.1-1.module+el8.3.0+9503+19cb079c", Self: rhel_vexValue749, Aliases: []claircore.Alias{rhel_vexValue750}},
		{Updater: "rhel-vex", Name: "CVE-2020-7788", Description: rhel_vexValue747, Links: rhel_vexValue752, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:L", NormalizedSeverity: 3, Package: &rhel_vexValue14, Repo: &rhel_vexValue20, FixedInVersion: "0:14.15.4-2.module+el8.3.0+9635+ffdf8381", Self: rhel_vexValue749, Aliases: []claircore.Alias{rhel_vexValue750}},
		{Updater: "rhel-vex", Name: "CVE-2020-7788", Description: rhel_vexValue747, Links: rhel_vexValue753, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:L", NormalizedSeverity: 3, Package: &rhel_vexValue14, Repo: &rhel_vexValue755, FixedInVersion: "0:14.18.2-2.module+el8.4.0+13643+6c0ebf22", Self: rhel_vexValue749, Aliases: []claircore.Alias{rhel_vexValue750}},
		{Updater: "rhel-vex", Name: "CVE-2020-7788", Description: rhel_vexValue747, Links: rhel_vexValue756, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:L", NormalizedSeverity: 3, Package: &rhel_vexValue26, Repo: &rhel_vexValue760, Self: rhel_vexValue749, Aliases: []claircore.Alias{rhel_vexValue750}},
		{Updater: "rhel-vex", Name: "CVE-2020-7788", Description: rhel_vexValue747, Links: rhel_vexValue756, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:L", NormalizedSeverity: 3, Package: &rhel_vexValue26, Repo: &rhel_vexValue131, Self: rhel_vexValue749, Aliases: []claircore.Alias{rhel_vexValue750}},
		{Updater: "rhel-vex", Name: "CVE-2020-7788", Description: rhel_vexValue747, Links: rhel_vexValue756, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:L", NormalizedSeverity: 3, Package: &rhel_vexValue26, Repo: &rhel_vexValue28, Self: rhel_vexValue749, Aliases: []claircore.Alias{rhel_vexValue750}},
		{Updater: "rhel-vex", Name: "CVE-2020-7788", Description: rhel_vexValue747, Links: rhel_vexValue756, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:L", NormalizedSeverity: 3, Package: &rhel_vexValue29, Repo: &rhel_vexValue28, Self: rhel_vexValue749, Aliases: []claircore.Alias{rhel_vexValue750}},
		{Updater: "rhel-vex", Name: "CVE-2020-7788", Description: rhel_vexValue747, Links: rhel_vexValue756, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:L", NormalizedSeverity: 3, Package: &rhel_vexValue30, Repo: &rhel_vexValue28, Self: rhel_vexValue749, Aliases: []claircore.Alias{rhel_vexValue750}},
		{Updater: "rhel-vex", Name: "CVE-2020-7788", Description: rhel_vexValue747, Links: rhel_vexValue756, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:L", NormalizedSeverity: 3, Package: &rhel_vexValue26, Repo: &rhel_vexValue532, Self: rhel_vexValue749, Aliases: []claircore.Alias{rhel_vexValue750}},
		{Updater: "rhel-vex", Name: "CVE-2020-7788", Description: rhel_vexValue747, Links: rhel_vexValue756, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:L", NormalizedSeverity: 3, Package: &rhel_vexValue30, Repo: &rhel_vexValue532, Self: rhel_vexValue749, Aliases: []claircore.Alias{rhel_vexValue750}},
		{Updater: "rhel-vex", Name: "CVE-2020-7788", Description: rhel_vexValue747, Links: rhel_vexValue756, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:L", NormalizedSeverity: 3, Package: &rhel_vexValue761, Repo: &rhel_vexValue28, Self: rhel_vexValue749, Aliases: []claircore.Alias{rhel_vexValue750}},
		{Updater: "rhel-vex", Name: "CVE-2020-7788", Description: rhel_vexValue747, Links: rhel_vexValue756, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:L", NormalizedSeverity: 3, Package: &rhel_vexValue26, Repo: &rhel_vexValue763, Self: rhel_vexValue749, Aliases: []claircore.Alias{rhel_vexValue750}},
		{Updater: "rhel-vex", Name: "CVE-2020-7788", Description: rhel_vexValue747, Links: rhel_vexValue756, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:L", NormalizedSeverity: 3, Package: &rhel_vexValue26, Repo: &rhel_vexValue765, Self: rhel_vexValue749, Aliases: []claircore.Alias{rhel_vexValue750}},
		{Updater: "rhel-vex", Name: "CVE-2021-33929", Description: rhel_vexValue766, Links: rhel_vexValue767, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue33, Repo: &rhel_vexValue38, FixedInVersion: "0:0.7.16-3.el8_4", Self: rhel_vexValue768, Aliases: []claircore.Alias{rhel_vexValue769}},
		{Updater: "rhel-vex", Name: "CVE-2021-33929", Description: rhel_vexValue766, Links: rhel_vexValue767, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue33, Repo: &rhel_vexValue43, Self: rhel_vexValue768, Aliases: []claircore.Alias{rhel_vexValue769}},
		{Updater: "rhel-vex", Name: "CVE-2021-33929", Description: rhel_vexValue766, Links: rhel_vexValue767, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue33, Repo: &rhel_vexValue45, FixedInVersion: "0:0.7.16-3.el8_4", Self: rhel_vexValue768, Aliases: []claircore.Alias{rhel_vexValue769}},
		{Updater: "rhel-vex", Name: "CVE-2021-33929", Description: rhel_vexValue766, Links: rhel_vexValue767, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue33, Repo: &rhel_vexValue43, FixedInVersion: "0:0.7.16-3.el8_4", Self: rhel_vexValue768, Aliases: []claircore.Alias{rhel_vexValue769}},
		{Updater: "rhel-vex", Name: "CVE-2021-33929", Description: rhel_vexValue766, Links: rhel_vexValue770, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue33, Repo: &rhel_vexValue51, FixedInVersion: "0:0.7.22-1.el7pc", Self: rhel_vexValue768, Aliases: []claircore.Alias{rhel_vexValue769}},
		{Updater: "rhel-vex", Name: "CVE-2021-33929", Description: rhel_vexValue766, Links: rhel_vexValue770, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue33, Repo: &rhel_vexValue54, FixedInVersion: "0:0.7.22-1.el8pc", Self: rhel_vexValue768, Aliases: []claircore.Alias{rhel_vexValue769}},
		{Updater: "rhel-vex", Name: "CVE-2021-33929", Description: rhel_vexValue766, Links: rhel_vexValue770, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue33, Repo: &rhel_vexValue57, FixedInVersion: "0:0.7.22-1.el7pc", Self: rhel_vexValue768, Aliases: []claircore.Alias{rhel_vexValue769}},
		{Updater: "rhel-vex", Name: "CVE-2021-33929", Description: rhel_vexValue766, Links: rhel_vexValue770, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue33, Repo: &rhel_vexValue59, FixedInVersion: "0:0.7.22-1.el8pc", Self: rhel_vexValue768, Aliases: []claircore.Alias{rhel_vexValue769}},
		{Updater: "rhel-vex", Name: "CVE-2021-33929", Description: rhel_vexValue766, Links: rhel_vexValue771, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue61, Repo: &rhel_vexValue64, Self: rhel_vexValue768, Aliases: []claircore.Alias{rhel_vexValue769}},
		{Updater: "rhel-vex", Name: "CVE-2021-33929", Description: rhel_vexValue766, Links: rhel_vexValue771, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue61, Repo: &rhel_vexValue28, Self: rhel_vexValue768, Aliases: []claircore.Alias{rhel_vexValue769}},
		{Updater: "rhel-vex", Name: "CVE-2021-33929", Description: rhel_vexValue766, Links: rhel_vexValue771, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue61, Repo: &rhel_vexValue66, Self: rhel_vexValue768, Aliases: []claircore.Alias{rhel_vexValue769}},
		{Updater: "rhel-vex", Name: "CVE-2021-33929", Description: rhel_vexValue766, Links: rhel_vexValue771, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue61, Repo: &rhel_vexValue70, Self: rhel_vexValue768, Aliases: []claircore.Alias{rhel_vexValue769}},
		{Updater: "rhel-vex", Name: "CVE-2023-38546", Description: rhel_vexValue772, Links: rhel_vexValue773, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:L/A:N", NormalizedSeverity: 2, Package: &rhel_vexValue138, Repo: &rhel_vexValue273, FixedInVersion: "0:7.76.1-14.el9_0.9", Self: rhel_vexValue774, Aliases: []claircore.Alias{rhel_vexValue775}},
		{Updater: "rhel-vex", Name: "CVE-2023-38546", Description: rhel_vexValue772, Links: rhel_vexValue773, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:L/A:N", NormalizedSeverity: 2, Package: &rhel_vexValue138, Repo: &rhel_vexValue277, FixedInVersion: "0:7.76.1-14.el9_0.9", Self: rhel_vexValue774, Aliases: []claircore.Alias{rhel_vexValue775}},
		{Updater: "rhel-vex", Name: "CVE-2023-38546", Description: rhel_vexValue772, Links: rhel_vexValue776, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:L/A:N", NormalizedSeverity: 2, Package: &rhel_vexValue138, Repo: &rhel_vexValue285, FixedInVersion: "0:7.76.1-23.el9_2.4", Self: rhel_vexValue774, Aliases: []claircore.Alias{rhel_vexValue775}},
		{Updater: "rhel-vex", Name: "CVE-2023-38546", Description: rhel_vexValue772, Links: rhel_vexValue776, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:L/A:N", NormalizedSeverity: 2, Package: &rhel_vexValue138, Repo: &rhel_vexValue281, FixedInVersion: "0:7.76.1-23.el9_2.4", Self: rhel_vexValue774, Aliases: []claircore.Alias{rhel_vexValue775}},
		{Updater: "rhel-vex", Name: "CVE-2023-38546", Description: rhel_vexValue772, Links: rhel_vexValue776, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:L/A:N", NormalizedSeverity: 2, Package: &rhel_vexValue138, Repo: &rhel_vexValue283, FixedInVersion: "0:7.76.1-23.el9_2.4", Self: rhel_vexValue774, Aliases: []claircore.Alias{rhel_vexValue775}},
		{Updater: "rhel-vex", Name: "CVE-2023-38546", Description: rhel_vexValue772, Links: rhel_vexValue777, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:L/A:N", NormalizedSeverity: 2, Package: &rhel_vexValue138, Repo: &rhel_vexValue220, FixedInVersion: "0:7.61.1-22.el8_6.9", Self: rhel_vexValue774, Aliases: []claircore.Alias{rhel_vexValue775}},
		{Updater: "rhel-vex", Name: "CVE-2023-38546", Description: rhel_vexValue772, Links: rhel_vexValue778, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:L/A:N", NormalizedSeverity: 2, Package: &rhel_vexValue138, Repo: &rhel_vexValue288, FixedInVersion: "0:7.76.1-26.el9_3.2", Self: rhel_vexValue774, Aliases: []claircore.Alias{rhel_vexValue775}},
		{Updater: "rhel-vex", Name: "CVE-2023-38546", Description: rhel_vexValue772, Links: rhel_vexValue778, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:L/A:N", NormalizedSeverity: 2, Package: &rhel_vexValue138, Repo: &rhel_vexValue285, FixedInVersion: "0:7.76.1-26.el9_3.2", Self: rhel_vexValue774, Aliases: []claircore.Alias{rhel_vexValue775}},
		{Updater: "rhel-vex", Name: "CVE-2023-38546", Description: rhel_vexValue772, Links: rhel_vexValue779, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:L/A:N", NormalizedSeverity: 2, Package: &rhel_vexValue138, Repo: &rhel_vexValue598, FixedInVersion: "0:7.61.1-30.el8_8.6", Self: rhel_vexValue774, Aliases: []claircore.Alias{rhel_vexValue775}},
		{Updater: "rhel-vex", Name: "CVE-2023-38546", Description: rhel_vexValue772, Links: rhel_vexValue780, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:L/A:N", NormalizedSeverity: 2, Package: &rhel_vexValue138, Repo: &rhel_vexValue45, FixedInVersion: "0:7.61.1-33.el8_9.5", Self: rhel_vexValue774, Aliases: []claircore.Alias{rhel_vexValue775}},
		{Updater: "rhel-vex", Name: "CVE-2023-38546", Description: rhel_vexValue772, Links: rhel_vexValue781, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:L/A:N", NormalizedSeverity: 2, Package: &rhel_vexValue149, Repo: &rhel_vexValue9, Self: rhel_vexValue774, Aliases: []claircore.Alias{rhel_vexValue775}},
		{Updater: "rhel-vex", Name: "CVE-2023-38546", Description: rhel_vexValue772, Links: rhel_vexValue781, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:L/A:N", NormalizedSeverity: 2, Package: &rhel_vexValue149, Repo: &rhel_vexValue64, Self: rhel_vexValue774, Aliases: []claircore.Alias{rhel_vexValue775}},
		{Updater: "rhel-vex", Name: "CVE-2020-7774", Description: rhel_vexValue782, Links: rhel_vexValue783, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:L", NormalizedSeverity: 3, Package: &rhel_vexValue14, Repo: &rhel_vexValue20, FixedInVersion: "0:12.19.1-1.module+el8.3.0+8851+b7b41ca0", Self: rhel_vexValue784, Aliases: []claircore.Alias{rhel_vexValue785}},
		{Updater: "rhel-vex", Name: "CVE-2020-7774", Description: rhel_vexValue782, Links: rhel_vexValue786, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:L", NormalizedSeverity: 3, Package: &rhel_vexValue14, Repo: &rhel_vexValue20, FixedInVersion: "0:10.23.1-1.module+el8.3.0+9502+012d8a97", Self: rhel_vexValue784, Aliases: []claircore.Alias{rhel_vexValue785}},
		{Updater: "rhel-vex", Name: "CVE-2020-7774", Description: rhel_vexValue782, Links: rhel_vexValue787, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:L", NormalizedSeverity: 3, Package: &rhel_vexValue14, Repo: &rhel_vexValue20, FixedInVersion: "0:14.15.4-2.module+el8.3.0+9635+ffdf8381", Self: rhel_vexValue784, Aliases: []claircore.Alias{rhel_vexValue785}},
		{Updater: "rhel-vex", Name: "CVE-2020-7774", Description: rhel_vexValue782, Links: rhel_vexValue788, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:L", NormalizedSeverity: 2, Package: &rhel_vexValue26, Repo: &rhel_vexValue28, Self: rhel_vexValue784, Aliases: []claircore.Alias{rhel_vexValue785}},
		{Updater: "rhel-vex", Name: "CVE-2020-7774", Description: rhel_vexValue782, Links: rhel_vexValue788, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:L", NormalizedSeverity: 2, Package: &rhel_vexValue29, Repo: &rhel_vexValue28, Self: rhel_vexValue784, Aliases: []claircore.Alias{rhel_vexValue785}},
		{Updater: "rhel-vex", Name: "CVE-2020-7774", Description: rhel_vexValue782, Links: rhel_vexValue788, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:L", NormalizedSeverity: 2, Package: &rhel_vexValue30, Repo: &rhel_vexValue28, Self: rhel_vexValue784, Aliases: []claircore.Alias{rhel_vexValue785}},
		{Updater: "rhel-vex", Name: "CVE-2020-7754", Description: rhel_vexValue797, Links: rhel_vexValue798, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue14, Repo: &rhel_vexValue20, FixedInVersion: "0:10.23.1-1.module+el8.3.0+9502+012d8a97", Self: rhel_vexValue799, Aliases: []claircore.Alias{rhel_vexValue800}},
		{Updater: "rhel-vex", Name: "CVE-2020-7754", Description: rhel_vexValue797, Links: rhel_vexValue801, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue14, Repo: &rhel_vexValue20, FixedInVersion: "0:12.20.1-1.module+el8.3.0+9503+19cb079c", Self: rhel_vexValue799, Aliases: []claircore.Alias{rhel_vexValue800}},
		{Updater: "rhel-vex", Name: "CVE-2020-7754", Description: rhel_vexValue797, Links: rhel_vexValue802, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue14, Repo: &rhel_vexValue20, FixedInVersion: "0:14.15.4-2.module+el8.3.0+9635+ffdf8381", Self: rhel_vexValue799, Aliases: []claircore.Alias{rhel_vexValue800}},
		{Updater: "rhel-vex", Name: "CVE-2020-7754", Description: rhel_vexValue797, Links: rhel_vexValue803, Severity: "CVSS:3.0/AV:N/AC:L/PR:N/UI:R/S:U/C:N/I:N/A:L", NormalizedSeverity: 2, Package: &rhel_vexValue26, Repo: &rhel_vexValue28, Self: rhel_vexValue799, Aliases: []claircore.Alias{rhel_vexValue800}},
		{Updater: "rhel-vex", Name: "CVE-2020-7754", Description: rhel_vexValue797, Links: rhel_vexValue803, Severity: "CVSS:3.0/AV:N/AC:L/PR:N/UI:R/S:U/C:N/I:N/A:L", NormalizedSeverity: 2, Package: &rhel_vexValue29, Repo: &rhel_vexValue28, Self: rhel_vexValue799, Aliases: []claircore.Alias{rhel_vexValue800}},
		{Updater: "rhel-vex", Name: "CVE-2020-7754", Description: rhel_vexValue797, Links: rhel_vexValue803, Severity: "CVSS:3.0/AV:N/AC:L/PR:N/UI:R/S:U/C:N/I:N/A:L", NormalizedSeverity: 2, Package: &rhel_vexValue30, Repo: &rhel_vexValue28, Self: rhel_vexValue799, Aliases: []claircore.Alias{rhel_vexValue800}},
		{Updater: "rhel-vex", Name: "CVE-2019-2201", Description: rhel_vexValue804, Links: rhel_vexValue805, Severity: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue744, Repo: &rhel_vexValue9, Self: rhel_vexValue806, Aliases: []claircore.Alias{rhel_vexValue807}},
		{Updater: "rhel-vex", Name: "CVE-2019-2201", Description: rhel_vexValue804, Links: rhel_vexValue805, Severity: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue744, Repo: &rhel_vexValue64, Self: rhel_vexValue806, Aliases: []claircore.Alias{rhel_vexValue807}},
		{Updater: "rhel-vex", Name: "CVE-2019-2201", Description: rhel_vexValue804, Links: rhel_vexValue805, Severity: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue744, Repo: &rhel_vexValue28, Self: rhel_vexValue806, Aliases: []claircore.Alias{rhel_vexValue807}},
		{Updater: "rhel-vex", Name: "CVE-2025-11468", Description: rhel_vexValue808, Links: rhel_vexValue809, Severity: "CVSS:3.1/AV:N/AC:L/PR:H/UI:R/S:U/C:N/I:H/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue810, Repo: &rhel_vexValue813, FixedInVersion: "0:3.14.4-1.hum1", Self: rhel_vexValue814, Aliases: []claircore.Alias{rhel_vexValue815}},
		{Updater: "rhel-vex", Name: "CVE-2025-11468", Description: rhel_vexValue808, Links: rhel_vexValue816, Severity: "CVSS:3.1/AV:N/AC:L/PR:H/UI:R/S:U/C:N/I:H/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue817, Repo: &rhel_vexValue821, Self: rhel_vexValue814, Aliases: []claircore.Alias{rhel_vexValue815}},
		{Updater: "rhel-vex", Name: "CVE-2025-11468", Description: rhel_vexValue808, Links: rhel_vexValue816, Severity: "CVSS:3.1/AV:N/AC:L/PR:H/UI:R/S:U/C:N/I:H/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue817, Repo: &rhel_vexValue824, Self: rhel_vexValue814, Aliases: []claircore.Alias{rhel_vexValue815}},
		{Updater: "rhel-vex", Name: "CVE-2025-11468", Description: rhel_vexValue808, Links: rhel_vexValue816, Severity: "CVSS:3.1/AV:N/AC:L/PR:H/UI:R/S:U/C:N/I:H/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue817, Repo: &rhel_vexValue827, Self: rhel_vexValue814, Aliases: []claircore.Alias{rhel_vexValue815}},
		{Updater: "rhel-vex", Name: "CVE-2025-11468", Description: rhel_vexValue808, Links: rhel_vexValue816, Severity: "CVSS:3.1/AV:N/AC:L/PR:H/UI:R/S:U/C:N/I:H/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue817, Repo: &rhel_vexValue64, Self: rhel_vexValue814, Aliases: []claircore.Alias{rhel_vexValue815}},
		{Updater: "rhel-vex", Name: "CVE-2025-11468", Description: rhel_vexValue808, Links: rhel_vexValue816, Severity: "CVSS:3.1/AV:N/AC:L/PR:H/UI:R/S:U/C:N/I:H/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue817, Repo: &rhel_vexValue760, Self: rhel_vexValue814, Aliases: []claircore.Alias{rhel_vexValue815}},
		{Updater: "rhel-vex", Name: "CVE-2025-11468", Description: rhel_vexValue808, Links: rhel_vexValue816, Severity: "CVSS:3.1/AV:N/AC:L/PR:H/UI:R/S:U/C:N/I:H/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue817, Repo: &rhel_vexValue131, Self: rhel_vexValue814, Aliases: []claircore.Alias{rhel_vexValue815}},
		{Updater: "rhel-vex", Name: "CVE-2025-11468", Description: rhel_vexValue808, Links: rhel_vexValue816, Severity: "CVSS:3.1/AV:N/AC:L/PR:H/UI:R/S:U/C:N/I:H/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue817, Repo: &rhel_vexValue532, Self: rhel_vexValue814, Aliases: []claircore.Alias{rhel_vexValue815}},
		{Updater: "rhel-vex", Name: "CVE-2025-11468", Description: rhel_vexValue808, Links: rhel_vexValue816, Severity: "CVSS:3.1/AV:N/AC:L/PR:H/UI:R/S:U/C:N/I:H/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue817, Repo: &rhel_vexValue763, Self: rhel_vexValue814, Aliases: []claircore.Alias{rhel_vexValue815}},
		{Updater: "rhel-vex", Name: "CVE-2025-11468", Description: rhel_vexValue808, Links: rhel_vexValue816, Severity: "CVSS:3.1/AV:N/AC:L/PR:H/UI:R/S:U/C:N/I:H/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue817, Repo: &rhel_vexValue765, Self: rhel_vexValue814, Aliases: []claircore.Alias{rhel_vexValue815}},
		{Updater: "rhel-vex", Name: "CVE-2025-11468", Description: rhel_vexValue808, Links: rhel_vexValue816, Severity: "CVSS:3.1/AV:N/AC:L/PR:H/UI:R/S:U/C:N/I:H/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue817, Repo: &rhel_vexValue829, Self: rhel_vexValue814, Aliases: []claircore.Alias{rhel_vexValue815}},
		{Updater: "rhel-vex", Name: "CVE-2025-11468", Description: rhel_vexValue808, Links: rhel_vexValue816, Severity: "CVSS:3.1/AV:N/AC:L/PR:H/UI:R/S:U/C:N/I:H/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue817, Repo: &rhel_vexValue831, Self: rhel_vexValue814, Aliases: []claircore.Alias{rhel_vexValue815}},
		{Updater: "rhel-vex", Name: "CVE-2025-11468", Description: rhel_vexValue808, Links: rhel_vexValue816, Severity: "CVSS:3.1/AV:N/AC:L/PR:H/UI:R/S:U/C:N/I:H/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue817, Repo: &rhel_vexValue834, Self: rhel_vexValue814, Aliases: []claircore.Alias{rhel_vexValue815}},
		{Updater: "rhel-vex", Name: "CVE-2025-11468", Description: rhel_vexValue808, Links: rhel_vexValue816, Severity: "CVSS:3.1/AV:N/AC:L/PR:H/UI:R/S:U/C:N/I:H/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue817, Repo: &rhel_vexValue837, Self: rhel_vexValue814, Aliases: []claircore.Alias{rhel_vexValue815}},
		{Updater: "rhel-vex", Name: "CVE-2025-11468", Description: rhel_vexValue808, Links: rhel_vexValue816, Severity: "CVSS:3.1/AV:N/AC:L/PR:H/UI:R/S:U/C:N/I:H/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue817, Repo: &rhel_vexValue135, Self: rhel_vexValue814, Aliases: []claircore.Alias{rhel_vexValue815}},
		{Updater: "rhel-vex", Name: "CVE-2022-22978", Description: rhel_vexValue838, Links: rhel_vexValue839, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue227, Repo: &rhel_vexValue238, FixedInVersion: "0:2.387.3.1684911776-3.el8", Self: rhel_vexValue840, Aliases: []claircore.Alias{rhel_vexValue841}},
		{Updater: "rhel-vex", Name: "CVE-2022-22978", Description: rhel_vexValue838, Links: rhel_vexValue842, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue843, Repo: &rhel_vexValue845, Self: rhel_vexValue840, Aliases: []claircore.Alias{rhel_vexValue841}},
		{Updater: "rhel-vex", Name: "CVE-2022-22978", Description: rhel_vexValue838, Links: rhel_vexValue842, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue843, Repo: &rhel_vexValue846, Self: rhel_vexValue840, Aliases: []claircore.Alias{rhel_vexValue841}},
		{Updater: "rhel-vex", Name: "CVE-2023-48795", Description: rhel_vexValue847, Links: rhel_vexValue848, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:H/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue227, Repo: &rhel_vexValue850, FixedInVersion: "0:2.440.3.1716387933-3.el8", Self: rhel_vexValue851, Aliases: []claircore.Alias{rhel_vexValue852}},
		{Updater: "rhel-vex", Name: "CVE-2023-48795", Description: rhel_vexValue847, Links: rhel_vexValue848, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:H/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue234, Repo: &rhel_vexValue850, FixedInVersion: "0:4.14.1716388016-1.el8", Self: rhel_vexValue851, Aliases: []claircore.Alias{rhel_vexValue852}},
		{Updater: "rhel-vex", Name: "CVE-2023-48795", Description: rhel_vexValue847, Links: rhel_vexValue853, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:H/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue227, Repo: &rhel_vexValue242, FixedInVersion: "0:2.440.3.1716445200-3.el8", Self: rhel_vexValue851, Aliases: []claircore.Alias{rhel_vexValue852}},
		{Updater: "rhel-vex", Name: "CVE-2023-48795", Description: rhel_vexValue847, Links: rhel_vexValue853, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:H/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue234, Repo: &rhel_vexValue242, FixedInVersion: "0:4.12.1716445211-1.el8", Self: rhel_vexValue851, Aliases: []claircore.Alias{rhel_vexValue852}},
		{Updater: "rhel-vex", Name: "CVE-2023-48795", Description: rhel_vexValue847, Links: rhel_vexValue854, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:H/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue227, Repo: &rhel_vexValue238, FixedInVersion: "0:2.440.3.1716445150-3.el8", Self: rhel_vexValue851, Aliases: []claircore.Alias{rhel_vexValue852}},
		{Updater: "rhel-vex", Name: "CVE-2023-48795", Description: rhel_vexValue847, Links: rhel_vexValue854, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:H/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue234, Repo: &rhel_vexValue238, FixedInVersion: "0:4.13.1716445207-1.el8", Self: rhel_vexValue851, Aliases: []claircore.Alias{rhel_vexValue852}},
		{Updater: "rhel-vex", Name: "CVE-2023-48795", Description: rhel_vexValue847, Links: rhel_vexValue855, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:H/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue227, Repo: &rhel_vexValue857, FixedInVersion: "0:2.440.3.1718879390-3.el8", Self: rhel_vexValue851, Aliases: []claircore.Alias{rhel_vexValue852}},
		{Updater: "rhel-vex", Name: "CVE-2023-48795", Description: rhel_vexValue847, Links: rhel_vexValue855, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:H/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue234, Repo: &rhel_vexValue857, FixedInVersion: "0:4.15.1718879538-1.el8", Self: rhel_vexValue851, Aliases: []claircore.Alias{rhel_vexValue852}},
		{Updater: "rhel-vex", Name: "CVE-2023-48795", Description: rhel_vexValue847, Links: rhel_vexValue858, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:H/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue843, Repo: &rhel_vexValue859, Range: &rhel_vexValue336, Self: rhel_vexValue851, Aliases: []claircore.Alias{rhel_vexValue852}},
		{Updater: "rhel-vex", Name: "CVE-2021-33938", Description: rhel_vexValue898, Links: rhel_vexValue899, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue33, Repo: &rhel_vexValue38, FixedInVersion: "0:0.7.16-3.el8_4", Self: rhel_vexValue900, Aliases: []claircore.Alias{rhel_vexValue901}},
		{Updater: "rhel-vex", Name: "CVE-2021-33938", Description: rhel_vexValue898, Links: rhel_vexValue899, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue33, Repo: &rhel_vexValue43, Self: rhel_vexValue900, Aliases: []claircore.Alias{rhel_vexValue901}},
		{Updater: "rhel-vex", Name: "CVE-2021-33938", Description: rhel_vexValue898, Links: rhel_vexValue899, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue33, Repo: &rhel_vexValue45, FixedInVersion: "0:0.7.16-3.el8_4", Self: rhel_vexValue900, Aliases: []claircore.Alias{rhel_vexValue901}},
		{Updater: "rhel-vex", Name: "CVE-2021-33938", Description: rhel_vexValue898, Links: rhel_vexValue899, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue33, Repo: &rhel_vexValue43, FixedInVersion: "0:0.7.16-3.el8_4", Self: rhel_vexValue900, Aliases: []claircore.Alias{rhel_vexValue901}},
		{Updater: "rhel-vex", Name: "CVE-2021-33938", Description: rhel_vexValue898, Links: rhel_vexValue902, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue33, Repo: &rhel_vexValue51, FixedInVersion: "0:0.7.22-1.el7pc", Self: rhel_vexValue900, Aliases: []claircore.Alias{rhel_vexValue901}},
		{Updater: "rhel-vex", Name: "CVE-2021-33938", Description: rhel_vexValue898, Links: rhel_vexValue902, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue33, Repo: &rhel_vexValue54, FixedInVersion: "0:0.7.22-1.el8pc", Self: rhel_vexValue900, Aliases: []claircore.Alias{rhel_vexValue901}},
		{Updater: "rhel-vex", Name: "CVE-2021-33938", Description: rhel_vexValue898, Links: rhel_vexValue902, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue33, Repo: &rhel_vexValue57, FixedInVersion: "0:0.7.22-1.el7pc", Self: rhel_vexValue900, Aliases: []claircore.Alias{rhel_vexValue901}},
		{Updater: "rhel-vex", Name: "CVE-2021-33938", Description: rhel_vexValue898, Links: rhel_vexValue902, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue33, Repo: &rhel_vexValue59, FixedInVersion: "0:0.7.22-1.el8pc", Self: rhel_vexValue900, Aliases: []claircore.Alias{rhel_vexValue901}},
		{Updater: "rhel-vex", Name: "CVE-2021-33938", Description: rhel_vexValue898, Links: rhel_vexValue903, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue61, Repo: &rhel_vexValue64, Self: rhel_vexValue900, Aliases: []claircore.Alias{rhel_vexValue901}},
		{Updater: "rhel-vex", Name: "CVE-2021-33938", Description: rhel_vexValue898, Links: rhel_vexValue903, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue61, Repo: &rhel_vexValue28, Self: rhel_vexValue900, Aliases: []claircore.Alias{rhel_vexValue901}},
		{Updater: "rhel-vex", Name: "CVE-2021-33938", Description: rhel_vexValue898, Links: rhel_vexValue903, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue61, Repo: &rhel_vexValue66, Self: rhel_vexValue900, Aliases: []claircore.Alias{rhel_vexValue901}},
		{Updater: "rhel-vex", Name: "CVE-2021-33938", Description: rhel_vexValue898, Links: rhel_vexValue903, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue61, Repo: &rhel_vexValue70, Self: rhel_vexValue900, Aliases: []claircore.Alias{rhel_vexValue901}},
		{Updater: "rhel-vex", Name: "CVE-2025-15467", Description: rhel_vexValue904, Links: rhel_vexValue905, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue92, Repo: &rhel_vexValue824, FixedInVersion: "1:3.5.1-7.el10_1", Self: rhel_vexValue906, Aliases: []claircore.Alias{rhel_vexValue907}},
		{Updater: "rhel-vex", Name: "CVE-2025-15467", Description: rhel_vexValue904, Links: rhel_vexValue905, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue86, Repo: &rhel_vexValue824, FixedInVersion: "1:3.5.1-7.el10_1", Self: rhel_vexValue906, Aliases: []claircore.Alias{rhel_vexValue907}},
		{Updater: "rhel-vex", Name: "CVE-2025-15467", Description: rhel_vexValue904, Links: rhel_vexValue908, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue92, Repo: &rhel_vexValue285, FixedInVersion: "1:3.5.1-7.el9_7", Self: rhel_vexValue906, Aliases: []claircore.Alias{rhel_vexValue907}},
		{Updater: "rhel-vex", Name: "CVE-2025-15467", Description: rhel_vexValue904, Links: rhel_vexValue908, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue86, Repo: &rhel_vexValue285, FixedInVersion: "1:3.5.1-7.el9_7", Self: rhel_vexValue906, Aliases: []claircore.Alias{rhel_vexValue907}},
		{Updater: "rhel-vex", Name: "CVE-2025-15467", Description: rhel_vexValue904, Links: rhel_vexValue908, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue86, Repo: &rhel_vexValue288, FixedInVersion: "1:3.5.1-7.el9_7", Self: rhel_vexValue906, Aliases: []claircore.Alias{rhel_vexValue907}},
		{Updater: "rhel-vex", Name: "CVE-2025-15467", Description: rhel_vexValue904, Links: rhel_vexValue909, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue86, Repo: &rhel_vexValue821, FixedInVersion: "1:3.2.2-16.el10_0.6", Self: rhel_vexValue906, Aliases: []claircore.Alias{rhel_vexValue907}},
		{Updater: "rhel-vex", Name: "CVE-2025-15467", Description: rhel_vexValue904, Links: rhel_vexValue909, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue92, Repo: &rhel_vexValue821, FixedInVersion: "1:3.2.2-16.el10_0.6", Self: rhel_vexValue906, Aliases: []claircore.Alias{rhel_vexValue907}},
		{Updater: "rhel-vex", Name: "CVE-2025-15467", Description: rhel_vexValue904, Links: rhel_vexValue910, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue92, Repo: &rhel_vexValue912, Self: rhel_vexValue906, Aliases: []claircore.Alias{rhel_vexValue907}},
		{Updater: "rhel-vex", Name: "CVE-2025-15467", Description: rhel_vexValue904, Links: rhel_vexValue910, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue86, Repo: &rhel_vexValue912, Self: rhel_vexValue906, Aliases: []claircore.Alias{rhel_vexValue907}},
		{Updater: "rhel-vex", Name: "CVE-2025-15467", Description: rhel_vexValue904, Links: rhel_vexValue910, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue86, Repo: &rhel_vexValue914, FixedInVersion: "1:3.2.2-7.el9_6.2", Self: rhel_vexValue906, Aliases: []claircore.Alias{rhel_vexValue907}},
		{Updater: "rhel-vex", Name: "CVE-2025-15467", Description: rhel_vexValue904, Links: rhel_vexValue910, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue92, Repo: &rhel_vexValue914, FixedInVersion: "1:3.2.2-7.el9_6.2", Self: rhel_vexValue906, Aliases: []claircore.Alias{rhel_vexValue907}},
		{Updater: "rhel-vex", Name: "CVE-2025-15467", Description: rhel_vexValue904, Links: rhel_vexValue915, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue92, Repo: &rhel_vexValue917, FixedInVersion: "1:3.0.7-29.el9_4.2", Self: rhel_vexValue906, Aliases: []claircore.Alias{rhel_vexValue907}},
		{Updater: "rhel-vex", Name: "CVE-2025-15467", Description: rhel_vexValue904, Links: rhel_vexValue915, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue86, Repo: &rhel_vexValue917, FixedInVersion: "1:3.0.7-29.el9_4.2", Self: rhel_vexValue906, Aliases: []claircore.Alias{rhel_vexValue907}},
		{Updater: "rhel-vex", Name: "CVE-2025-15467", Description: rhel_vexValue904, Links: rhel_vexValue915, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue92, Repo: &rhel_vexValue919, Self: rhel_vexValue906, Aliases: []claircore.Alias{rhel_vexValue907}},
		{Updater: "rhel-vex", Name: "CVE-2025-15467", Description: rhel_vexValue904, Links: rhel_vexValue915, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue86, Repo: &rhel_vexValue919, Self: rhel_vexValue906, Aliases: []claircore.Alias{rhel_vexValue907}},
		{Updater: "rhel-vex", Name: "CVE-2025-15467", Description: rhel_vexValue904, Links: rhel_vexValue915, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue86, Repo: &rhel_vexValue919, FixedInVersion: "1:3.0.7-29.el9_4.2", Self: rhel_vexValue906, Aliases: []claircore.Alias{rhel_vexValue907}},
		{Updater: "rhel-vex", Name: "CVE-2025-15467", Description: rhel_vexValue904, Links: rhel_vexValue920, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue92, Repo: &rhel_vexValue922, FixedInVersion: "1:3.0.7-18.el9_2.3", Self: rhel_vexValue906, Aliases: []claircore.Alias{rhel_vexValue907}},
		{Updater: "rhel-vex", Name: "CVE-2025-15467", Description: rhel_vexValue904, Links: rhel_vexValue920, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue86, Repo: &rhel_vexValue924, FixedInVersion: "1:3.0.7-18.el9_2.3", Self: rhel_vexValue906, Aliases: []claircore.Alias{rhel_vexValue907}},
		{Updater: "rhel-vex", Name: "CVE-2025-15467", Description: rhel_vexValue904, Links: rhel_vexValue920, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue86, Repo: &rhel_vexValue922, FixedInVersion: "1:3.0.7-18.el9_2.3", Self: rhel_vexValue906, Aliases: []claircore.Alias{rhel_vexValue907}},
		{Updater: "rhel-vex", Name: "CVE-2025-15467", Description: rhel_vexValue904, Links: rhel_vexValue925, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue92, Repo: &rhel_vexValue927, FixedInVersion: "1:3.0.1-46.el9_0.7", Self: rhel_vexValue906, Aliases: []claircore.Alias{rhel_vexValue907}},
		{Updater: "rhel-vex", Name: "CVE-2025-15467", Description: rhel_vexValue904, Links: rhel_vexValue925, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue86, Repo: &rhel_vexValue927, FixedInVersion: "1:3.0.1-46.el9_0.7", Self: rhel_vexValue906, Aliases: []claircore.Alias{rhel_vexValue907}},
		{Updater: "rhel-vex", Name: "CVE-2025-15467", Description: rhel_vexValue904, Links: rhel_vexValue925, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue86, Repo: &rhel_vexValue929, Self: rhel_vexValue906, Aliases: []claircore.Alias{rhel_vexValue907}},
		{Updater: "rhel-vex", Name: "CVE-2025-15467", Description: rhel_vexValue904, Links: rhel_vexValue925, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue92, Repo: &rhel_vexValue929, Self: rhel_vexValue906, Aliases: []claircore.Alias{rhel_vexValue907}},
		{Updater: "rhel-vex", Name: "CVE-2025-15467", Description: rhel_vexValue904, Links: rhel_vexValue930, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue92, Repo: &rhel_vexValue813, FixedInVersion: "0:3.5.6-0.1.hum1", Self: rhel_vexValue906, Aliases: []claircore.Alias{rhel_vexValue907}},
		{Updater: "rhel-vex", Name: "CVE-2025-15467", Description: rhel_vexValue904, Links: rhel_vexValue930, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 4, Package: &rhel_vexValue86, Repo: &rhel_vexValue813, FixedInVersion: "0:3.5.6-0.1.hum1", Self: rhel_vexValue906, Aliases: []claircore.Alias{rhel_vexValue907}},
		{Updater: "rhel-vex", Name: "CVE-2022-34176", Description: rhel_vexValue931, Links: rhel_vexValue932, Severity: "CVSS:3.1/AV:N/AC:L/PR:L/UI:R/S:C/C:L/I:L/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue227, Repo: &rhel_vexValue621, Self: rhel_vexValue933, Aliases: []claircore.Alias{rhel_vexValue934}},
		{Updater: "rhel-vex", Name: "CVE-2022-34176", Description: rhel_vexValue931, Links: rhel_vexValue932, Severity: "CVSS:3.1/AV:N/AC:L/PR:L/UI:R/S:C/C:L/I:L/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue234, Repo: &rhel_vexValue621, FixedInVersion: "0:4.9.1667460322-1.el8", Self: rhel_vexValue933, Aliases: []claircore.Alias{rhel_vexValue934}},
		{Updater: "rhel-vex", Name: "CVE-2022-34176", Description: rhel_vexValue931, Links: rhel_vexValue935, Severity: "CVSS:3.1/AV:N/AC:L/PR:L/UI:R/S:C/C:L/I:L/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue227, Repo: &rhel_vexValue616, Self: rhel_vexValue933, Aliases: []claircore.Alias{rhel_vexValue934}},
		{Updater: "rhel-vex", Name: "CVE-2022-34176", Description: rhel_vexValue931, Links: rhel_vexValue935, Severity: "CVSS:3.1/AV:N/AC:L/PR:L/UI:R/S:C/C:L/I:L/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue234, Repo: &rhel_vexValue616, FixedInVersion: "0:4.10.1663147786-1.el8", Self: rhel_vexValue933, Aliases: []claircore.Alias{rhel_vexValue934}},
		{Updater: "rhel-vex", Name: "CVE-2022-34176", Description: rhel_vexValue931, Links: rhel_vexValue936, Severity: "CVSS:3.1/AV:N/AC:L/PR:L/UI:R/S:C/C:L/I:L/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue227, Repo: &rhel_vexValue624, Self: rhel_vexValue933, Aliases: []claircore.Alias{rhel_vexValue934}},
		{Updater: "rhel-vex", Name: "CVE-2022-34176", Description: rhel_vexValue931, Links: rhel_vexValue936, Severity: "CVSS:3.1/AV:N/AC:L/PR:L/UI:R/S:C/C:L/I:L/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue234, Repo: &rhel_vexValue624, FixedInVersion: "0:4.8.1672842762-1.el8", Self: rhel_vexValue933, Aliases: []claircore.Alias{rhel_vexValue934}},
		{Updater: "rhel-vex", Name: "CVE-2022-34176", Description: rhel_vexValue931, Links: rhel_vexValue937, Severity: "CVSS:3.1/AV:N/AC:L/PR:L/UI:R/S:C/C:L/I:L/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue244, Repo: &rhel_vexValue254, Self: rhel_vexValue933, Aliases: []claircore.Alias{rhel_vexValue934}},
		{Updater: "rhel-vex", Name: "CVE-2022-34176", Description: rhel_vexValue931, Links: rhel_vexValue937, Severity: "CVSS:3.1/AV:N/AC:L/PR:L/UI:R/S:C/C:L/I:L/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue244, Repo: &rhel_vexValue256, Self: rhel_vexValue933, Aliases: []claircore.Alias{rhel_vexValue934}},
		{Updater: "rhel-vex", Name: "CVE-2022-34176", Description: rhel_vexValue931, Links: rhel_vexValue937, Severity: "CVSS:3.1/AV:N/AC:L/PR:L/UI:R/S:C/C:L/I:L/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue244, Repo: &rhel_vexValue632, Self: rhel_vexValue933, Aliases: []claircore.Alias{rhel_vexValue934}},
		{Updater: "rhel-vex", Name: "CVE-2022-34176", Description: rhel_vexValue931, Links: rhel_vexValue937, Severity: "CVSS:3.1/AV:N/AC:L/PR:L/UI:R/S:C/C:L/I:L/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue244, Repo: &rhel_vexValue261, Self: rhel_vexValue933, Aliases: []claircore.Alias{rhel_vexValue934}},
		{Updater: "rhel-vex", Name: "CVE-2022-34176", Description: rhel_vexValue931, Links: rhel_vexValue937, Severity: "CVSS:3.1/AV:N/AC:L/PR:L/UI:R/S:C/C:L/I:L/A:N", NormalizedSeverity: 3, Package: &rhel_vexValue244, Repo: &rhel_vexValue635, Self: rhel_vexValue933, Aliases: []claircore.Alias{rhel_vexValue934}},
		{Updater: "rhel-vex", Name: "CVE-2022-1650", Description: rhel_vexValue938, Links: rhel_vexValue939, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:C/C:H/I:H/A:N", NormalizedSeverity: 4, Package: &rhel_vexValue940, Repo: &rhel_vexValue603, Self: rhel_vexValue941, Aliases: []claircore.Alias{rhel_vexValue942}},
		{Updater: "rhel-vex", Name: "CVE-2022-1650", Description: rhel_vexValue938, Links: rhel_vexValue939, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:C/C:H/I:H/A:N", NormalizedSeverity: 4, Package: &rhel_vexValue943, Repo: &rhel_vexValue603, Self: rhel_vexValue941, Aliases: []claircore.Alias{rhel_vexValue942}},
		{Updater: "rhel-vex", Name: "CVE-2022-1650", Description: rhel_vexValue938, Links: rhel_vexValue939, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:C/C:H/I:H/A:N", NormalizedSeverity: 4, Package: &rhel_vexValue944, Repo: &rhel_vexValue603, Self: rhel_vexValue941, Aliases: []claircore.Alias{rhel_vexValue942}},
		{Updater: "rhel-vex", Name: "CVE-2022-1650", Description: rhel_vexValue938, Links: rhel_vexValue939, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:C/C:H/I:H/A:N", NormalizedSeverity: 4, Package: &rhel_vexValue945, Repo: &rhel_vexValue43, FixedInVersion: "0:6.0.107-1.el8_6", Self: rhel_vexValue941, Aliases: []claircore.Alias{rhel_vexValue942}},
		{Updater: "rhel-vex", Name: "CVE-2022-1650", Description: rhel_vexValue938, Links: rhel_vexValue939, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:C/C:H/I:H/A:N", NormalizedSeverity: 4, Package: &rhel_vexValue940, Repo: &rhel_vexValue301, FixedInVersion: "0:6.0.7-1.el8_6", Self: rhel_vexValue941, Aliases: []claircore.Alias{rhel_vexValue942}},
		{Updater: "rhel-vex", Name: "CVE-2022-1650", Description: rhel_vexValue938, Links: rhel_vexValue939, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:C/C:H/I:H/A:N", NormalizedSeverity: 4, Package: &rhel_vexValue943, Repo: &rhel_vexValue301, FixedInVersion: "0:6.0.107-1.el8_6", Self: rhel_vexValue941, Aliases: []claircore.Alias{rhel_vexValue942}},
		{Updater: "rhel-vex", Name: "CVE-2022-1650", Description: rhel_vexValue938, Links: rhel_vexValue939, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:C/C:H/I:H/A:N", NormalizedSeverity: 4, Package: &rhel_vexValue944, Repo: &rhel_vexValue301, FixedInVersion: "0:6.0.7-1.el8_6", Self: rhel_vexValue941, Aliases: []claircore.Alias{rhel_vexValue942}},
		{Updater: "rhel-vex", Name: "CVE-2022-1650", Description: rhel_vexValue938, Links: rhel_vexValue939, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:C/C:H/I:H/A:N", NormalizedSeverity: 4, Package: &rhel_vexValue944, Repo: &rhel_vexValue20, FixedInVersion: "0:6.0.7-1.el8_6", Self: rhel_vexValue941, Aliases: []claircore.Alias{rhel_vexValue942}},
		{Updater: "rhel-vex", Name: "CVE-2022-1650", Description: rhel_vexValue938, Links: rhel_vexValue939, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:C/C:H/I:H/A:N", NormalizedSeverity: 4, Package: &rhel_vexValue943, Repo: &rhel_vexValue20, FixedInVersion: "0:6.0.107-1.el8_6", Self: rhel_vexValue941, Aliases: []claircore.Alias{rhel_vexValue942}},
		{Updater: "rhel-vex", Name: "CVE-2022-1650", Description: rhel_vexValue938, Links: rhel_vexValue939, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:C/C:H/I:H/A:N", NormalizedSeverity: 4, Package: &rhel_vexValue940, Repo: &rhel_vexValue20, FixedInVersion: "0:6.0.7-1.el8_6", Self: rhel_vexValue941, Aliases: []claircore.Alias{rhel_vexValue942}},
		{Updater: "rhel-vex", Name: "CVE-2022-1650", Description: rhel_vexValue938, Links: rhel_vexValue946, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:C/C:H/I:H/A:N", NormalizedSeverity: 4, Package: &rhel_vexValue940, Repo: &rhel_vexValue592, Self: rhel_vexValue941, Aliases: []claircore.Alias{rhel_vexValue942}},
		{Updater: "rhel-vex", Name: "CVE-2022-1650", Description: rhel_vexValue938, Links: rhel_vexValue946, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:C/C:H/I:H/A:N", NormalizedSeverity: 4, Package: &rhel_vexValue940, Repo: &rhel_vexValue273, FixedInVersion: "0:6.0.7-1.el9_0", Self: rhel_vexValue941, Aliases: []claircore.Alias{rhel_vexValue942}},
		{Updater: "rhel-vex", Name: "CVE-2022-1650", Description: rhel_vexValue938, Links: rhel_vexValue946, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:C/C:H/I:H/A:N", NormalizedSeverity: 4, Package: &rhel_vexValue944, Repo: &rhel_vexValue273, FixedInVersion: "0:6.0.7-1.el9_0", Self: rhel_vexValue941, Aliases: []claircore.Alias{rhel_vexValue942}},
		{Updater: "rhel-vex", Name: "CVE-2022-1650", Description: rhel_vexValue938, Links: rhel_vexValue946, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:C/C:H/I:H/A:N", NormalizedSeverity: 4, Package: &rhel_vexValue945, Repo: &rhel_vexValue557, FixedInVersion: "0:6.0.107-1.el9_0", Self: rhel_vexValue941, Aliases: []claircore.Alias{rhel_vexValue942}},
		{Updater: "rhel-vex", Name: "CVE-2022-1650", Description: rhel_vexValue938, Links: rhel_vexValue946, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:C/C:H/I:H/A:N", NormalizedSeverity: 4, Package: &rhel_vexValue944, Repo: &rhel_vexValue592, Self: rhel_vexValue941, Aliases: []claircore.Alias{rhel_vexValue942}},
		{Updater: "rhel-vex", Name: "CVE-2022-1650", Description: rhel_vexValue938, Links: rhel_vexValue947, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:C/C:H/I:H/A:N", NormalizedSeverity: 4, Package: &rhel_vexValue943, Repo: &rhel_vexValue20, Self: rhel_vexValue941, Aliases: []claircore.Alias{rhel_vexValue942}},
		{Updater: "rhel-vex", Name: "CVE-2022-1650", Description: rhel_vexValue938, Links: rhel_vexValue947, Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:C/C:H/I:H/A:N", NormalizedSeverity: 4, Package: &rhel_vexValue943, Repo: &rhel_vexValue301, Self: rhel_vexValue941, Aliases: []claircore.Alias{rhel_vexValue942}},
		{Updater: "rhel-vex", Name: "CVE-2019-11068", Description: rhel_vexValue952, Links: rhel_vexValue953, Severity: "CVSS:3.0/AV:N/AC:L/PR:N/UI:R/S:U/C:L/I:L/A:L", NormalizedSeverity: 3, Package: &rhel_vexValue954, Repo: &rhel_vexValue95, FixedInVersion: "0:1.1.28-6.el7", Self: rhel_vexValue955, Aliases: []claircore.Alias{rhel_vexValue956}},
		{Updater: "rhel-vex", Name: "CVE-2019-11068", Description: rhel_vexValue952, Links: rhel_vexValue953, Severity: "CVSS:3.0/AV:N/AC:L/PR:N/UI:R/S:U/C:L/I:L/A:L", NormalizedSeverity: 3, Package: &rhel_vexValue954, Repo: &rhel_vexValue98, FixedInVersion: "0:1.1.28-6.el7", Self: rhel_vexValue955, Aliases: []claircore.Alias{rhel_vexValue956}},
		{Updater: "rhel-vex", Name: "CVE-2019-11068", Description: rhel_vexValue952, Links: rhel_vexValue953, Severity: "CVSS:3.0/AV:N/AC:L/PR:N/UI:R/S:U/C:L/I:L/A:L", NormalizedSeverity: 3, Package: &rhel_vexValue954, Repo: &rhel_vexValue89, FixedInVersion: "0:1.1.28-6.el7", Self: rhel_vexValue955, Aliases: []claircore.Alias{rhel_vexValue956}},
		{Updater: "rhel-vex", Name: "CVE-2019-11068", Description: rhel_vexValue952, Links: rhel_vexValue953, Severity: "CVSS:3.0/AV:N/AC:L/PR:N/UI:R/S:U/C:L/I:L/A:L", NormalizedSeverity: 3, Package: &rhel_vexValue954, Repo: &rhel_vexValue101, FixedInVersion: "0:1.1.28-6.el7", Self: rhel_vexValue955, Aliases: []claircore.Alias{rhel_vexValue956}},
		{Updater: "rhel-vex", Name: "CVE-2019-11068", Description: rhel_vexValue952, Links: rhel_vexValue957, Severity: "CVSS:3.0/AV:N/AC:L/PR:N/UI:R/S:U/C:L/I:L/A:L", NormalizedSeverity: 3, Package: &rhel_vexValue954, Repo: &rhel_vexValue20, Self: rhel_vexValue955, Aliases: []claircore.Alias{rhel_vexValue956}},
		{Updater: "rhel-vex", Name: "CVE-2019-11068", Description: rhel_vexValue952, Links: rhel_vexValue957, Severity: "CVSS:3.0/AV:N/AC:L/PR:N/UI:R/S:U/C:L/I:L/A:L", NormalizedSeverity: 3, Package: &rhel_vexValue954, Repo: &rhel_vexValue20, FixedInVersion: "0:1.1.32-5.el8", Self: rhel_vexValue955, Aliases: []claircore.Alias{rhel_vexValue956}},
		{Updater: "rhel-vex", Name: "CVE-2019-11068", Description: rhel_vexValue952, Links: rhel_vexValue957, Severity: "CVSS:3.0/AV:N/AC:L/PR:N/UI:R/S:U/C:L/I:L/A:L", NormalizedSeverity: 3, Package: &rhel_vexValue954, Repo: &rhel_vexValue45, FixedInVersion: "0:1.1.32-5.el8", Self: rhel_vexValue955, Aliases: []claircore.Alias{rhel_vexValue956}},
		{Updater: "rhel-vex", Name: "CVE-2019-11068", Description: rhel_vexValue952, Links: rhel_vexValue958, Severity: "CVSS:3.0/AV:N/AC:L/PR:N/UI:R/S:U/C:L/I:L/A:L", NormalizedSeverity: 3, Package: &rhel_vexValue959, Repo: &rhel_vexValue962, Self: rhel_vexValue955, Aliases: []claircore.Alias{rhel_vexValue956}},
		{Updater: "rhel-vex", Name: "CVE-2019-11068", Description: rhel_vexValue952, Links: rhel_vexValue958, Severity: "CVSS:3.0/AV:N/AC:L/PR:N/UI:R/S:U/C:L/I:L/A:L", NormalizedSeverity: 3, Package: &rhel_vexValue959, Repo: &rhel_vexValue965, Self: rhel_vexValue955, Aliases: []claircore.Alias{rhel_vexValue956}},
		{Updater: "rhel-vex", Name: "CVE-2019-11068", Description: rhel_vexValue952, Links: rhel_vexValue958, Severity: "CVSS:3.0/AV:N/AC:L/PR:N/UI:R/S:U/C:L/I:L/A:L", NormalizedSeverity: 3, Package: &rhel_vexValue959, Repo: &rhel_vexValue968, Self: rhel_vexValue955, Aliases: []claircore.Alias{rhel_vexValue956}},
		{Updater: "rhel-vex", Name: "CVE-2019-11068", Description: rhel_vexValue952, Links: rhel_vexValue958, Severity: "CVSS:3.0/AV:N/AC:L/PR:N/UI:R/S:U/C:L/I:L/A:L", NormalizedSeverity: 3, Package: &rhel_vexValue959, Repo: &rhel_vexValue970, Self: rhel_vexValue955, Aliases: []claircore.Alias{rhel_vexValue956}},
		{Updater: "rhel-vex", Name: "CVE-2019-11068", Description: rhel_vexValue952, Links: rhel_vexValue958, Severity: "CVSS:3.0/AV:N/AC:L/PR:N/UI:R/S:U/C:L/I:L/A:L", NormalizedSeverity: 3, Package: &rhel_vexValue959, Repo: &rhel_vexValue182, Self: rhel_vexValue955, Aliases: []claircore.Alias{rhel_vexValue956}},
		{Updater: "rhel-vex", Name: "CVE-2019-11068", Description: rhel_vexValue952, Links: rhel_vexValue958, Severity: "CVSS:3.0/AV:N/AC:L/PR:N/UI:R/S:U/C:L/I:L/A:L", NormalizedSeverity: 3, Package: &rhel_vexValue959, Repo: &rhel_vexValue9, Self: rhel_vexValue955, Aliases: []claircore.Alias{rhel_vexValue956}},
		{Updater: "rhel-vex", Name: "CVE-2019-11068", Description: rhel_vexValue952, Links: rhel_vexValue958, Severity: "CVSS:3.0/AV:N/AC:L/PR:N/UI:R/S:U/C:L/I:L/A:L", NormalizedSeverity: 3, Package: &rhel_vexValue959, Repo: &rhel_vexValue64, Self: rhel_vexValue955, Aliases: []claircore.Alias{rhel_vexValue956}},
		{Updater: "rhel-vex", Name: "CVE-2019-11068", Description: rhel_vexValue952, Links: rhel_vexValue958, Severity: "CVSS:3.0/AV:N/AC:L/PR:N/UI:R/S:U/C:L/I:L/A:L", NormalizedSeverity: 3, Package: &rhel_vexValue959, Repo: &rhel_vexValue28, Self: rhel_vexValue955, Aliases: []claircore.Alias{rhel_vexValue956}},
		{Updater: "rhel-vex", Name: "CVE-2019-11068", Description: rhel_vexValue952, Links: rhel_vexValue958, Severity: "CVSS:3.0/AV:N/AC:L/PR:N/UI:R/S:U/C:L/I:L/A:L", NormalizedSeverity: 3, Package: &rhel_vexValue959, Repo: &rhel_vexValue973, Self: rhel_vexValue955, Aliases: []claircore.Alias{rhel_vexValue956}},
		{Updater: "rhel-vex", Name: "CVE-2022-1785", Description: rhel_vexValue974, Links: rhel_vexValue975, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:L/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue296, Repo: &rhel_vexValue45, FixedInVersion: "2:8.0.1763-19.el8_6.4", Self: rhel_vexValue976, Aliases: []claircore.Alias{rhel_vexValue977}},
		{Updater: "rhel-vex", Name: "CVE-2022-1785", Description: rhel_vexValue974, Links: rhel_vexValue975, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:L/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue296, Repo: &rhel_vexValue20, Self: rhel_vexValue976, Aliases: []claircore.Alias{rhel_vexValue977}},
		{Updater: "rhel-vex", Name: "CVE-2022-1785", Description: rhel_vexValue974, Links: rhel_vexValue975, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:L/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue299, Repo: &rhel_vexValue301, Self: rhel_vexValue976, Aliases: []claircore.Alias{rhel_vexValue977}},
		{Updater: "rhel-vex", Name: "CVE-2022-1785", Description: rhel_vexValue974, Links: rhel_vexValue975, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:L/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue296, Repo: &rhel_vexValue301, Self: rhel_vexValue976, Aliases: []claircore.Alias{rhel_vexValue977}},
		{Updater: "rhel-vex", Name: "CVE-2022-1785", Description: rhel_vexValue974, Links: rhel_vexValue975, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:L/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue299, Repo: &rhel_vexValue220, Self: rhel_vexValue976, Aliases: []claircore.Alias{rhel_vexValue977}},
		{Updater: "rhel-vex", Name: "CVE-2022-1785", Description: rhel_vexValue974, Links: rhel_vexValue975, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:L/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue296, Repo: &rhel_vexValue220, FixedInVersion: "2:8.0.1763-19.el8_6.4", Self: rhel_vexValue976, Aliases: []claircore.Alias{rhel_vexValue977}},
		{Updater: "rhel-vex", Name: "CVE-2022-1785", Description: rhel_vexValue974, Links: rhel_vexValue975, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:L/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue296, Repo: &rhel_vexValue320, FixedInVersion: "2:8.0.1763-19.el8_6.4", Self: rhel_vexValue976, Aliases: []claircore.Alias{rhel_vexValue977}},
		{Updater: "rhel-vex", Name: "CVE-2022-1785", Description: rhel_vexValue974, Links: rhel_vexValue975, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:L/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue299, Repo: &rhel_vexValue20, Self: rhel_vexValue976, Aliases: []claircore.Alias{rhel_vexValue977}},
		{Updater: "rhel-vex", Name: "CVE-2022-1785", Description: rhel_vexValue974, Links: rhel_vexValue978, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:L/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue296, Repo: &rhel_vexValue277, FixedInVersion: "2:8.2.2637-16.el9_0.3", Self: rhel_vexValue976, Aliases: []claircore.Alias{rhel_vexValue977}},
		{Updater: "rhel-vex", Name: "CVE-2022-1785", Description: rhel_vexValue974, Links: rhel_vexValue978, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:L/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue296, Repo: &rhel_vexValue285, FixedInVersion: "2:8.2.2637-16.el9_0.3", Self: rhel_vexValue976, Aliases: []claircore.Alias{rhel_vexValue977}},
		{Updater: "rhel-vex", Name: "CVE-2022-1785", Description: rhel_vexValue974, Links: rhel_vexValue978, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:L/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue296, Repo: &rhel_vexValue273, Self: rhel_vexValue976, Aliases: []claircore.Alias{rhel_vexValue977}},
		{Updater: "rhel-vex", Name: "CVE-2022-1785", Description: rhel_vexValue974, Links: rhel_vexValue979, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:L/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue304, Repo: &rhel_vexValue28, Self: rhel_vexValue976, Aliases: []claircore.Alias{rhel_vexValue977}},
		{Updater: "rhel-vex", Name: "CVE-2022-1785", Description: rhel_vexValue974, Links: rhel_vexValue979, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:L/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue305, Repo: &rhel_vexValue28, Self: rhel_vexValue976, Aliases: []claircore.Alias{rhel_vexValue977}},
		{Updater: "rhel-vex", Name: "CVE-2022-1785", Description: rhel_vexValue974, Links: rhel_vexValue979, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:L/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue304, Repo: &rhel_vexValue135, Self: rhel_vexValue976, Aliases: []claircore.Alias{rhel_vexValue977}},
		{Updater: "rhel-vex", Name: "CVE-2022-1785", Description: rhel_vexValue974, Links: rhel_vexValue979, Severity: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:L/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue305, Repo: &rhel_vexValue135, Self: rhel_vexValue976, Aliases: []claircore.Alias{rhel_vexValue977}},
		{Updater: "rhel-vex", Name: "CVE-2021-22884", Description: rhel_vexValue980, Links: rhel_vexValue981, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:R/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue14, Repo: &rhel_vexValue20, FixedInVersion: "0:12.21.0-1.module+el8.3.0+10191+34fb5a07", Self: rhel_vexValue982, Aliases: []claircore.Alias{rhel_vexValue983}},
		{Updater: "rhel-vex", Name: "CVE-2021-22884", Description: rhel_vexValue980, Links: rhel_vexValue984, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:R/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue14, Repo: &rhel_vexValue20, FixedInVersion: "0:10.24.0-1.module+el8.3.0+10166+b07ac28e", Self: rhel_vexValue982, Aliases: []claircore.Alias{rhel_vexValue983}},
		{Updater: "rhel-vex", Name: "CVE-2021-22884", Description: rhel_vexValue980, Links: rhel_vexValue985, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:R/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue14, Repo: &rhel_vexValue120, FixedInVersion: "0:10.24.0-1.module+el8.2.0+10165+019e8570", Self: rhel_vexValue982, Aliases: []claircore.Alias{rhel_vexValue983}},
		{Updater: "rhel-vex", Name: "CVE-2021-22884", Description: rhel_vexValue980, Links: rhel_vexValue986, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:R/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue14, Repo: &rhel_vexValue124, FixedInVersion: "0:12.21.0-1.module+el8.1.0+10194+d5e49c90", Self: rhel_vexValue982, Aliases: []claircore.Alias{rhel_vexValue983}},
		{Updater: "rhel-vex", Name: "CVE-2021-22884", Description: rhel_vexValue980, Links: rhel_vexValue987, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:R/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue14, Repo: &rhel_vexValue120, FixedInVersion: "0:12.21.0-1.module+el8.2.0+10192+8959c43b", Self: rhel_vexValue982, Aliases: []claircore.Alias{rhel_vexValue983}},
		{Updater: "rhel-vex", Name: "CVE-2021-22884", Description: rhel_vexValue980, Links: rhel_vexValue988, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:R/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue14, Repo: &rhel_vexValue124, FixedInVersion: "0:10.24.0-1.module+el8.1.0+10161+5cffdac6", Self: rhel_vexValue982, Aliases: []claircore.Alias{rhel_vexValue983}},
		{Updater: "rhel-vex", Name: "CVE-2021-22884", Description: rhel_vexValue980, Links: rhel_vexValue989, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:R/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue14, Repo: &rhel_vexValue20, FixedInVersion: "0:14.16.0-2.module+el8.3.0+10180+b92e1eb6", Self: rhel_vexValue982, Aliases: []claircore.Alias{rhel_vexValue983}},
		{Updater: "rhel-vex", Name: "CVE-2021-22884", Description: rhel_vexValue980, Links: rhel_vexValue990, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:R/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue26, Repo: &rhel_vexValue28, Self: rhel_vexValue982, Aliases: []claircore.Alias{rhel_vexValue983}},
		{Updater: "rhel-vex", Name: "CVE-2021-22884", Description: rhel_vexValue980, Links: rhel_vexValue990, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:R/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue29, Repo: &rhel_vexValue28, Self: rhel_vexValue982, Aliases: []claircore.Alias{rhel_vexValue983}},
		{Updater: "rhel-vex", Name: "CVE-2021-22884", Description: rhel_vexValue980, Links: rhel_vexValue990, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:R/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue26, Repo: &rhel_vexValue131, Self: rhel_vexValue982, Aliases: []claircore.Alias{rhel_vexValue983}},
		{Updater: "rhel-vex", Name: "CVE-2021-22884", Description: rhel_vexValue980, Links: rhel_vexValue990, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:R/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue29, Repo: &rhel_vexValue131, Self: rhel_vexValue982, Aliases: []claircore.Alias{rhel_vexValue983}},
		{Updater: "rhel-vex", Name: "CVE-2021-22884", Description: rhel_vexValue980, Links: rhel_vexValue990, Severity: "CVSS:3.1/AV:N/AC:H/PR:N/UI:R/S:U/C:H/I:H/A:H", NormalizedSeverity: 3, Package: &rhel_vexValue30, Repo: &rhel_vexValue28, Self: rhel_vexValue982, Aliases: []claircore.Alias{rhel_vexValue983}},
	}},
}
var rhel_vexValue0 = "A flaw was found in BusyBox. It did not properly sanitize while processing a crafted awk pattern, leading to possible code execution."

var rhel_vexValue1 = "https://access.redhat.com/security/cve/CVE-2022-30065 https://nvd.nist.gov/vuln/detail/CVE-2022-30065 https://www.cve.org/CVERecord?id=CVE-2022-30065 https://security.access.redhat.com/data/csaf/v2/vex-feed/2022/cve-2022-30065.json"

var rhel_vexValue2 = claircore.Package{Name: "busybox", Kind: 1}

var rhel_vexValue3 = cpe.Value{V: "o", Kind: 3}

var rhel_vexValue4 = cpe.Value{V: "redhat", Kind: 3}

var rhel_vexValue5 = cpe.Value{V: "rhel_els", Kind: 3}

var rhel_vexValue6 = cpe.Value{V: "6", Kind: 3}

var rhel_vexValue7 = cpe.Value{Kind: 1}

var rhel_vexValue8 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue3, rhel_vexValue4, rhel_vexValue5, rhel_vexValue6, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue9 = claircore.Repository{Name: "cpe:2.3:o:redhat:rhel_els:6:*:*:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue8}

var rhel_vexValue10 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2022-30065"}

var rhel_vexValue11 = claircore.Alias{Space: unique.Make("CVE"), Name: "2022-30065"}

var rhel_vexValue12 = "A flaw was found in nodejs-ajv. A carefully crafted JSON schema could be provided that allows execution of other code by prototype pollution. While untrusted schemas are recommended against, the worst case of an untrusted schema should be a denial of service, not execution of code."

var rhel_vexValue13 = "https://access.redhat.com/security/cve/CVE-2020-15366 https://nvd.nist.gov/vuln/detail/CVE-2020-15366 https://www.cve.org/CVERecord?id=CVE-2020-15366 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-15366.json https://access.redhat.com/errata/RHSA-2020:5499"

var rhel_vexValue14 = claircore.Package{Name: "nodejs", Kind: 2}

var rhel_vexValue15 = cpe.Value{V: "a", Kind: 3}

var rhel_vexValue16 = cpe.Value{V: "enterprise_linux", Kind: 3}

var rhel_vexValue17 = cpe.Value{V: "8", Kind: 3}

var rhel_vexValue18 = cpe.Value{V: "appstream", Kind: 3}

var rhel_vexValue19 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue16, rhel_vexValue17, rhel_vexValue7, rhel_vexValue18, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue20 = claircore.Repository{Name: "cpe:2.3:a:redhat:enterprise_linux:8:*:appstream:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue19}

var rhel_vexValue21 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2020-15366"}

var rhel_vexValue22 = claircore.Alias{Space: unique.Make("CVE"), Name: "2020-15366"}

var rhel_vexValue23 = "https://access.redhat.com/security/cve/CVE-2020-15366 https://nvd.nist.gov/vuln/detail/CVE-2020-15366 https://www.cve.org/CVERecord?id=CVE-2020-15366 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-15366.json https://access.redhat.com/errata/RHSA-2021:0548"

var rhel_vexValue24 = "https://access.redhat.com/security/cve/CVE-2020-15366 https://nvd.nist.gov/vuln/detail/CVE-2020-15366 https://www.cve.org/CVERecord?id=CVE-2020-15366 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-15366.json https://access.redhat.com/errata/RHSA-2021:0551"

var rhel_vexValue25 = "https://access.redhat.com/security/cve/CVE-2020-15366 https://nvd.nist.gov/vuln/detail/CVE-2020-15366 https://www.cve.org/CVERecord?id=CVE-2020-15366 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-15366.json"

var rhel_vexValue26 = claircore.Package{Name: "nodejs", Kind: 1, Module: "nodejs:10"}

var rhel_vexValue27 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue16, rhel_vexValue17, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue28 = claircore.Repository{Name: "cpe:2.3:a:redhat:enterprise_linux:8:*:*:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue27}

var rhel_vexValue29 = claircore.Package{Name: "nodejs", Kind: 1, Module: "nodejs:12"}

var rhel_vexValue30 = claircore.Package{Name: "nodejs", Kind: 1, Module: "nodejs:14"}

var rhel_vexValue31 = "A flaw was found in libsolv. A buffer overflow in the pool_installable function allows attackers to cause a denial of service. The highest threat from this vulnerability is to system availability."

var rhel_vexValue32 = "https://access.redhat.com/security/cve/CVE-2021-33928 https://nvd.nist.gov/vuln/detail/CVE-2021-33928 https://www.cve.org/CVERecord?id=CVE-2021-33928 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-33928.json https://access.redhat.com/errata/RHSA-2021:4060"

var rhel_vexValue33 = claircore.Package{Name: "libsolv", Kind: 2}

var rhel_vexValue34 = cpe.Value{V: "rhel_eus", Kind: 3}

var rhel_vexValue35 = cpe.Value{V: "8\\.4", Kind: 3}

var rhel_vexValue36 = cpe.Value{V: "baseos", Kind: 3}

var rhel_vexValue37 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue3, rhel_vexValue4, rhel_vexValue34, rhel_vexValue35, rhel_vexValue7, rhel_vexValue36, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue38 = claircore.Repository{Name: "cpe:2.3:o:redhat:rhel_eus:8.4:*:baseos:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue37}

var rhel_vexValue39 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2021-33928"}

var rhel_vexValue40 = claircore.Alias{Space: unique.Make("CVE"), Name: "2021-33928"}

var rhel_vexValue41 = cpe.Value{V: "crb", Kind: 3}

var rhel_vexValue42 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue16, rhel_vexValue17, rhel_vexValue7, rhel_vexValue41, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue43 = claircore.Repository{Name: "cpe:2.3:a:redhat:enterprise_linux:8:*:crb:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue42}

var rhel_vexValue44 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue3, rhel_vexValue4, rhel_vexValue16, rhel_vexValue17, rhel_vexValue7, rhel_vexValue36, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue45 = claircore.Repository{Name: "cpe:2.3:o:redhat:enterprise_linux:8:*:baseos:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue44}

var rhel_vexValue46 = "https://access.redhat.com/security/cve/CVE-2021-33928 https://nvd.nist.gov/vuln/detail/CVE-2021-33928 https://www.cve.org/CVERecord?id=CVE-2021-33928 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-33928.json https://access.redhat.com/errata/RHSA-2022:5498"

var rhel_vexValue47 = cpe.Value{V: "satellite_capsule", Kind: 3}

var rhel_vexValue48 = cpe.Value{V: "6\\.11", Kind: 3}

var rhel_vexValue49 = cpe.Value{V: "el7", Kind: 3}

var rhel_vexValue50 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue47, rhel_vexValue48, rhel_vexValue7, rhel_vexValue49, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue51 = claircore.Repository{Name: "cpe:2.3:a:redhat:satellite_capsule:6.11:*:el7:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue50}

var rhel_vexValue52 = cpe.Value{V: "el8", Kind: 3}

var rhel_vexValue53 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue47, rhel_vexValue48, rhel_vexValue7, rhel_vexValue52, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue54 = claircore.Repository{Name: "cpe:2.3:a:redhat:satellite_capsule:6.11:*:el8:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue53}

var rhel_vexValue55 = cpe.Value{V: "satellite", Kind: 3}

var rhel_vexValue56 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue55, rhel_vexValue48, rhel_vexValue7, rhel_vexValue49, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue57 = claircore.Repository{Name: "cpe:2.3:a:redhat:satellite:6.11:*:el7:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue56}

var rhel_vexValue58 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue55, rhel_vexValue48, rhel_vexValue7, rhel_vexValue52, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue59 = claircore.Repository{Name: "cpe:2.3:a:redhat:satellite:6.11:*:el8:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue58}

var rhel_vexValue60 = "https://access.redhat.com/security/cve/CVE-2021-33928 https://nvd.nist.gov/vuln/detail/CVE-2021-33928 https://www.cve.org/CVERecord?id=CVE-2021-33928 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-33928.json"

var rhel_vexValue61 = claircore.Package{Name: "libsolv", Kind: 1}

var rhel_vexValue62 = cpe.Value{V: "7", Kind: 3}

var rhel_vexValue63 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue3, rhel_vexValue4, rhel_vexValue5, rhel_vexValue62, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue64 = claircore.Repository{Name: "cpe:2.3:o:redhat:rhel_els:7:*:*:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue63}

var rhel_vexValue65 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue55, rhel_vexValue6, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue66 = claircore.Repository{Name: "cpe:2.3:a:redhat:satellite:6:*:*:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue65}

var rhel_vexValue67 = cpe.Value{V: "rhui", Kind: 3}

var rhel_vexValue68 = cpe.Value{V: "3", Kind: 3}

var rhel_vexValue69 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue67, rhel_vexValue68, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue70 = claircore.Repository{Name: "cpe:2.3:a:redhat:rhui:3:*:*:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue69}

var rhel_vexValue71 = "An out-of-bounds read was discovered in Libsolv when the last schema has a length that is less than the length of the input schema. A remote attacker may abuse this flaw to crash an application that uses Libsolv."

var rhel_vexValue72 = "https://access.redhat.com/security/cve/CVE-2019-20387 https://nvd.nist.gov/vuln/detail/CVE-2019-20387 https://www.cve.org/CVERecord?id=CVE-2019-20387 https://security.access.redhat.com/data/csaf/v2/vex-feed/2019/cve-2019-20387.json https://access.redhat.com/errata/RHSA-2020:4508"

var rhel_vexValue73 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2019-20387"}

var rhel_vexValue74 = claircore.Alias{Space: unique.Make("CVE"), Name: "2019-20387"}

var rhel_vexValue75 = "https://access.redhat.com/security/cve/CVE-2019-20387 https://nvd.nist.gov/vuln/detail/CVE-2019-20387 https://www.cve.org/CVERecord?id=CVE-2019-20387 https://security.access.redhat.com/data/csaf/v2/vex-feed/2019/cve-2019-20387.json"

var rhel_vexValue76 = "A flaw was found in jackson-databind 2.x in versions prior to 2.9.10.5. The interaction between serialization gadgets and typing is mishandled. The highest threat from this vulnerability is to data confidentiality and integrity as well as system availability."

var rhel_vexValue77 = "https://access.redhat.com/security/cve/CVE-2020-14060 https://nvd.nist.gov/vuln/detail/CVE-2020-14060 https://www.cve.org/CVERecord?id=CVE-2020-14060 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-14060.json"

var rhel_vexValue78 = claircore.Package{Name: "rh-maven35-jackson-databind", Kind: 1}

var rhel_vexValue79 = cpe.Value{V: "rhel_software_collections", Kind: 3}

var rhel_vexValue80 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue79, rhel_vexValue68, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue81 = claircore.Repository{Name: "cpe:2.3:a:redhat:rhel_software_collections:3:*:*:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue80}

var rhel_vexValue82 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2020-14060"}

var rhel_vexValue83 = claircore.Alias{Space: unique.Make("CVE"), Name: "2020-14060"}

var rhel_vexValue84 = "The OpenSSL ECDSA signature algorithm has been shown to be vulnerable to a timing side channel attack. An attacker could use variations in the signing algorithm to recover the private key. Fixed in OpenSSL 1.1.0j (Affected 1.1.0-1.1.0i). Fixed in OpenSSL 1.1.1a (Affected 1.1.1)."

var rhel_vexValue85 = "https://access.redhat.com/security/cve/CVE-2018-0735 https://nvd.nist.gov/vuln/detail/CVE-2018-0735 https://www.cve.org/CVERecord?id=CVE-2018-0735 https://security.access.redhat.com/data/csaf/v2/vex-feed/2018/cve-2018-0735.json https://access.redhat.com/errata/RHSA-2019:0483"

var rhel_vexValue86 = claircore.Package{Name: "openssl", Kind: 2}

var rhel_vexValue87 = cpe.Value{V: "server", Kind: 3}

var rhel_vexValue88 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue3, rhel_vexValue4, rhel_vexValue16, rhel_vexValue62, rhel_vexValue7, rhel_vexValue87, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue89 = claircore.Repository{Name: "cpe:2.3:o:redhat:enterprise_linux:7:*:server:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue88}

var rhel_vexValue90 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2018-0735"}

var rhel_vexValue91 = claircore.Alias{Space: unique.Make("CVE"), Name: "2018-0735"}

var rhel_vexValue92 = claircore.Package{Name: "openssl-libs", Kind: 2}

var rhel_vexValue93 = cpe.Value{V: "client", Kind: 3}

var rhel_vexValue94 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue3, rhel_vexValue4, rhel_vexValue16, rhel_vexValue62, rhel_vexValue7, rhel_vexValue93, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue95 = claircore.Repository{Name: "cpe:2.3:o:redhat:enterprise_linux:7:*:client:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue94}

var rhel_vexValue96 = cpe.Value{V: "computenode", Kind: 3}

var rhel_vexValue97 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue3, rhel_vexValue4, rhel_vexValue16, rhel_vexValue62, rhel_vexValue7, rhel_vexValue96, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue98 = claircore.Repository{Name: "cpe:2.3:o:redhat:enterprise_linux:7:*:computenode:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue97}

var rhel_vexValue99 = cpe.Value{V: "workstation", Kind: 3}

var rhel_vexValue100 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue3, rhel_vexValue4, rhel_vexValue16, rhel_vexValue62, rhel_vexValue7, rhel_vexValue99, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue101 = claircore.Repository{Name: "cpe:2.3:o:redhat:enterprise_linux:7:*:workstation:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue100}

var rhel_vexValue102 = "https://access.redhat.com/security/cve/CVE-2018-0735 https://nvd.nist.gov/vuln/detail/CVE-2018-0735 https://www.cve.org/CVERecord?id=CVE-2018-0735 https://security.access.redhat.com/data/csaf/v2/vex-feed/2018/cve-2018-0735.json https://access.redhat.com/errata/RHSA-2019:3700"

var rhel_vexValue103 = "https://access.redhat.com/security/cve/CVE-2018-0735 https://nvd.nist.gov/vuln/detail/CVE-2018-0735 https://www.cve.org/CVERecord?id=CVE-2018-0735 https://security.access.redhat.com/data/csaf/v2/vex-feed/2018/cve-2018-0735.json"

var rhel_vexValue104 = claircore.Package{Name: "openssl", Kind: 1}

var rhel_vexValue105 = claircore.Package{Name: "openssl-libs", Kind: 1}

var rhel_vexValue106 = "A flaw was found in libsolv. A buffer overflow vulnerability in the pool_installable_whatprovides function allows attackers to cause a denial of service. The highest threat from this vulnerability is to system availability."

var rhel_vexValue107 = "https://access.redhat.com/security/cve/CVE-2021-33930 https://nvd.nist.gov/vuln/detail/CVE-2021-33930 https://www.cve.org/CVERecord?id=CVE-2021-33930 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-33930.json https://access.redhat.com/errata/RHSA-2021:4060"

var rhel_vexValue108 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2021-33930"}

var rhel_vexValue109 = claircore.Alias{Space: unique.Make("CVE"), Name: "2021-33930"}

var rhel_vexValue110 = "https://access.redhat.com/security/cve/CVE-2021-33930 https://nvd.nist.gov/vuln/detail/CVE-2021-33930 https://www.cve.org/CVERecord?id=CVE-2021-33930 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-33930.json https://access.redhat.com/errata/RHSA-2022:5498"

var rhel_vexValue111 = "https://access.redhat.com/security/cve/CVE-2021-33930 https://nvd.nist.gov/vuln/detail/CVE-2021-33930 https://www.cve.org/CVERecord?id=CVE-2021-33930 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-33930.json"

var rhel_vexValue112 = "A flaw was found in nodejs. When too many connection attempts with an 'unknownProtocol' are established a leak of file descriptors can occur leading to a potential denial of service. If a file descriptor limit is configured on the system, then the server is unable to accept new connections and prevent the process also from opening. If no file descriptor limit is configured, then this can lead to an excessive memory usage and cause the system to run out of memory. The highest threat from this vulnerability is to system availability."

var rhel_vexValue113 = "https://access.redhat.com/security/cve/CVE-2021-22883 https://nvd.nist.gov/vuln/detail/CVE-2021-22883 https://www.cve.org/CVERecord?id=CVE-2021-22883 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-22883.json https://access.redhat.com/errata/RHSA-2021:0734"

var rhel_vexValue114 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2021-22883"}

var rhel_vexValue115 = claircore.Alias{Space: unique.Make("CVE"), Name: "2021-22883"}

var rhel_vexValue116 = "https://access.redhat.com/security/cve/CVE-2021-22883 https://nvd.nist.gov/vuln/detail/CVE-2021-22883 https://www.cve.org/CVERecord?id=CVE-2021-22883 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-22883.json https://access.redhat.com/errata/RHSA-2021:0735"

var rhel_vexValue117 = "https://access.redhat.com/security/cve/CVE-2021-22883 https://nvd.nist.gov/vuln/detail/CVE-2021-22883 https://www.cve.org/CVERecord?id=CVE-2021-22883 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-22883.json https://access.redhat.com/errata/RHSA-2021:0738"

var rhel_vexValue118 = cpe.Value{V: "8\\.2", Kind: 3}

var rhel_vexValue119 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue34, rhel_vexValue118, rhel_vexValue7, rhel_vexValue18, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue120 = claircore.Repository{Name: "cpe:2.3:a:redhat:rhel_eus:8.2:*:appstream:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue119}

var rhel_vexValue121 = "https://access.redhat.com/security/cve/CVE-2021-22883 https://nvd.nist.gov/vuln/detail/CVE-2021-22883 https://www.cve.org/CVERecord?id=CVE-2021-22883 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-22883.json https://access.redhat.com/errata/RHSA-2021:0739"

var rhel_vexValue122 = cpe.Value{V: "8\\.1", Kind: 3}

var rhel_vexValue123 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue34, rhel_vexValue122, rhel_vexValue7, rhel_vexValue18, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue124 = claircore.Repository{Name: "cpe:2.3:a:redhat:rhel_eus:8.1:*:appstream:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue123}

var rhel_vexValue125 = "https://access.redhat.com/security/cve/CVE-2021-22883 https://nvd.nist.gov/vuln/detail/CVE-2021-22883 https://www.cve.org/CVERecord?id=CVE-2021-22883 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-22883.json https://access.redhat.com/errata/RHSA-2021:0740"

var rhel_vexValue126 = "https://access.redhat.com/security/cve/CVE-2021-22883 https://nvd.nist.gov/vuln/detail/CVE-2021-22883 https://www.cve.org/CVERecord?id=CVE-2021-22883 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-22883.json https://access.redhat.com/errata/RHSA-2021:0741"

var rhel_vexValue127 = "https://access.redhat.com/security/cve/CVE-2021-22883 https://nvd.nist.gov/vuln/detail/CVE-2021-22883 https://www.cve.org/CVERecord?id=CVE-2021-22883 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-22883.json https://access.redhat.com/errata/RHSA-2021:0744"

var rhel_vexValue128 = "https://access.redhat.com/security/cve/CVE-2021-22883 https://nvd.nist.gov/vuln/detail/CVE-2021-22883 https://www.cve.org/CVERecord?id=CVE-2021-22883 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-22883.json"

var rhel_vexValue129 = cpe.Value{V: "rhel_aus", Kind: 3}

var rhel_vexValue130 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue129, rhel_vexValue118, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue131 = claircore.Repository{Name: "cpe:2.3:a:redhat:rhel_aus:8.2:*:*:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue130}

var rhel_vexValue132 = claircore.Package{Name: "nodejs", Kind: 1}

var rhel_vexValue133 = cpe.Value{V: "9", Kind: 3}

var rhel_vexValue134 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue16, rhel_vexValue133, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue135 = claircore.Repository{Name: "cpe:2.3:a:redhat:enterprise_linux:9:*:*:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue134}

var rhel_vexValue136 = "A heap buffer overflow in the TFTP receiving code allows for DoS or arbitrary code execution in libcurl versions 7.19.4 through 7.64.1."

var rhel_vexValue137 = "https://access.redhat.com/security/cve/CVE-2019-5436 https://nvd.nist.gov/vuln/detail/CVE-2019-5436 https://www.cve.org/CVERecord?id=CVE-2019-5436 https://security.access.redhat.com/data/csaf/v2/vex-feed/2019/cve-2019-5436.json https://access.redhat.com/errata/RHSA-2020:1020"

var rhel_vexValue138 = claircore.Package{Name: "curl", Kind: 2}

var rhel_vexValue139 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2019-5436"}

var rhel_vexValue140 = claircore.Alias{Space: unique.Make("CVE"), Name: "2019-5436"}

var rhel_vexValue141 = "https://access.redhat.com/security/cve/CVE-2019-5436 https://nvd.nist.gov/vuln/detail/CVE-2019-5436 https://www.cve.org/CVERecord?id=CVE-2019-5436 https://security.access.redhat.com/data/csaf/v2/vex-feed/2019/cve-2019-5436.json https://access.redhat.com/errata/RHSA-2020:1792"

var rhel_vexValue142 = "https://access.redhat.com/security/cve/CVE-2019-5436 https://nvd.nist.gov/vuln/detail/CVE-2019-5436 https://www.cve.org/CVERecord?id=CVE-2019-5436 https://security.access.redhat.com/data/csaf/v2/vex-feed/2019/cve-2019-5436.json https://access.redhat.com/errata/RHSA-2020:2505"

var rhel_vexValue143 = cpe.Value{V: "7\\.7", Kind: 3}

var rhel_vexValue144 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue3, rhel_vexValue4, rhel_vexValue34, rhel_vexValue143, rhel_vexValue7, rhel_vexValue96, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue145 = claircore.Repository{Name: "cpe:2.3:o:redhat:rhel_eus:7.7:*:computenode:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue144}

var rhel_vexValue146 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue3, rhel_vexValue4, rhel_vexValue34, rhel_vexValue143, rhel_vexValue7, rhel_vexValue87, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue147 = claircore.Repository{Name: "cpe:2.3:o:redhat:rhel_eus:7.7:*:server:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue146}

var rhel_vexValue148 = "https://access.redhat.com/security/cve/CVE-2019-5436 https://nvd.nist.gov/vuln/detail/CVE-2019-5436 https://www.cve.org/CVERecord?id=CVE-2019-5436 https://security.access.redhat.com/data/csaf/v2/vex-feed/2019/cve-2019-5436.json"

var rhel_vexValue149 = claircore.Package{Name: "curl", Kind: 1}

var rhel_vexValue150 = "A vulnerability was found in nodesjs-yargs-parser, where it can be tricked into adding or modifying properties of the Object.prototype using a \"__proto__\" payload. The highest threat from this vulnerability is to confidentiality, integrity, as well as system availability."

var rhel_vexValue151 = "https://access.redhat.com/security/cve/CVE-2020-7608 https://nvd.nist.gov/vuln/detail/CVE-2020-7608 https://www.cve.org/CVERecord?id=CVE-2020-7608 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-7608.json https://access.redhat.com/errata/RHSA-2020:5499"

var rhel_vexValue152 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2020-7608"}

var rhel_vexValue153 = claircore.Alias{Space: unique.Make("CVE"), Name: "2020-7608"}

var rhel_vexValue154 = "https://access.redhat.com/security/cve/CVE-2020-7608 https://nvd.nist.gov/vuln/detail/CVE-2020-7608 https://www.cve.org/CVERecord?id=CVE-2020-7608 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-7608.json https://access.redhat.com/errata/RHSA-2021:0548"

var rhel_vexValue155 = "https://access.redhat.com/security/cve/CVE-2020-7608 https://nvd.nist.gov/vuln/detail/CVE-2020-7608 https://www.cve.org/CVERecord?id=CVE-2020-7608 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-7608.json"

var rhel_vexValue156 = "A heap buffer overflow leading to out-of-bounds write was found in freetype. Memory allocation based on truncated PNG width and height values allows for an out-of-bounds write to occur in application memory when an attacker supplies a specially crafted TTF file."

var rhel_vexValue157 = "https://access.redhat.com/security/cve/CVE-2020-15999 https://nvd.nist.gov/vuln/detail/CVE-2020-15999 https://www.cve.org/CVERecord?id=CVE-2020-15999 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-15999.json https://access.redhat.com/errata/RHSA-2020:4907"

var rhel_vexValue158 = claircore.Package{Name: "freetype", Kind: 2}

var rhel_vexValue159 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2020-15999"}

var rhel_vexValue160 = claircore.Alias{Space: unique.Make("CVE"), Name: "2020-15999"}

var rhel_vexValue161 = "https://access.redhat.com/security/cve/CVE-2020-15999 https://nvd.nist.gov/vuln/detail/CVE-2020-15999 https://www.cve.org/CVERecord?id=CVE-2020-15999 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-15999.json https://access.redhat.com/errata/RHSA-2020:4949"

var rhel_vexValue162 = cpe.Value{V: "rhel_e4s", Kind: 3}

var rhel_vexValue163 = cpe.Value{V: "8\\.0", Kind: 3}

var rhel_vexValue164 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue3, rhel_vexValue4, rhel_vexValue162, rhel_vexValue163, rhel_vexValue7, rhel_vexValue36, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue165 = claircore.Repository{Name: "cpe:2.3:o:redhat:rhel_e4s:8.0:*:baseos:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue164}

var rhel_vexValue166 = "https://access.redhat.com/security/cve/CVE-2020-15999 https://nvd.nist.gov/vuln/detail/CVE-2020-15999 https://www.cve.org/CVERecord?id=CVE-2020-15999 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-15999.json https://access.redhat.com/errata/RHSA-2020:4950"

var rhel_vexValue167 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue3, rhel_vexValue4, rhel_vexValue34, rhel_vexValue122, rhel_vexValue7, rhel_vexValue36, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue168 = claircore.Repository{Name: "cpe:2.3:o:redhat:rhel_eus:8.1:*:baseos:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue167}

var rhel_vexValue169 = "https://access.redhat.com/security/cve/CVE-2020-15999 https://nvd.nist.gov/vuln/detail/CVE-2020-15999 https://www.cve.org/CVERecord?id=CVE-2020-15999 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-15999.json https://access.redhat.com/errata/RHSA-2020:4951"

var rhel_vexValue170 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue3, rhel_vexValue4, rhel_vexValue34, rhel_vexValue118, rhel_vexValue7, rhel_vexValue36, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue171 = claircore.Repository{Name: "cpe:2.3:o:redhat:rhel_eus:8.2:*:baseos:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue170}

var rhel_vexValue172 = "https://access.redhat.com/security/cve/CVE-2020-15999 https://nvd.nist.gov/vuln/detail/CVE-2020-15999 https://www.cve.org/CVERecord?id=CVE-2020-15999 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-15999.json https://access.redhat.com/errata/RHSA-2020:4952"

var rhel_vexValue173 = "https://access.redhat.com/security/cve/CVE-2020-15999 https://nvd.nist.gov/vuln/detail/CVE-2020-15999 https://www.cve.org/CVERecord?id=CVE-2020-15999 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-15999.json"

var rhel_vexValue174 = claircore.Package{Name: "freetype", Kind: 1}

var rhel_vexValue175 = "A microprocessor side-channel vulnerability was found on SMT (e.g, Hyper-Threading) architectures. An attacker running a malicious process on the same core of the processor as the victim process can extract certain secret information."

var rhel_vexValue176 = "https://access.redhat.com/security/cve/CVE-2018-5407 https://nvd.nist.gov/vuln/detail/CVE-2018-5407 https://www.cve.org/CVERecord?id=CVE-2018-5407 https://security.access.redhat.com/data/csaf/v2/vex-feed/2018/cve-2018-5407.json https://access.redhat.com/errata/RHSA-2019:0483"

var rhel_vexValue177 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2018-5407"}

var rhel_vexValue178 = claircore.Alias{Space: unique.Make("CVE"), Name: "2018-5407"}

var rhel_vexValue179 = "https://access.redhat.com/security/cve/CVE-2018-5407 https://nvd.nist.gov/vuln/detail/CVE-2018-5407 https://www.cve.org/CVERecord?id=CVE-2018-5407 https://security.access.redhat.com/data/csaf/v2/vex-feed/2018/cve-2018-5407.json"

var rhel_vexValue180 = cpe.Value{V: "5", Kind: 3}

var rhel_vexValue181 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue3, rhel_vexValue4, rhel_vexValue5, rhel_vexValue180, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue182 = claircore.Repository{Name: "cpe:2.3:o:redhat:rhel_els:5:*:*:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue181}

var rhel_vexValue183 = "A flaw was found in jackson-databind 2.x in versions prior to 2.9.10.6. The interaction between serialization gadgets and typing is mishandled. The highest threat from this vulnerability is to data confidentiality and system availability."

var rhel_vexValue184 = "https://access.redhat.com/security/cve/CVE-2020-24750 https://nvd.nist.gov/vuln/detail/CVE-2020-24750 https://www.cve.org/CVERecord?id=CVE-2020-24750 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-24750.json https://access.redhat.com/errata/RHSA-2020:4173"

var rhel_vexValue185 = claircore.Package{Name: "rh-maven35-jackson-databind", Kind: 2}

var rhel_vexValue186 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue79, rhel_vexValue68, rhel_vexValue7, rhel_vexValue49, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue187 = claircore.Repository{Name: "cpe:2.3:a:redhat:rhel_software_collections:3:*:el7:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue186}

var rhel_vexValue188 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2020-24750"}

var rhel_vexValue189 = claircore.Alias{Space: unique.Make("CVE"), Name: "2020-24750"}

var rhel_vexValue190 = "https://access.redhat.com/security/cve/CVE-2020-24750 https://nvd.nist.gov/vuln/detail/CVE-2020-24750 https://www.cve.org/CVERecord?id=CVE-2020-24750 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-24750.json"

var rhel_vexValue191 = "A flaw was found in FasterXML jackson-databind 2.x in versions prior to 2.9.10.6. The interaction between serialization gadgets and typing are mishandled. The highest threat from this vulnerability is to data confidentiality and integrity as well as system availability."

var rhel_vexValue192 = "https://access.redhat.com/security/cve/CVE-2020-24616 https://nvd.nist.gov/vuln/detail/CVE-2020-24616 https://www.cve.org/CVERecord?id=CVE-2020-24616 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-24616.json"

var rhel_vexValue193 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2020-24616"}

var rhel_vexValue194 = claircore.Alias{Space: unique.Make("CVE"), Name: "2020-24616"}

var rhel_vexValue195 = "A flaw was found in jackson-databind 2.x in versions prior to 2.9.10.5. FasterXML jackson-databind 2.x mishandles the interaction between serialization gadgets and typing. The highest threat from this vulnerability is to data confidentiality and integrity as well as system availability."

var rhel_vexValue196 = "https://access.redhat.com/security/cve/CVE-2020-14061 https://nvd.nist.gov/vuln/detail/CVE-2020-14061 https://www.cve.org/CVERecord?id=CVE-2020-14061 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-14061.json"

var rhel_vexValue197 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2020-14061"}

var rhel_vexValue198 = claircore.Alias{Space: unique.Make("CVE"), Name: "2020-14061"}

var rhel_vexValue199 = "A flaw was found in libsolv. A buffer overflow vulnerability could cause a denial of service. The highest threat from this vulnerability is to system availability."

var rhel_vexValue200 = "https://access.redhat.com/security/cve/CVE-2021-3200 https://nvd.nist.gov/vuln/detail/CVE-2021-3200 https://www.cve.org/CVERecord?id=CVE-2021-3200 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-3200.json https://access.redhat.com/errata/RHSA-2021:4408"

var rhel_vexValue201 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2021-3200"}

var rhel_vexValue202 = claircore.Alias{Space: unique.Make("CVE"), Name: "2021-3200"}

var rhel_vexValue203 = "https://access.redhat.com/security/cve/CVE-2021-3200 https://nvd.nist.gov/vuln/detail/CVE-2021-3200 https://www.cve.org/CVERecord?id=CVE-2021-3200 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-3200.json https://access.redhat.com/errata/RHSA-2022:5498"

var rhel_vexValue204 = "https://access.redhat.com/security/cve/CVE-2021-3200 https://nvd.nist.gov/vuln/detail/CVE-2021-3200 https://www.cve.org/CVERecord?id=CVE-2021-3200 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-3200.json"

var rhel_vexValue205 = cpe.Value{V: "ansible_automation_platform", Kind: 3}

var rhel_vexValue206 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue205, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue207 = claircore.Repository{Name: "cpe:2.3:a:redhat:ansible_automation_platform:*:*:*:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue206}

var rhel_vexValue208 = "A flaw was found in nodejs. When writing to a TLS enabled socket, node::StreamBase::Write calls node::TLSWrap::DoWrite with a freshly allocated WriteWrap object as first argument. If the DoWrite method does not return an error, this object is passed back to the caller as part of a StreamWriteResult structure. This may be exploited to corrupt memory leading to a Denial of Service or potentially other exploits."

var rhel_vexValue209 = "https://access.redhat.com/security/cve/CVE-2020-8265 https://nvd.nist.gov/vuln/detail/CVE-2020-8265 https://www.cve.org/CVERecord?id=CVE-2020-8265 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-8265.json https://access.redhat.com/errata/RHSA-2021:0548"

var rhel_vexValue210 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2020-8265"}

var rhel_vexValue211 = claircore.Alias{Space: unique.Make("CVE"), Name: "2020-8265"}

var rhel_vexValue212 = "https://access.redhat.com/security/cve/CVE-2020-8265 https://nvd.nist.gov/vuln/detail/CVE-2020-8265 https://www.cve.org/CVERecord?id=CVE-2020-8265 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-8265.json https://access.redhat.com/errata/RHSA-2021:0549"

var rhel_vexValue213 = "https://access.redhat.com/security/cve/CVE-2020-8265 https://nvd.nist.gov/vuln/detail/CVE-2020-8265 https://www.cve.org/CVERecord?id=CVE-2020-8265 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-8265.json https://access.redhat.com/errata/RHSA-2021:0551"

var rhel_vexValue214 = "https://access.redhat.com/security/cve/CVE-2020-8265 https://nvd.nist.gov/vuln/detail/CVE-2020-8265 https://www.cve.org/CVERecord?id=CVE-2020-8265 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-8265.json"

var rhel_vexValue215 = "A flaw was found in libgcrypt's ElGamal implementation, where it allows plain text recovery. During the interaction between two cryptographic libraries, a certain combination of the prime defined by the receiver's public key, the generator defined by the receiver's public key, and the sender's ephemeral exponents can lead to a cross-configuration attack against OpenPGP. The highest threat from this vulnerability is to confidentiality."

var rhel_vexValue216 = "https://access.redhat.com/security/cve/CVE-2021-40528 https://nvd.nist.gov/vuln/detail/CVE-2021-40528 https://www.cve.org/CVERecord?id=CVE-2021-40528 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-40528.json https://access.redhat.com/errata/RHSA-2022:5311"

var rhel_vexValue217 = claircore.Package{Name: "libgcrypt", Kind: 2}

var rhel_vexValue218 = cpe.Value{V: "8\\.6", Kind: 3}

var rhel_vexValue219 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue3, rhel_vexValue4, rhel_vexValue34, rhel_vexValue218, rhel_vexValue7, rhel_vexValue36, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue220 = claircore.Repository{Name: "cpe:2.3:o:redhat:rhel_eus:8.6:*:baseos:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue219}

var rhel_vexValue221 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2021-40528"}

var rhel_vexValue222 = claircore.Alias{Space: unique.Make("CVE"), Name: "2021-40528"}

var rhel_vexValue223 = "https://access.redhat.com/security/cve/CVE-2021-40528 https://nvd.nist.gov/vuln/detail/CVE-2021-40528 https://www.cve.org/CVERecord?id=CVE-2021-40528 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-40528.json"

var rhel_vexValue224 = claircore.Package{Name: "libgcrypt", Kind: 1}

var rhel_vexValue225 = "A flaw was found in maven. Repositories that are defined in a dependency’s Project Object Model (pom), which may be unknown to users, are used by default resulting in potential risk if a malicious actor takes over that repository or is able to insert themselves into a position to pretend to be that repository. The highest threat from this vulnerability is to data confidentiality and integrity."

var rhel_vexValue226 = "https://access.redhat.com/security/cve/CVE-2021-26291 https://nvd.nist.gov/vuln/detail/CVE-2021-26291 https://www.cve.org/CVERecord?id=CVE-2021-26291 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-26291.json https://access.redhat.com/errata/RHSA-2023:3198"

var rhel_vexValue227 = claircore.Package{Name: "jenkins", Kind: 2}

var rhel_vexValue228 = cpe.Value{V: "ocp_tools", Kind: 3}

var rhel_vexValue229 = cpe.Value{V: "4\\.11", Kind: 3}

var rhel_vexValue230 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue228, rhel_vexValue229, rhel_vexValue7, rhel_vexValue52, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue231 = claircore.Repository{Name: "cpe:2.3:a:redhat:ocp_tools:4.11:*:el8:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue230}

var rhel_vexValue232 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2021-26291"}

var rhel_vexValue233 = claircore.Alias{Space: unique.Make("CVE"), Name: "2021-26291"}

var rhel_vexValue234 = claircore.Package{Name: "jenkins-2-plugins", Kind: 2}

var rhel_vexValue235 = "https://access.redhat.com/security/cve/CVE-2021-26291 https://nvd.nist.gov/vuln/detail/CVE-2021-26291 https://www.cve.org/CVERecord?id=CVE-2021-26291 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-26291.json https://access.redhat.com/errata/RHSA-2024:0776"

var rhel_vexValue236 = cpe.Value{V: "4\\.13", Kind: 3}

var rhel_vexValue237 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue228, rhel_vexValue236, rhel_vexValue7, rhel_vexValue52, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue238 = claircore.Repository{Name: "cpe:2.3:a:redhat:ocp_tools:4.13:*:el8:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue237}

var rhel_vexValue239 = "https://access.redhat.com/security/cve/CVE-2021-26291 https://nvd.nist.gov/vuln/detail/CVE-2021-26291 https://www.cve.org/CVERecord?id=CVE-2021-26291 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-26291.json https://access.redhat.com/errata/RHSA-2024:0778"

var rhel_vexValue240 = cpe.Value{V: "4\\.12", Kind: 3}

var rhel_vexValue241 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue228, rhel_vexValue240, rhel_vexValue7, rhel_vexValue52, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue242 = claircore.Repository{Name: "cpe:2.3:a:redhat:ocp_tools:4.12:*:el8:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue241}

var rhel_vexValue243 = "https://access.redhat.com/security/cve/CVE-2021-26291 https://nvd.nist.gov/vuln/detail/CVE-2021-26291 https://www.cve.org/CVERecord?id=CVE-2021-26291 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-26291.json"

var rhel_vexValue244 = claircore.Package{Name: "jenkins-2-plugins", Kind: 1}

var rhel_vexValue245 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue228, rhel_vexValue229, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue246 = claircore.Repository{Name: "cpe:2.3:a:redhat:ocp_tools:4.11:*:*:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue245}

var rhel_vexValue247 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue228, rhel_vexValue240, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue248 = claircore.Repository{Name: "cpe:2.3:a:redhat:ocp_tools:4.12:*:*:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue247}

var rhel_vexValue249 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue228, rhel_vexValue236, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue250 = claircore.Repository{Name: "cpe:2.3:a:redhat:ocp_tools:4.13:*:*:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue249}

var rhel_vexValue251 = cpe.Value{V: "openshift", Kind: 3}

var rhel_vexValue252 = cpe.Value{V: "4\\.10", Kind: 3}

var rhel_vexValue253 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue251, rhel_vexValue252, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue254 = claircore.Repository{Name: "cpe:2.3:a:redhat:openshift:4.10:*:*:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue253}

var rhel_vexValue255 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue251, rhel_vexValue229, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue256 = claircore.Repository{Name: "cpe:2.3:a:redhat:openshift:4.11:*:*:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue255}

var rhel_vexValue257 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue251, rhel_vexValue240, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue258 = claircore.Repository{Name: "cpe:2.3:a:redhat:openshift:4.12:*:*:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue257}

var rhel_vexValue259 = cpe.Value{V: "4\\.9", Kind: 3}

var rhel_vexValue260 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue251, rhel_vexValue259, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue261 = claircore.Repository{Name: "cpe:2.3:a:redhat:openshift:4.9:*:*:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue260}

var rhel_vexValue262 = "There's a flaw in lz4. An attacker who submits a crafted file to an application linked with lz4 may be able to trigger an integer overflow, leading to calling of memmove() on a negative size argument, causing an out-of-bounds write and/or a crash. The greatest impact of this flaw is to availability, with some potential impact to confidentiality and integrity as well."

var rhel_vexValue263 = "https://access.redhat.com/security/cve/CVE-2021-3520 https://nvd.nist.gov/vuln/detail/CVE-2021-3520 https://www.cve.org/CVERecord?id=CVE-2021-3520 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-3520.json https://access.redhat.com/errata/RHSA-2021:2575"

var rhel_vexValue264 = claircore.Package{Name: "lz4", Kind: 2}

var rhel_vexValue265 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2021-3520"}

var rhel_vexValue266 = claircore.Alias{Space: unique.Make("CVE"), Name: "2021-3520"}

var rhel_vexValue267 = "https://access.redhat.com/security/cve/CVE-2021-3520 https://nvd.nist.gov/vuln/detail/CVE-2021-3520 https://www.cve.org/CVERecord?id=CVE-2021-3520 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-3520.json"

var rhel_vexValue268 = claircore.Package{Name: "lz4", Kind: 1}

var rhel_vexValue269 = "A heap-based buffer overflow flaw was found in the SOCKS5 proxy handshake in the Curl package. If Curl is unable to resolve the address itself, it passes the hostname to the SOCKS5 proxy. However, the maximum length of the hostname that can be passed is 255 bytes. If the hostname is longer, then Curl switches to the local name resolving and passes the resolved address only to the proxy. The local variable that instructs Curl to \"let the host resolve the name\" could obtain the wrong value during a slow SOCKS5 handshake, resulting in the too-long hostname being copied to the target buffer instead of the resolved address, which was not the intended behavior."

var rhel_vexValue270 = "https://access.redhat.com/security/cve/CVE-2023-38545 https://nvd.nist.gov/vuln/detail/CVE-2023-38545 https://www.cve.org/CVERecord?id=CVE-2023-38545 https://security.access.redhat.com/data/csaf/v2/vex-feed/2023/cve-2023-38545.json https://access.redhat.com/errata/RHSA-2023:5700"

var rhel_vexValue271 = cpe.Value{V: "9\\.0", Kind: 3}

var rhel_vexValue272 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue34, rhel_vexValue271, rhel_vexValue7, rhel_vexValue18, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue273 = claircore.Repository{Name: "cpe:2.3:a:redhat:rhel_eus:9.0:*:appstream:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue272}

var rhel_vexValue274 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2023-38545"}

var rhel_vexValue275 = claircore.Alias{Space: unique.Make("CVE"), Name: "2023-38545"}

var rhel_vexValue276 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue3, rhel_vexValue4, rhel_vexValue34, rhel_vexValue271, rhel_vexValue7, rhel_vexValue36, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue277 = claircore.Repository{Name: "cpe:2.3:o:redhat:rhel_eus:9.0:*:baseos:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue276}

var rhel_vexValue278 = "https://access.redhat.com/security/cve/CVE-2023-38545 https://nvd.nist.gov/vuln/detail/CVE-2023-38545 https://www.cve.org/CVERecord?id=CVE-2023-38545 https://security.access.redhat.com/data/csaf/v2/vex-feed/2023/cve-2023-38545.json https://access.redhat.com/errata/RHSA-2023:5763"

var rhel_vexValue279 = cpe.Value{V: "9\\.2", Kind: 3}

var rhel_vexValue280 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue34, rhel_vexValue279, rhel_vexValue7, rhel_vexValue18, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue281 = claircore.Repository{Name: "cpe:2.3:a:redhat:rhel_eus:9.2:*:appstream:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue280}

var rhel_vexValue282 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue3, rhel_vexValue4, rhel_vexValue34, rhel_vexValue279, rhel_vexValue7, rhel_vexValue36, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue283 = claircore.Repository{Name: "cpe:2.3:o:redhat:rhel_eus:9.2:*:baseos:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue282}

var rhel_vexValue284 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue3, rhel_vexValue4, rhel_vexValue16, rhel_vexValue133, rhel_vexValue7, rhel_vexValue36, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue285 = claircore.Repository{Name: "cpe:2.3:o:redhat:enterprise_linux:9:*:baseos:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue284}

var rhel_vexValue286 = "https://access.redhat.com/security/cve/CVE-2023-38545 https://nvd.nist.gov/vuln/detail/CVE-2023-38545 https://www.cve.org/CVERecord?id=CVE-2023-38545 https://security.access.redhat.com/data/csaf/v2/vex-feed/2023/cve-2023-38545.json https://access.redhat.com/errata/RHSA-2023:6745"

var rhel_vexValue287 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue16, rhel_vexValue133, rhel_vexValue7, rhel_vexValue18, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue288 = claircore.Repository{Name: "cpe:2.3:a:redhat:enterprise_linux:9:*:appstream:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue287}

var rhel_vexValue289 = "https://access.redhat.com/security/cve/CVE-2023-38545 https://nvd.nist.gov/vuln/detail/CVE-2023-38545 https://www.cve.org/CVERecord?id=CVE-2023-38545 https://security.access.redhat.com/data/csaf/v2/vex-feed/2023/cve-2023-38545.json"

var rhel_vexValue290 = "A flaw was found in FasterXML Jackson Databind, where it did not have entity expansion secured properly. This flaw allows vulnerability to XML external entity (XXE) attacks. The highest threat from this vulnerability is data integrity."

var rhel_vexValue291 = "https://access.redhat.com/security/cve/CVE-2020-25649 https://nvd.nist.gov/vuln/detail/CVE-2020-25649 https://www.cve.org/CVERecord?id=CVE-2020-25649 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-25649.json https://access.redhat.com/errata/RHSA-2020:4312"

var rhel_vexValue292 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2020-25649"}

var rhel_vexValue293 = claircore.Alias{Space: unique.Make("CVE"), Name: "2020-25649"}

var rhel_vexValue294 = "A flaw was found in vim. The vulnerability occurs due to Illegal memory access and leads to a buffer over-read vulnerability in the utf_ptr2char function. This flaw allows an attacker to input a specially crafted file, leading to a crash or code execution."

var rhel_vexValue295 = "https://access.redhat.com/security/cve/CVE-2022-1927 https://nvd.nist.gov/vuln/detail/CVE-2022-1927 https://www.cve.org/CVERecord?id=CVE-2022-1927 https://security.access.redhat.com/data/csaf/v2/vex-feed/2022/cve-2022-1927.json https://access.redhat.com/errata/RHSA-2022:5813"

var rhel_vexValue296 = claircore.Package{Name: "vim-minimal", Kind: 2}

var rhel_vexValue297 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2022-1927"}

var rhel_vexValue298 = claircore.Alias{Space: unique.Make("CVE"), Name: "2022-1927"}

var rhel_vexValue299 = claircore.Package{Name: "vim", Kind: 2}

var rhel_vexValue300 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue34, rhel_vexValue218, rhel_vexValue7, rhel_vexValue18, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue301 = claircore.Repository{Name: "cpe:2.3:a:redhat:rhel_eus:8.6:*:appstream:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue300}

var rhel_vexValue302 = "https://access.redhat.com/security/cve/CVE-2022-1927 https://nvd.nist.gov/vuln/detail/CVE-2022-1927 https://www.cve.org/CVERecord?id=CVE-2022-1927 https://security.access.redhat.com/data/csaf/v2/vex-feed/2022/cve-2022-1927.json https://access.redhat.com/errata/RHSA-2022:5942"

var rhel_vexValue303 = "https://access.redhat.com/security/cve/CVE-2022-1927 https://nvd.nist.gov/vuln/detail/CVE-2022-1927 https://www.cve.org/CVERecord?id=CVE-2022-1927 https://security.access.redhat.com/data/csaf/v2/vex-feed/2022/cve-2022-1927.json"

var rhel_vexValue304 = claircore.Package{Name: "vim", Kind: 1}

var rhel_vexValue305 = claircore.Package{Name: "vim-minimal", Kind: 1}

var rhel_vexValue306 = "A flaw has been found in libuv. The realpath() implementation performs an incorrect calculation when allocating a buffer, leading to a potential buffer overflow. The highest threat from this vulnerability is to data confidentiality and integrity as well as system availability."

var rhel_vexValue307 = "https://access.redhat.com/security/cve/CVE-2020-8252 https://nvd.nist.gov/vuln/detail/CVE-2020-8252 https://www.cve.org/CVERecord?id=CVE-2020-8252 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-8252.json https://access.redhat.com/errata/RHSA-2020:4272"

var rhel_vexValue308 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2020-8252"}

var rhel_vexValue309 = claircore.Alias{Space: unique.Make("CVE"), Name: "2020-8252"}

var rhel_vexValue310 = "https://access.redhat.com/security/cve/CVE-2020-8252 https://nvd.nist.gov/vuln/detail/CVE-2020-8252 https://www.cve.org/CVERecord?id=CVE-2020-8252 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-8252.json https://access.redhat.com/errata/RHSA-2020:4903"

var rhel_vexValue311 = "https://access.redhat.com/security/cve/CVE-2020-8252 https://nvd.nist.gov/vuln/detail/CVE-2020-8252 https://www.cve.org/CVERecord?id=CVE-2020-8252 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-8252.json https://access.redhat.com/errata/RHSA-2021:0548"

var rhel_vexValue312 = "https://access.redhat.com/security/cve/CVE-2020-8252 https://nvd.nist.gov/vuln/detail/CVE-2020-8252 https://www.cve.org/CVERecord?id=CVE-2020-8252 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-8252.json"

var rhel_vexValue313 = "A flaw was found in vim. The vulnerability occurs due to Illegal memory access and leads to an out-of-bounds write vulnerability in the vim_regsub_both function. This flaw allows an attacker to input a specially crafted file, leading to a crash or code execution."

var rhel_vexValue314 = "https://access.redhat.com/security/cve/CVE-2022-1897 https://nvd.nist.gov/vuln/detail/CVE-2022-1897 https://www.cve.org/CVERecord?id=CVE-2022-1897 https://security.access.redhat.com/data/csaf/v2/vex-feed/2022/cve-2022-1897.json https://access.redhat.com/errata/RHSA-2022:5813"

var rhel_vexValue315 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2022-1897"}

var rhel_vexValue316 = claircore.Alias{Space: unique.Make("CVE"), Name: "2022-1897"}

var rhel_vexValue317 = cpe.Value{V: "rhev_hypervisor", Kind: 3}

var rhel_vexValue318 = cpe.Value{V: "4\\.4", Kind: 3}

var rhel_vexValue319 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue3, rhel_vexValue4, rhel_vexValue317, rhel_vexValue318, rhel_vexValue7, rhel_vexValue52, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue320 = claircore.Repository{Name: "cpe:2.3:o:redhat:rhev_hypervisor:4.4:*:el8:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue319}

var rhel_vexValue321 = "https://access.redhat.com/security/cve/CVE-2022-1897 https://nvd.nist.gov/vuln/detail/CVE-2022-1897 https://www.cve.org/CVERecord?id=CVE-2022-1897 https://security.access.redhat.com/data/csaf/v2/vex-feed/2022/cve-2022-1897.json https://access.redhat.com/errata/RHSA-2022:5942"

var rhel_vexValue322 = "https://access.redhat.com/security/cve/CVE-2022-1897 https://nvd.nist.gov/vuln/detail/CVE-2022-1897 https://www.cve.org/CVERecord?id=CVE-2022-1897 https://security.access.redhat.com/data/csaf/v2/vex-feed/2022/cve-2022-1897.json"

var rhel_vexValue323 = "A prototype pollution flaw was found in nodejs-dot-prop. The function set could be tricked into adding or modifying properties of Object.prototype using any of the constructor, prototype, or _proto_ paths. The highest threat from this vulnerability is to data confidentiality and integrity as well as system availability."

var rhel_vexValue324 = "https://access.redhat.com/security/cve/CVE-2020-8116 https://nvd.nist.gov/vuln/detail/CVE-2020-8116 https://www.cve.org/CVERecord?id=CVE-2020-8116 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-8116.json https://access.redhat.com/errata/RHSA-2020:4272"

var rhel_vexValue325 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2020-8116"}

var rhel_vexValue326 = claircore.Alias{Space: unique.Make("CVE"), Name: "2020-8116"}

var rhel_vexValue327 = "https://access.redhat.com/security/cve/CVE-2020-8116 https://nvd.nist.gov/vuln/detail/CVE-2020-8116 https://www.cve.org/CVERecord?id=CVE-2020-8116 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-8116.json https://access.redhat.com/errata/RHSA-2020:4903"

var rhel_vexValue328 = "https://access.redhat.com/security/cve/CVE-2020-8116 https://nvd.nist.gov/vuln/detail/CVE-2020-8116 https://www.cve.org/CVERecord?id=CVE-2020-8116 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-8116.json https://access.redhat.com/errata/RHSA-2021:0548"

var rhel_vexValue329 = "https://access.redhat.com/security/cve/CVE-2020-8116 https://nvd.nist.gov/vuln/detail/CVE-2020-8116 https://www.cve.org/CVERecord?id=CVE-2020-8116 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-8116.json"

var rhel_vexValue330 = claircore.Package{Name: "openshift4/ose-prometheus-rhel9", Kind: 4}

var rhel_vexValue331 = cpe.Value{V: "4", Kind: 3}

var rhel_vexValue332 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue251, rhel_vexValue331, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue333 = claircore.Repository{Name: "cpe:2.3:a:redhat:openshift:4:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue332}

var rhel_vexValue334 = claircore.Version{Kind: "rhctag"}

var rhel_vexValue335 = claircore.Version{Kind: "rhctag", V: [10]int32{2147483647, 0, 0, 0, 0, 0, 0, 0, 0, 0}}

var rhel_vexValue336 = claircore.Range{Lower: rhel_vexValue334, Upper: rhel_vexValue335}

var rhel_vexValue337 = claircore.Package{Name: "ocs4/mcg-core-rhel8", Kind: 4}

var rhel_vexValue338 = cpe.Value{V: "openshift_container_storage", Kind: 3}

var rhel_vexValue339 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue338, rhel_vexValue331, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue340 = claircore.Repository{Name: "cpe:2.3:a:redhat:openshift_container_storage:4:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue339}

var rhel_vexValue341 = "A flaw was found in the Apache Log4j logging library 2.x. when the logging configuration uses a non-default Pattern Layout with a Context Lookup. Attackers with control over Thread Context Map (MDC) input data can craft malicious input data that contains a recursive lookup and can cause Denial of Service."

var rhel_vexValue342 = "https://access.redhat.com/security/cve/CVE-2021-45105 https://nvd.nist.gov/vuln/detail/CVE-2021-45105 https://www.cve.org/CVERecord?id=CVE-2021-45105 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-45105.json"

var rhel_vexValue343 = claircore.Package{Name: "openshift4/ose-metering-hadoop", Kind: 4}

var rhel_vexValue344 = cpe.Value{V: "4\\.8", Kind: 3}

var rhel_vexValue345 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue251, rhel_vexValue344, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue346 = claircore.Repository{Name: "cpe:2.3:a:redhat:openshift:4.8:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue345}

var rhel_vexValue347 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2021-45105"}

var rhel_vexValue348 = claircore.Alias{Space: unique.Make("CVE"), Name: "2021-45105"}

var rhel_vexValue349 = claircore.Package{Name: "openshift4/ose-metering-hive", Kind: 4}

var rhel_vexValue350 = claircore.Package{Name: "openshift4/ose-metering-presto", Kind: 4}

var rhel_vexValue351 = "A flaw was found in jackson-databind 2.x in versions prior to 2.9.10.5. FasterXML jackson-databind mishandles the interaction between serialization gadgets and typing. The highest threat from this vulnerability is to data confidentiality and integrity as well as system availability."

var rhel_vexValue352 = "https://access.redhat.com/security/cve/CVE-2020-14195 https://nvd.nist.gov/vuln/detail/CVE-2020-14195 https://www.cve.org/CVERecord?id=CVE-2020-14195 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-14195.json"

var rhel_vexValue353 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2020-14195"}

var rhel_vexValue354 = claircore.Alias{Space: unique.Make("CVE"), Name: "2020-14195"}

var rhel_vexValue355 = "It was found that vim applies the opened file read permissions to the swap file, overriding the process' umask. An attacker might search for vim swap files that were not deleted properly, in order to retrieve sensitive data."

var rhel_vexValue356 = "https://access.redhat.com/security/cve/CVE-2017-1000382 https://nvd.nist.gov/vuln/detail/CVE-2017-1000382 https://www.cve.org/CVERecord?id=CVE-2017-1000382 https://security.access.redhat.com/data/csaf/v2/vex-feed/2017/cve-2017-1000382.json"

var rhel_vexValue357 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2017-1000382"}

var rhel_vexValue358 = claircore.Alias{Space: unique.Make("CVE"), Name: "2017-1000382"}

var rhel_vexValue359 = "A flaw was found in openssl. The flag that enables additional security checks of certificates present in a certificate chain was not enabled allowing a confirmation step to verify that certificates in the chain are valid CA certificates is bypassed. The highest threat from this vulnerability is to data confidentiality and integrity."

var rhel_vexValue360 = "https://access.redhat.com/security/cve/CVE-2021-3450 https://nvd.nist.gov/vuln/detail/CVE-2021-3450 https://www.cve.org/CVERecord?id=CVE-2021-3450 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-3450.json https://access.redhat.com/errata/RHSA-2021:1024"

var rhel_vexValue361 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2021-3450"}

var rhel_vexValue362 = claircore.Alias{Space: unique.Make("CVE"), Name: "2021-3450"}

var rhel_vexValue363 = "https://access.redhat.com/security/cve/CVE-2021-3450 https://nvd.nist.gov/vuln/detail/CVE-2021-3450 https://www.cve.org/CVERecord?id=CVE-2021-3450 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-3450.json"

var rhel_vexValue364 = "A flaw was found in the x/crypto/ssh go library. Applications and libraries that misuse the ServerConfig.PublicKeyCallback callback may be susceptible to an authorization bypass. For example, an attacker may send public keys A and B and authenticate with A. PublicKeyCallback would be called only twice, first with A and then with B. A vulnerable application may then make authorization decisions based on key B, for which the attacker does not control the private key. The misuse of ServerConfig.PublicKeyCallback may cause an authorization bypass."

var rhel_vexValue365 = "https://access.redhat.com/security/cve/CVE-2024-45337 https://nvd.nist.gov/vuln/detail/CVE-2024-45337 https://www.cve.org/CVERecord?id=CVE-2024-45337 https://security.access.redhat.com/data/csaf/v2/vex-feed/2024/cve-2024-45337.json"

var rhel_vexValue366 = claircore.Package{Name: "cert-manager/jetstack-cert-manager-acmesolver-rhel9", Kind: 4}

var rhel_vexValue367 = cpe.Value{V: "cert_manager", Kind: 3}

var rhel_vexValue368 = cpe.Value{V: "1\\.14", Kind: 3}

var rhel_vexValue369 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue367, rhel_vexValue368, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue370 = claircore.Repository{Name: "cpe:2.3:a:redhat:cert_manager:1.14:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue369}

var rhel_vexValue371 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2024-45337"}

var rhel_vexValue372 = claircore.Alias{Space: unique.Make("CVE"), Name: "2024-45337"}

var rhel_vexValue373 = claircore.Package{Name: "cert-manager/jetstack-cert-manager-rhel9", Kind: 4}

var rhel_vexValue374 = claircore.Package{Name: "container-native-virtualization/sidecar-shim-rhel9", Kind: 4}

var rhel_vexValue375 = cpe.Value{V: "container_native_virtualization", Kind: 3}

var rhel_vexValue376 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue375, rhel_vexValue240, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue377 = claircore.Repository{Name: "cpe:2.3:a:redhat:container_native_virtualization:4.12:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue376}

var rhel_vexValue378 = claircore.Package{Name: "container-native-virtualization/virt-artifacts-server", Kind: 4}

var rhel_vexValue379 = claircore.Package{Name: "container-native-virtualization/virt-artifacts-server-rhel9", Kind: 4}

var rhel_vexValue380 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue375, rhel_vexValue236, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue381 = claircore.Repository{Name: "cpe:2.3:a:redhat:container_native_virtualization:4.13:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue380}

var rhel_vexValue382 = claircore.Package{Name: "container-native-virtualization/virt-api-rhel9", Kind: 4}

var rhel_vexValue383 = claircore.Package{Name: "container-native-virtualization/virt-controller-rhel9", Kind: 4}

var rhel_vexValue384 = claircore.Package{Name: "container-native-virtualization/virt-exportproxy", Kind: 4}

var rhel_vexValue385 = claircore.Package{Name: "container-native-virtualization/virt-exportproxy-rhel9", Kind: 4}

var rhel_vexValue386 = claircore.Package{Name: "container-native-virtualization/virt-exportserver", Kind: 4}

var rhel_vexValue387 = claircore.Package{Name: "container-native-virtualization/virt-exportserver-rhel9", Kind: 4}

var rhel_vexValue388 = claircore.Package{Name: "container-native-virtualization/virt-handler", Kind: 4}

var rhel_vexValue389 = claircore.Package{Name: "container-native-virtualization/virt-handler-rhel9", Kind: 4}

var rhel_vexValue390 = claircore.Package{Name: "container-native-virtualization/virt-launcher", Kind: 4}

var rhel_vexValue391 = claircore.Package{Name: "container-native-virtualization/virt-launcher-rhel9", Kind: 4}

var rhel_vexValue392 = claircore.Package{Name: "container-native-virtualization/virt-operator-rhel9", Kind: 4}

var rhel_vexValue393 = cpe.Value{V: "4\\.14", Kind: 3}

var rhel_vexValue394 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue375, rhel_vexValue393, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue395 = claircore.Repository{Name: "cpe:2.3:a:redhat:container_native_virtualization:4.14:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue394}

var rhel_vexValue396 = cpe.Value{V: "4\\.15", Kind: 3}

var rhel_vexValue397 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue375, rhel_vexValue396, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue398 = claircore.Repository{Name: "cpe:2.3:a:redhat:container_native_virtualization:4.15:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue397}

var rhel_vexValue399 = cpe.Value{V: "4\\.16", Kind: 3}

var rhel_vexValue400 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue375, rhel_vexValue399, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue401 = claircore.Repository{Name: "cpe:2.3:a:redhat:container_native_virtualization:4.16:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue400}

var rhel_vexValue402 = cpe.Value{V: "4\\.17", Kind: 3}

var rhel_vexValue403 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue375, rhel_vexValue402, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue404 = claircore.Repository{Name: "cpe:2.3:a:redhat:container_native_virtualization:4.17:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue403}

var rhel_vexValue405 = cpe.Value{V: "4\\.18", Kind: 3}

var rhel_vexValue406 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue375, rhel_vexValue405, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue407 = claircore.Repository{Name: "cpe:2.3:a:redhat:container_native_virtualization:4.18:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue406}

var rhel_vexValue408 = claircore.Package{Name: "multicluster-engine/cluster-api-provider-azure-rhel9", Kind: 4}

var rhel_vexValue409 = cpe.Value{V: "multicluster_engine", Kind: 3}

var rhel_vexValue410 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue409, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue411 = claircore.Repository{Name: "cpe:2.3:a:redhat:multicluster_engine:*:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue410}

var rhel_vexValue412 = claircore.Package{Name: "ocp-tools-4/jenkins-agent-base-rhel8", Kind: 4}

var rhel_vexValue413 = claircore.Repository{Name: "cpe:2.3:a:redhat:ocp_tools:4.11:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue245}

var rhel_vexValue414 = claircore.Package{Name: "ocp-tools-4/jenkins-rhel8", Kind: 4}

var rhel_vexValue415 = claircore.Package{Name: "lifecycle-agent-operator-bundle-container", Kind: 4}

var rhel_vexValue416 = claircore.Repository{Name: "cpe:2.3:a:redhat:openshift:4.12:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue257}

var rhel_vexValue417 = claircore.Package{Name: "openshift-tech-preview/metallb-rhel8", Kind: 4}

var rhel_vexValue418 = claircore.Package{Name: "openshift4-wincw/windows-machine-config-rhel8-operator", Kind: 4}

var rhel_vexValue419 = claircore.Package{Name: "openshift4/ose-baremetal-installer-rhel9", Kind: 4}

var rhel_vexValue420 = claircore.Package{Name: "openshift4/ose-csi-driver-manila-rhel8", Kind: 4}

var rhel_vexValue421 = claircore.Package{Name: "openshift4/ose-csi-driver-nfs-rhel8", Kind: 4}

var rhel_vexValue422 = claircore.Package{Name: "openshift4/ose-docker-builder-rhel9", Kind: 4}

var rhel_vexValue423 = claircore.Package{Name: "openshift4/ose-hyperkube-rhel9", Kind: 4}

var rhel_vexValue424 = claircore.Package{Name: "openshift4/ose-ibmcloud-cluster-api-controllers-rhel8", Kind: 4}

var rhel_vexValue425 = claircore.Package{Name: "openshift4/ose-machine-config-operator", Kind: 4}

var rhel_vexValue426 = claircore.Package{Name: "openshift4/ose-openstack-cinder-csi-driver-rhel8", Kind: 4}

var rhel_vexValue427 = claircore.Package{Name: "openshift4/ose-openstack-cloud-controller-manager-rhel8", Kind: 4}

var rhel_vexValue428 = claircore.Package{Name: "openshift4/ose-tests", Kind: 4}

var rhel_vexValue429 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue251, rhel_vexValue236, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue430 = claircore.Repository{Name: "cpe:2.3:a:redhat:openshift:4.13:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue429}

var rhel_vexValue431 = claircore.Package{Name: "openshift4/ose-installer-altinfra-rhel9", Kind: 4}

var rhel_vexValue432 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue251, rhel_vexValue393, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue433 = claircore.Repository{Name: "cpe:2.3:a:redhat:openshift:4.14:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue432}

var rhel_vexValue434 = claircore.Package{Name: "openshift4/ose-hypershift-rhel8", Kind: 4}

var rhel_vexValue435 = claircore.Package{Name: "openshift4/ose-installer", Kind: 4}

var rhel_vexValue436 = claircore.Package{Name: "openshift4/ose-installer-artifacts", Kind: 4}

var rhel_vexValue437 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue251, rhel_vexValue396, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue438 = claircore.Repository{Name: "cpe:2.3:a:redhat:openshift:4.15:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue437}

var rhel_vexValue439 = claircore.Package{Name: "openshift4/ose-aws-ebs-csi-driver-rhel9", Kind: 4}

var rhel_vexValue440 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue251, rhel_vexValue399, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue441 = claircore.Repository{Name: "cpe:2.3:a:redhat:openshift:4.16:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue440}

var rhel_vexValue442 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue251, rhel_vexValue402, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue443 = claircore.Repository{Name: "cpe:2.3:a:redhat:openshift:4.17:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue442}

var rhel_vexValue444 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue251, rhel_vexValue405, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue445 = claircore.Repository{Name: "cpe:2.3:a:redhat:openshift:4.18:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue444}

var rhel_vexValue446 = claircore.Package{Name: "openshift4/ose-network-metrics-daemon-rhel8", Kind: 4}

var rhel_vexValue447 = claircore.Package{Name: "openshift4/ose-node-feature-discovery-rhel9", Kind: 4}

var rhel_vexValue448 = claircore.Package{Name: "openshift4/csi-provisioner", Kind: 4}

var rhel_vexValue449 = cpe.Value{V: "4\\.19", Kind: 3}

var rhel_vexValue450 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue251, rhel_vexValue449, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue451 = claircore.Repository{Name: "cpe:2.3:a:redhat:openshift:4.19:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue450}

var rhel_vexValue452 = claircore.Package{Name: "openshift4/ose-aws-efs-csi-driver-container-rhel9", Kind: 4}

var rhel_vexValue453 = claircore.Package{Name: "openshift4/ose-azure-disk-csi-driver-rhel8", Kind: 4}

var rhel_vexValue454 = claircore.Package{Name: "openshift4/ose-azure-file-csi-driver-rhel8", Kind: 4}

var rhel_vexValue455 = claircore.Package{Name: "openshift4/ose-smb-csi-driver-rhel9", Kind: 4}

var rhel_vexValue456 = claircore.Package{Name: "openshift4/ose-vmware-vsphere-csi-driver-rhel8", Kind: 4}

var rhel_vexValue457 = claircore.Package{Name: "openshift4/ose-vsphere-csi-driver-syncer-rhel9", Kind: 4}

var rhel_vexValue458 = claircore.Package{Name: "odf4/mcg-rhel9-operator", Kind: 4}

var rhel_vexValue459 = cpe.Value{V: "openshift_data_foundation", Kind: 3}

var rhel_vexValue460 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue459, rhel_vexValue331, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue461 = claircore.Repository{Name: "cpe:2.3:a:redhat:openshift_data_foundation:4:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue460}

var rhel_vexValue462 = claircore.Package{Name: "odf4/odr-rhel9-operator", Kind: 4}

var rhel_vexValue463 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue459, rhel_vexValue393, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue464 = claircore.Repository{Name: "cpe:2.3:a:redhat:openshift_data_foundation:4.14:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue463}

var rhel_vexValue465 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue459, rhel_vexValue396, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue466 = claircore.Repository{Name: "cpe:2.3:a:redhat:openshift_data_foundation:4.15:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue465}

var rhel_vexValue467 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue459, rhel_vexValue399, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue468 = claircore.Repository{Name: "cpe:2.3:a:redhat:openshift_data_foundation:4.16:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue467}

var rhel_vexValue469 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue459, rhel_vexValue402, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue470 = claircore.Repository{Name: "cpe:2.3:a:redhat:openshift_data_foundation:4.17:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue469}

var rhel_vexValue471 = claircore.Package{Name: "osp-director-provisioner-container", Kind: 4}

var rhel_vexValue472 = cpe.Value{V: "openstack", Kind: 3}

var rhel_vexValue473 = cpe.Value{V: "16\\.2", Kind: 3}

var rhel_vexValue474 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue472, rhel_vexValue473, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue475 = claircore.Repository{Name: "cpe:2.3:a:redhat:openstack:16.2:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue474}

var rhel_vexValue476 = claircore.Package{Name: "rhosp-rhel8-tech-preview/osp-director-downloader", Kind: 4}

var rhel_vexValue477 = claircore.Package{Name: "rhosp-rhel8/osp-director-agent", Kind: 4}

var rhel_vexValue478 = claircore.Package{Name: "rhosp-rhel8/osp-director-operator", Kind: 4}

var rhel_vexValue479 = cpe.Value{V: "17\\.1", Kind: 3}

var rhel_vexValue480 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue472, rhel_vexValue479, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue481 = claircore.Repository{Name: "cpe:2.3:a:redhat:openstack:17.1:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue480}

var rhel_vexValue482 = claircore.Package{Name: "rhosp-rhel9/osp-director-agent", Kind: 4}

var rhel_vexValue483 = claircore.Package{Name: "rhosp-rhel9/osp-director-downloader", Kind: 4}

var rhel_vexValue484 = claircore.Package{Name: "octavia-operator-bundle-container", Kind: 4}

var rhel_vexValue485 = cpe.Value{V: "18\\.0", Kind: 3}

var rhel_vexValue486 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue472, rhel_vexValue485, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue487 = claircore.Repository{Name: "cpe:2.3:a:redhat:openstack:18.0:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue486}

var rhel_vexValue488 = claircore.Package{Name: "rhoso-operators/octavia-rhel9-operator", Kind: 4}

var rhel_vexValue489 = claircore.Package{Name: "multicluster-globalhub-agent-container", Kind: 4}

var rhel_vexValue490 = cpe.Value{V: "acm", Kind: 3}

var rhel_vexValue491 = cpe.Value{V: "2\\.10", Kind: 3}

var rhel_vexValue492 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue490, rhel_vexValue491, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue493 = claircore.Repository{Name: "cpe:2.3:a:redhat:acm:2.10:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue492}

var rhel_vexValue494 = claircore.Package{Name: "multicluster-globalhub-grafana-container", Kind: 4}

var rhel_vexValue495 = claircore.Package{Name: "multicluster-globalhub-kessel-inventory-api-container", Kind: 4}

var rhel_vexValue496 = claircore.Package{Name: "multicluster-globalhub-manager-container", Kind: 4}

var rhel_vexValue497 = claircore.Package{Name: "multicluster-globalhub-operator-bundle-container", Kind: 4}

var rhel_vexValue498 = claircore.Package{Name: "multicluster-globalhub-operator-container", Kind: 4}

var rhel_vexValue499 = claircore.Package{Name: "multicluster-globalhub-postgres-exporter-container", Kind: 4}

var rhel_vexValue500 = claircore.Package{Name: "rhacm2/gatekeeper-rhel8-operator", Kind: 4}

var rhel_vexValue501 = cpe.Value{V: "2\\.11", Kind: 3}

var rhel_vexValue502 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue490, rhel_vexValue501, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue503 = claircore.Repository{Name: "cpe:2.3:a:redhat:acm:2.11:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue502}

var rhel_vexValue504 = cpe.Value{V: "2", Kind: 3}

var rhel_vexValue505 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue490, rhel_vexValue504, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue506 = claircore.Repository{Name: "cpe:2.3:a:redhat:acm:2:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue505}

var rhel_vexValue507 = claircore.Package{Name: "rhacm2/gatekeeper-rhel8", Kind: 4}

var rhel_vexValue508 = claircore.Package{Name: "advanced-cluster-security/rhacs-scanner-db-rhel8", Kind: 4}

var rhel_vexValue509 = cpe.Value{V: "advanced_cluster_security", Kind: 3}

var rhel_vexValue510 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue509, rhel_vexValue331, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue511 = claircore.Repository{Name: "cpe:2.3:a:redhat:advanced_cluster_security:4:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue510}

var rhel_vexValue512 = claircore.Package{Name: "advanced-cluster-security/rhacs-scanner-db-slim-rhel8", Kind: 4}

var rhel_vexValue513 = claircore.Package{Name: "advanced-cluster-security/rhacs-scanner-rhel8", Kind: 4}

var rhel_vexValue514 = claircore.Package{Name: "advanced-cluster-security/rhacs-scanner-slim-rhel8", Kind: 4}

var rhel_vexValue515 = claircore.Package{Name: "advanced-cluster-security/rhacs-scanner-v4-db-rhel8", Kind: 4}

var rhel_vexValue516 = claircore.Package{Name: "advanced-cluster-security/rhacs-scanner-v4-rhel8", Kind: 4}

var rhel_vexValue517 = claircore.Package{Name: "odh-model-registry-container", Kind: 4}

var rhel_vexValue518 = cpe.Value{V: "openshift_ai", Kind: 3}

var rhel_vexValue519 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue518, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue520 = claircore.Repository{Name: "cpe:2.3:a:redhat:openshift_ai:*:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue519}

var rhel_vexValue521 = claircore.Package{Name: "rhtas/rekor-cli-rhel9", Kind: 4}

var rhel_vexValue522 = cpe.Value{V: "trusted_artifact_signer", Kind: 3}

var rhel_vexValue523 = cpe.Value{V: "1\\.0", Kind: 3}

var rhel_vexValue524 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue522, rhel_vexValue523, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue525 = claircore.Repository{Name: "cpe:2.3:a:redhat:trusted_artifact_signer:1.0:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue524}

var rhel_vexValue526 = "A side-channel attack flaw was found in the way libgcrypt implemented Elgamal encryption. This flaw allows an attacker to decrypt parts of ciphertext encrypted using Elgamal, for example, when using OpenPGP. The highest threat from this vulnerability is to confidentiality."

var rhel_vexValue527 = "https://access.redhat.com/security/cve/CVE-2021-33560 https://nvd.nist.gov/vuln/detail/CVE-2021-33560 https://www.cve.org/CVERecord?id=CVE-2021-33560 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-33560.json https://access.redhat.com/errata/RHSA-2021:4409"

var rhel_vexValue528 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2021-33560"}

var rhel_vexValue529 = claircore.Alias{Space: unique.Make("CVE"), Name: "2021-33560"}

var rhel_vexValue530 = "https://access.redhat.com/security/cve/CVE-2021-33560 https://nvd.nist.gov/vuln/detail/CVE-2021-33560 https://www.cve.org/CVERecord?id=CVE-2021-33560 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-33560.json"

var rhel_vexValue531 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue34, rhel_vexValue35, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue532 = claircore.Repository{Name: "cpe:2.3:a:redhat:rhel_eus:8.4:*:*:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue531}

var rhel_vexValue533 = "A stack-based buffer overflow was found in the way OpenSSL processes X.509 certificates with a specially crafted email address field. This issue could cause a server or a client application compiled with OpenSSL to crash when trying to process the malicious certificate."

var rhel_vexValue534 = "https://access.redhat.com/security/cve/CVE-2022-3602 https://nvd.nist.gov/vuln/detail/CVE-2022-3602 https://www.cve.org/CVERecord?id=CVE-2022-3602 https://security.access.redhat.com/data/csaf/v2/vex-feed/2022/cve-2022-3602.json https://access.redhat.com/errata/RHSA-2022:7288"

var rhel_vexValue535 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2022-3602"}

var rhel_vexValue536 = claircore.Alias{Space: unique.Make("CVE"), Name: "2022-3602"}

var rhel_vexValue537 = "https://access.redhat.com/security/cve/CVE-2022-3602 https://nvd.nist.gov/vuln/detail/CVE-2022-3602 https://www.cve.org/CVERecord?id=CVE-2022-3602 https://security.access.redhat.com/data/csaf/v2/vex-feed/2022/cve-2022-3602.json"

var rhel_vexValue538 = claircore.Package{Name: "rhacm2/management-ingress-rhel8", Kind: 4}

var rhel_vexValue539 = cpe.Value{V: "2\\.12", Kind: 3}

var rhel_vexValue540 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue490, rhel_vexValue539, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue541 = claircore.Repository{Name: "cpe:2.3:a:redhat:acm:2.12:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue540}

var rhel_vexValue542 = cpe.Value{V: "2\\.13", Kind: 3}

var rhel_vexValue543 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue490, rhel_vexValue542, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue544 = claircore.Repository{Name: "cpe:2.3:a:redhat:acm:2.13:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue543}

var rhel_vexValue545 = cpe.Value{V: "2\\.14", Kind: 3}

var rhel_vexValue546 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue490, rhel_vexValue545, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue547 = claircore.Repository{Name: "cpe:2.3:a:redhat:acm:2.14:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue546}

var rhel_vexValue548 = cpe.Value{V: "2\\.15", Kind: 3}

var rhel_vexValue549 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue490, rhel_vexValue548, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue550 = claircore.Repository{Name: "cpe:2.3:a:redhat:acm:2.15:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue549}

var rhel_vexValue551 = "A vulnerability was found in systemd-resolved. This issue may allow systemd-resolved to accept records of DNSSEC-signed domains even when they have no signature, allowing man-in-the-middles (or the upstream DNS resolver) to manipulate records."

var rhel_vexValue552 = "https://access.redhat.com/security/cve/CVE-2023-7008 https://nvd.nist.gov/vuln/detail/CVE-2023-7008 https://www.cve.org/CVERecord?id=CVE-2023-7008 https://security.access.redhat.com/data/csaf/v2/vex-feed/2023/cve-2023-7008.json https://access.redhat.com/errata/RHSA-2024:2463"

var rhel_vexValue553 = claircore.Package{Name: "systemd", Kind: 2}

var rhel_vexValue554 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2023-7008"}

var rhel_vexValue555 = claircore.Alias{Space: unique.Make("CVE"), Name: "2023-7008"}

var rhel_vexValue556 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue16, rhel_vexValue133, rhel_vexValue7, rhel_vexValue41, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue557 = claircore.Repository{Name: "cpe:2.3:a:redhat:enterprise_linux:9:*:crb:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue556}

var rhel_vexValue558 = "https://access.redhat.com/security/cve/CVE-2023-7008 https://nvd.nist.gov/vuln/detail/CVE-2023-7008 https://www.cve.org/CVERecord?id=CVE-2023-7008 https://security.access.redhat.com/data/csaf/v2/vex-feed/2023/cve-2023-7008.json https://access.redhat.com/errata/RHSA-2024:3203"

var rhel_vexValue559 = "https://access.redhat.com/security/cve/CVE-2023-7008 https://nvd.nist.gov/vuln/detail/CVE-2023-7008 https://www.cve.org/CVERecord?id=CVE-2023-7008 https://security.access.redhat.com/data/csaf/v2/vex-feed/2023/cve-2023-7008.json"

var rhel_vexValue560 = claircore.Package{Name: "systemd", Kind: 1}

var rhel_vexValue561 = "A flaw was found in jackson-databind. FasterXML mishandles the interaction between serialization gadgets and typing. The highest threat from this vulnerability is to data confidentiality and integrity as well as system availability."

var rhel_vexValue562 = "https://access.redhat.com/security/cve/CVE-2020-35491 https://nvd.nist.gov/vuln/detail/CVE-2020-35491 https://www.cve.org/CVERecord?id=CVE-2020-35491 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-35491.json"

var rhel_vexValue563 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2020-35491"}

var rhel_vexValue564 = claircore.Alias{Space: unique.Make("CVE"), Name: "2020-35491"}

var rhel_vexValue565 = cpe.Value{V: "4\\.20", Kind: 3}

var rhel_vexValue566 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue251, rhel_vexValue565, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue567 = claircore.Repository{Name: "cpe:2.3:a:redhat:openshift:4.20:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue566}

var rhel_vexValue568 = "Improper validation of certificate with host mismatch in Apache Log4j SMTP appender. This could allow an SMTPS connection to be intercepted by a man-in-the-middle attack which could leak any log messages sent through that appender. Fixed in Apache Log4j 2.12.3 and 2.13.1"

var rhel_vexValue569 = "https://access.redhat.com/security/cve/CVE-2020-9488 https://nvd.nist.gov/vuln/detail/CVE-2020-9488 https://www.cve.org/CVERecord?id=CVE-2020-9488 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-9488.json"

var rhel_vexValue570 = claircore.Package{Name: "rh-maven35-log4j12", Kind: 1}

var rhel_vexValue571 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2020-9488"}

var rhel_vexValue572 = claircore.Alias{Space: unique.Make("CVE"), Name: "2020-9488"}

var rhel_vexValue573 = "A flaw was found in openssl. A server crash and denial of service attack could occur if a client sends a TLSv1.2 renegotiation ClientHello and omits the signature_algorithms extension but includes a signature_algorithms_cert extension. The highest threat from this vulnerability is to system availability."

var rhel_vexValue574 = "https://access.redhat.com/security/cve/CVE-2021-3449 https://nvd.nist.gov/vuln/detail/CVE-2021-3449 https://www.cve.org/CVERecord?id=CVE-2021-3449 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-3449.json https://access.redhat.com/errata/RHSA-2021:1024"

var rhel_vexValue575 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2021-3449"}

var rhel_vexValue576 = claircore.Alias{Space: unique.Make("CVE"), Name: "2021-3449"}

var rhel_vexValue577 = "https://access.redhat.com/security/cve/CVE-2021-3449 https://nvd.nist.gov/vuln/detail/CVE-2021-3449 https://www.cve.org/CVERecord?id=CVE-2021-3449 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-3449.json https://access.redhat.com/errata/RHSA-2021:1063"

var rhel_vexValue578 = "https://access.redhat.com/security/cve/CVE-2021-3449 https://nvd.nist.gov/vuln/detail/CVE-2021-3449 https://www.cve.org/CVERecord?id=CVE-2021-3449 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-3449.json https://access.redhat.com/errata/RHSA-2021:1131"

var rhel_vexValue579 = "https://access.redhat.com/security/cve/CVE-2021-3449 https://nvd.nist.gov/vuln/detail/CVE-2021-3449 https://www.cve.org/CVERecord?id=CVE-2021-3449 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-3449.json"

var rhel_vexValue580 = "https://access.redhat.com/security/cve/CVE-2020-14062 https://nvd.nist.gov/vuln/detail/CVE-2020-14062 https://www.cve.org/CVERecord?id=CVE-2020-14062 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-14062.json"

var rhel_vexValue581 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2020-14062"}

var rhel_vexValue582 = claircore.Alias{Space: unique.Make("CVE"), Name: "2020-14062"}

var rhel_vexValue583 = "A buffer overflow was discovered in the GNU C Library's dynamic loader ld.so while processing the GLIBC_TUNABLES environment variable. This issue could allow a local attacker to use maliciously crafted GLIBC_TUNABLES environment variables when launching binaries with SUID permission to execute code with elevated privileges."

var rhel_vexValue584 = "https://access.redhat.com/security/cve/CVE-2023-4911 https://nvd.nist.gov/vuln/detail/CVE-2023-4911 https://www.cve.org/CVERecord?id=CVE-2023-4911 https://security.access.redhat.com/data/csaf/v2/vex-feed/2023/cve-2023-4911.json https://access.redhat.com/errata/RHSA-2023:5453"

var rhel_vexValue585 = claircore.Package{Name: "glibc", Kind: 2}

var rhel_vexValue586 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2023-4911"}

var rhel_vexValue587 = claircore.Alias{Space: unique.Make("CVE"), Name: "2023-4911"}

var rhel_vexValue588 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue34, rhel_vexValue279, rhel_vexValue7, rhel_vexValue41, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue589 = claircore.Repository{Name: "cpe:2.3:a:redhat:rhel_eus:9.2:*:crb:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue588}

var rhel_vexValue590 = "https://access.redhat.com/security/cve/CVE-2023-4911 https://nvd.nist.gov/vuln/detail/CVE-2023-4911 https://www.cve.org/CVERecord?id=CVE-2023-4911 https://security.access.redhat.com/data/csaf/v2/vex-feed/2023/cve-2023-4911.json https://access.redhat.com/errata/RHSA-2023:5454"

var rhel_vexValue591 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue34, rhel_vexValue271, rhel_vexValue7, rhel_vexValue41, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue592 = claircore.Repository{Name: "cpe:2.3:a:redhat:rhel_eus:9.0:*:crb:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue591}

var rhel_vexValue593 = "https://access.redhat.com/security/cve/CVE-2023-4911 https://nvd.nist.gov/vuln/detail/CVE-2023-4911 https://www.cve.org/CVERecord?id=CVE-2023-4911 https://security.access.redhat.com/data/csaf/v2/vex-feed/2023/cve-2023-4911.json https://access.redhat.com/errata/RHSA-2023:5455"

var rhel_vexValue594 = cpe.Value{V: "8\\.8", Kind: 3}

var rhel_vexValue595 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue34, rhel_vexValue594, rhel_vexValue7, rhel_vexValue41, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue596 = claircore.Repository{Name: "cpe:2.3:a:redhat:rhel_eus:8.8:*:crb:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue595}

var rhel_vexValue597 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue3, rhel_vexValue4, rhel_vexValue34, rhel_vexValue594, rhel_vexValue7, rhel_vexValue36, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue598 = claircore.Repository{Name: "cpe:2.3:o:redhat:rhel_eus:8.8:*:baseos:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue597}

var rhel_vexValue599 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue34, rhel_vexValue594, rhel_vexValue7, rhel_vexValue18, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue600 = claircore.Repository{Name: "cpe:2.3:a:redhat:rhel_eus:8.8:*:appstream:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue599}

var rhel_vexValue601 = "https://access.redhat.com/security/cve/CVE-2023-4911 https://nvd.nist.gov/vuln/detail/CVE-2023-4911 https://www.cve.org/CVERecord?id=CVE-2023-4911 https://security.access.redhat.com/data/csaf/v2/vex-feed/2023/cve-2023-4911.json https://access.redhat.com/errata/RHSA-2023:5476"

var rhel_vexValue602 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue34, rhel_vexValue218, rhel_vexValue7, rhel_vexValue41, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue603 = claircore.Repository{Name: "cpe:2.3:a:redhat:rhel_eus:8.6:*:crb:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue602}

var rhel_vexValue604 = "https://access.redhat.com/security/cve/CVE-2023-4911 https://nvd.nist.gov/vuln/detail/CVE-2023-4911 https://www.cve.org/CVERecord?id=CVE-2023-4911 https://security.access.redhat.com/data/csaf/v2/vex-feed/2023/cve-2023-4911.json"

var rhel_vexValue605 = claircore.Package{Name: "glibc", Kind: 1}

var rhel_vexValue606 = "Versions of the npm CLI prior to 6.14.6 are vulnerable to an information exposure vulnerability through log files. The CLI supports URLs like \"<protocol>://[<user>[:<password>]@]<hostname>[:<port>][:][/]<path>\". The password value is not redacted and is printed to stdout and also to any generated log files."

var rhel_vexValue607 = "https://access.redhat.com/security/cve/CVE-2020-15095 https://nvd.nist.gov/vuln/detail/CVE-2020-15095 https://www.cve.org/CVERecord?id=CVE-2020-15095 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-15095.json https://access.redhat.com/errata/RHSA-2020:4272"

var rhel_vexValue608 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2020-15095"}

var rhel_vexValue609 = claircore.Alias{Space: unique.Make("CVE"), Name: "2020-15095"}

var rhel_vexValue610 = "https://access.redhat.com/security/cve/CVE-2020-15095 https://nvd.nist.gov/vuln/detail/CVE-2020-15095 https://www.cve.org/CVERecord?id=CVE-2020-15095 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-15095.json https://access.redhat.com/errata/RHSA-2020:4903"

var rhel_vexValue611 = "https://access.redhat.com/security/cve/CVE-2020-15095 https://nvd.nist.gov/vuln/detail/CVE-2020-15095 https://www.cve.org/CVERecord?id=CVE-2020-15095 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-15095.json https://access.redhat.com/errata/RHSA-2021:0548"

var rhel_vexValue612 = "https://access.redhat.com/security/cve/CVE-2020-15095 https://nvd.nist.gov/vuln/detail/CVE-2020-15095 https://www.cve.org/CVERecord?id=CVE-2020-15095 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-15095.json"

var rhel_vexValue613 = "A flaw was found in the Pipeline Input Step Plugin. This issue affects the code of the component Archive File Handler. The manipulation of the argument file with a malicious input leads to a directory traversal vulnerability."

var rhel_vexValue614 = "https://access.redhat.com/security/cve/CVE-2022-34177 https://nvd.nist.gov/vuln/detail/CVE-2022-34177 https://www.cve.org/CVERecord?id=CVE-2022-34177 https://security.access.redhat.com/data/csaf/v2/vex-feed/2022/cve-2022-34177.json https://access.redhat.com/errata/RHSA-2022:6531"

var rhel_vexValue615 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue251, rhel_vexValue252, rhel_vexValue7, rhel_vexValue52, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue616 = claircore.Repository{Name: "cpe:2.3:a:redhat:openshift:4.10:*:el8:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue615}

var rhel_vexValue617 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2022-34177"}

var rhel_vexValue618 = claircore.Alias{Space: unique.Make("CVE"), Name: "2022-34177"}

var rhel_vexValue619 = "https://access.redhat.com/security/cve/CVE-2022-34177 https://nvd.nist.gov/vuln/detail/CVE-2022-34177 https://www.cve.org/CVERecord?id=CVE-2022-34177 https://security.access.redhat.com/data/csaf/v2/vex-feed/2022/cve-2022-34177.json https://access.redhat.com/errata/RHSA-2022:9110"

var rhel_vexValue620 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue251, rhel_vexValue259, rhel_vexValue7, rhel_vexValue52, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue621 = claircore.Repository{Name: "cpe:2.3:a:redhat:openshift:4.9:*:el8:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue620}

var rhel_vexValue622 = "https://access.redhat.com/security/cve/CVE-2022-34177 https://nvd.nist.gov/vuln/detail/CVE-2022-34177 https://www.cve.org/CVERecord?id=CVE-2022-34177 https://security.access.redhat.com/data/csaf/v2/vex-feed/2022/cve-2022-34177.json https://access.redhat.com/errata/RHSA-2023:0017"

var rhel_vexValue623 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue251, rhel_vexValue344, rhel_vexValue7, rhel_vexValue52, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue624 = claircore.Repository{Name: "cpe:2.3:a:redhat:openshift:4.8:*:el8:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue623}

var rhel_vexValue625 = "https://access.redhat.com/security/cve/CVE-2022-34177 https://nvd.nist.gov/vuln/detail/CVE-2022-34177 https://www.cve.org/CVERecord?id=CVE-2022-34177 https://security.access.redhat.com/data/csaf/v2/vex-feed/2022/cve-2022-34177.json"

var rhel_vexValue626 = cpe.Value{V: "4\\.6", Kind: 3}

var rhel_vexValue627 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue251, rhel_vexValue626, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue628 = claircore.Repository{Name: "cpe:2.3:a:redhat:openshift:4.6:*:*:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue627}

var rhel_vexValue629 = cpe.Value{V: "4\\.7", Kind: 3}

var rhel_vexValue630 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue251, rhel_vexValue629, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue631 = claircore.Repository{Name: "cpe:2.3:a:redhat:openshift:4.7:*:*:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue630}

var rhel_vexValue632 = claircore.Repository{Name: "cpe:2.3:a:redhat:openshift:4.8:*:*:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue345}

var rhel_vexValue633 = cpe.Value{V: "3\\.11", Kind: 3}

var rhel_vexValue634 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue251, rhel_vexValue633, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue635 = claircore.Repository{Name: "cpe:2.3:a:redhat:openshift:3.11:*:*:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue634}

var rhel_vexValue636 = "https://access.redhat.com/security/cve/CVE-2020-35490 https://nvd.nist.gov/vuln/detail/CVE-2020-35490 https://www.cve.org/CVERecord?id=CVE-2020-35490 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-35490.json"

var rhel_vexValue637 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2020-35490"}

var rhel_vexValue638 = claircore.Alias{Space: unique.Make("CVE"), Name: "2020-35490"}

var rhel_vexValue639 = "Apache Log4j2 versions 2.0-beta7 through 2.17.0 (excluding security fix releases 2.3.2 and 2.12.4) are vulnerable to a remote code execution (RCE) attack where an attacker with permission to modify the logging configuration file can construct a malicious configuration using a JDBC Appender with a data source referencing a JNDI URI which can execute remote code. This issue is fixed by limiting JNDI data source names to the java protocol in Log4j2 versions 2.17.1, 2.12.4, and 2.3.2."

var rhel_vexValue640 = "https://access.redhat.com/security/cve/CVE-2021-44832 https://nvd.nist.gov/vuln/detail/CVE-2021-44832 https://www.cve.org/CVERecord?id=CVE-2021-44832 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-44832.json"

var rhel_vexValue641 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2021-44832"}

var rhel_vexValue642 = claircore.Alias{Space: unique.Make("CVE"), Name: "2021-44832"}

var rhel_vexValue643 = "A stack-based buffer overflow was found in the way OpenSSL processes X.509 certificates with a specially crafted email address field. This issue could cause a server or a client application compiled with OpenSSL to crash or possibly execute remote code when trying to process the malicious certificate."

var rhel_vexValue644 = "https://access.redhat.com/security/cve/CVE-2022-3786 https://nvd.nist.gov/vuln/detail/CVE-2022-3786 https://www.cve.org/CVERecord?id=CVE-2022-3786 https://security.access.redhat.com/data/csaf/v2/vex-feed/2022/cve-2022-3786.json https://access.redhat.com/errata/RHSA-2022:7288"

var rhel_vexValue645 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2022-3786"}

var rhel_vexValue646 = claircore.Alias{Space: unique.Make("CVE"), Name: "2022-3786"}

var rhel_vexValue647 = "https://access.redhat.com/security/cve/CVE-2022-3786 https://nvd.nist.gov/vuln/detail/CVE-2022-3786 https://www.cve.org/CVERecord?id=CVE-2022-3786 https://security.access.redhat.com/data/csaf/v2/vex-feed/2022/cve-2022-3786.json"

var rhel_vexValue648 = "AES OCB mode for 32-bit x86 platforms using the AES-NI assembly optimized implementation will not encrypt the entirety of the data under some circumstances. This could reveal sixteen bytes of data that was preexisting in the memory that wasn't written. In the special case of \"in place\" encryption, sixteen bytes of the plaintext would be revealed."

var rhel_vexValue649 = "https://access.redhat.com/security/cve/CVE-2022-2097 https://nvd.nist.gov/vuln/detail/CVE-2022-2097 https://www.cve.org/CVERecord?id=CVE-2022-2097 https://security.access.redhat.com/data/csaf/v2/vex-feed/2022/cve-2022-2097.json https://access.redhat.com/errata/RHSA-2022:5818"

var rhel_vexValue650 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2022-2097"}

var rhel_vexValue651 = claircore.Alias{Space: unique.Make("CVE"), Name: "2022-2097"}

var rhel_vexValue652 = "https://access.redhat.com/security/cve/CVE-2022-2097 https://nvd.nist.gov/vuln/detail/CVE-2022-2097 https://www.cve.org/CVERecord?id=CVE-2022-2097 https://security.access.redhat.com/data/csaf/v2/vex-feed/2022/cve-2022-2097.json https://access.redhat.com/errata/RHSA-2022:6224"

var rhel_vexValue653 = "https://access.redhat.com/security/cve/CVE-2022-2097 https://nvd.nist.gov/vuln/detail/CVE-2022-2097 https://www.cve.org/CVERecord?id=CVE-2022-2097 https://security.access.redhat.com/data/csaf/v2/vex-feed/2022/cve-2022-2097.json"

var rhel_vexValue654 = "A flaw was found in systemd. The use of alloca function with an uncontrolled size in function unit_name_path_escape allows a local attacker, able to mount a filesystem on a very long path, to crash systemd and the whole system by allocating a very large space in the stack. The highest threat from this vulnerability is to the system availability."

var rhel_vexValue655 = "https://access.redhat.com/security/cve/CVE-2021-33910 https://nvd.nist.gov/vuln/detail/CVE-2021-33910 https://www.cve.org/CVERecord?id=CVE-2021-33910 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-33910.json https://access.redhat.com/errata/RHSA-2021:2717"

var rhel_vexValue656 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2021-33910"}

var rhel_vexValue657 = claircore.Alias{Space: unique.Make("CVE"), Name: "2021-33910"}

var rhel_vexValue658 = "https://access.redhat.com/security/cve/CVE-2021-33910 https://nvd.nist.gov/vuln/detail/CVE-2021-33910 https://www.cve.org/CVERecord?id=CVE-2021-33910 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-33910.json https://access.redhat.com/errata/RHSA-2021:2721"

var rhel_vexValue659 = "https://access.redhat.com/security/cve/CVE-2021-33910 https://nvd.nist.gov/vuln/detail/CVE-2021-33910 https://www.cve.org/CVERecord?id=CVE-2021-33910 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-33910.json https://access.redhat.com/errata/RHSA-2021:2724"

var rhel_vexValue660 = "https://access.redhat.com/security/cve/CVE-2021-33910 https://nvd.nist.gov/vuln/detail/CVE-2021-33910 https://www.cve.org/CVERecord?id=CVE-2021-33910 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-33910.json"

var rhel_vexValue661 = "A flaw was found in the golang.org/x/crypto/ssh package. SSH clients and servers are vulnerable to increased resource consumption, possibly leading to memory exhaustion and a DoS. This can occur during key exchange when the other party is slow to respond during key exchange."

var rhel_vexValue662 = "https://access.redhat.com/security/cve/CVE-2025-22869 https://nvd.nist.gov/vuln/detail/CVE-2025-22869 https://www.cve.org/CVERecord?id=CVE-2025-22869 https://security.access.redhat.com/data/csaf/v2/vex-feed/2025/cve-2025-22869.json"

var rhel_vexValue663 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2025-22869"}

var rhel_vexValue664 = claircore.Alias{Space: unique.Make("CVE"), Name: "2025-22869"}

var rhel_vexValue665 = claircore.Package{Name: "cert-manager/cert-manager-operator-rhel9", Kind: 4}

var rhel_vexValue666 = cpe.Value{V: "1\\.16", Kind: 3}

var rhel_vexValue667 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue367, rhel_vexValue666, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue668 = claircore.Repository{Name: "cpe:2.3:a:redhat:cert_manager:1.16:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue667}

var rhel_vexValue669 = cpe.Value{V: "1\\.17", Kind: 3}

var rhel_vexValue670 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue367, rhel_vexValue669, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue671 = claircore.Repository{Name: "cpe:2.3:a:redhat:cert_manager:1.17:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue670}

var rhel_vexValue672 = claircore.Package{Name: "container-native-virtualization/kubevirt-realtime-checkup-rhel9", Kind: 4}

var rhel_vexValue673 = claircore.Package{Name: "openshift-gitops-1/gitops-operator-bundle", Kind: 4}

var rhel_vexValue674 = cpe.Value{V: "openshift_gitops", Kind: 3}

var rhel_vexValue675 = cpe.Value{V: "1", Kind: 3}

var rhel_vexValue676 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue674, rhel_vexValue675, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue677 = claircore.Repository{Name: "cpe:2.3:a:redhat:openshift_gitops:1:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue676}

var rhel_vexValue678 = claircore.Package{Name: "openshift-gitops-1/argocd-rhel9", Kind: 4}

var rhel_vexValue679 = claircore.Package{Name: "openshift-gitops-1/argocd-rhel8", Kind: 4}

var rhel_vexValue680 = cpe.Value{V: "1\\.15", Kind: 3}

var rhel_vexValue681 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue674, rhel_vexValue680, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue682 = claircore.Repository{Name: "cpe:2.3:a:redhat:openshift_gitops:1.15:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue681}

var rhel_vexValue683 = claircore.Package{Name: "openshift4/azure-service-rhel9-operator", Kind: 4}

var rhel_vexValue684 = claircore.Package{Name: "openshift4/lifecycle-agent-operator-bundle", Kind: 4}

var rhel_vexValue685 = claircore.Package{Name: "openshift4/ose-azure-cloud-controller-manager-rhel9", Kind: 4}

var rhel_vexValue686 = claircore.Package{Name: "openshift4/ose-azure-cloud-node-manager-rhel9", Kind: 4}

var rhel_vexValue687 = claircore.Package{Name: "openshift4/ose-cluster-autoscaler-rhel9", Kind: 4}

var rhel_vexValue688 = claircore.Package{Name: "openshift4/ose-cluster-config-api-rhel9", Kind: 4}

var rhel_vexValue689 = claircore.Package{Name: "openshift4/ose-csi-driver-nfs-rhel9", Kind: 4}

var rhel_vexValue690 = claircore.Package{Name: "openshift4/ose-machine-api-provider-azure-rhel9", Kind: 4}

var rhel_vexValue691 = claircore.Package{Name: "openshift4/ose-openstack-cloud-controller-manager-rhel9", Kind: 4}

var rhel_vexValue692 = claircore.Package{Name: "openshift-builds/openshift-builds-controller-rhel9", Kind: 4}

var rhel_vexValue693 = cpe.Value{V: "openshift_builds", Kind: 3}

var rhel_vexValue694 = cpe.Value{V: "1\\.4", Kind: 3}

var rhel_vexValue695 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue693, rhel_vexValue694, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue696 = claircore.Repository{Name: "cpe:2.3:a:redhat:openshift_builds:1.4:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue695}

var rhel_vexValue697 = claircore.Package{Name: "openshift-builds/openshift-builds-git-cloner-rhel9", Kind: 4}

var rhel_vexValue698 = claircore.Package{Name: "openshift-builds/openshift-builds-image-bundler-rhel9", Kind: 4}

var rhel_vexValue699 = claircore.Package{Name: "openshift-builds/openshift-builds-image-processing-rhel9", Kind: 4}

var rhel_vexValue700 = claircore.Package{Name: "openshift-builds/openshift-builds-waiters-rhel9", Kind: 4}

var rhel_vexValue701 = claircore.Package{Name: "openshift-builds/openshift-builds-webhook-rhel9", Kind: 4}

var rhel_vexValue702 = cpe.Value{V: "1\\.5", Kind: 3}

var rhel_vexValue703 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue693, rhel_vexValue702, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue704 = claircore.Repository{Name: "cpe:2.3:a:redhat:openshift_builds:1.5:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue703}

var rhel_vexValue705 = cpe.Value{V: "1\\.6", Kind: 3}

var rhel_vexValue706 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue693, rhel_vexValue705, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue707 = claircore.Repository{Name: "cpe:2.3:a:redhat:openshift_builds:1.6:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue706}

var rhel_vexValue708 = claircore.Package{Name: "rhoso-operators/octavia-operator-bundle", Kind: 4}

var rhel_vexValue709 = claircore.Package{Name: "multicluster-globalhub/multicluster-globalhub-operator-bundle", Kind: 4}

var rhel_vexValue710 = claircore.Package{Name: "rhacm2/acm-flightctl-api-rhel9", Kind: 4}

var rhel_vexValue711 = claircore.Package{Name: "rhacm2/acm-flightctl-periodic-rhel9", Kind: 4}

var rhel_vexValue712 = claircore.Package{Name: "rhacm2/acm-flightctl-worker-rhel9", Kind: 4}

var rhel_vexValue713 = claircore.Package{Name: "rhacm2/volsync-rhel9", Kind: 4}

var rhel_vexValue714 = claircore.Package{Name: "devspaces/devspaces-rhel9-operator", Kind: 4}

var rhel_vexValue715 = cpe.Value{V: "openshift_devspaces", Kind: 3}

var rhel_vexValue716 = cpe.Value{V: "3\\.15", Kind: 3}

var rhel_vexValue717 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue715, rhel_vexValue716, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue718 = claircore.Repository{Name: "cpe:2.3:a:redhat:openshift_devspaces:3.15:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue717}

var rhel_vexValue719 = claircore.Package{Name: "devspaces/traefik-rhel9", Kind: 4}

var rhel_vexValue720 = cpe.Value{V: "3\\.16", Kind: 3}

var rhel_vexValue721 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue715, rhel_vexValue720, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue722 = claircore.Repository{Name: "cpe:2.3:a:redhat:openshift_devspaces:3.16:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue721}

var rhel_vexValue723 = cpe.Value{V: "3\\.17", Kind: 3}

var rhel_vexValue724 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue715, rhel_vexValue723, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue725 = claircore.Repository{Name: "cpe:2.3:a:redhat:openshift_devspaces:3.17:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue724}

var rhel_vexValue726 = claircore.Package{Name: "devspaces/devspaces-rhel8-operator", Kind: 4}

var rhel_vexValue727 = cpe.Value{V: "3\\.18", Kind: 3}

var rhel_vexValue728 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue715, rhel_vexValue727, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue729 = claircore.Repository{Name: "cpe:2.3:a:redhat:openshift_devspaces:3.18:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue728}

var rhel_vexValue730 = claircore.Package{Name: "devspaces/traefik-rhel8", Kind: 4}

var rhel_vexValue731 = claircore.Package{Name: "rhtas/cosign-rhel9", Kind: 4}

var rhel_vexValue732 = cpe.Value{V: "1\\.1", Kind: 3}

var rhel_vexValue733 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue522, rhel_vexValue732, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue734 = claircore.Repository{Name: "cpe:2.3:a:redhat:trusted_artifact_signer:1.1:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue733}

var rhel_vexValue735 = "A flaw was found in nodejs. Affected versions of Node.js allow two copies of a header field in an HTTP request. The first header field is recognized while the second is ignored leading to HTTP request smuggling. The highest threat from this vulnerability is to data confidentiality and integrity."

var rhel_vexValue736 = "https://access.redhat.com/security/cve/CVE-2020-8287 https://nvd.nist.gov/vuln/detail/CVE-2020-8287 https://www.cve.org/CVERecord?id=CVE-2020-8287 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-8287.json https://access.redhat.com/errata/RHSA-2021:0548"

var rhel_vexValue737 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2020-8287"}

var rhel_vexValue738 = claircore.Alias{Space: unique.Make("CVE"), Name: "2020-8287"}

var rhel_vexValue739 = "https://access.redhat.com/security/cve/CVE-2020-8287 https://nvd.nist.gov/vuln/detail/CVE-2020-8287 https://www.cve.org/CVERecord?id=CVE-2020-8287 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-8287.json https://access.redhat.com/errata/RHSA-2021:0549"

var rhel_vexValue740 = "https://access.redhat.com/security/cve/CVE-2020-8287 https://nvd.nist.gov/vuln/detail/CVE-2020-8287 https://www.cve.org/CVERecord?id=CVE-2020-8287 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-8287.json https://access.redhat.com/errata/RHSA-2021:0551"

var rhel_vexValue741 = "https://access.redhat.com/security/cve/CVE-2020-8287 https://nvd.nist.gov/vuln/detail/CVE-2020-8287 https://www.cve.org/CVERecord?id=CVE-2020-8287 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-8287.json"

var rhel_vexValue742 = "The fill_input_buffer function in jdatasrc.c in libjpeg-turbo 1.5.1 allows remote attackers to cause a denial of service (invalid memory access and application crash) or possibly have unspecified other impact via a crafted jpg file. NOTE: Maintainer asserts the issue is due to a bug in downstream code caused by misuse of the libjpeg API"

var rhel_vexValue743 = "https://access.redhat.com/security/cve/CVE-2017-9614 https://nvd.nist.gov/vuln/detail/CVE-2017-9614 https://www.cve.org/CVERecord?id=CVE-2017-9614 https://security.access.redhat.com/data/csaf/v2/vex-feed/2017/cve-2017-9614.json"

var rhel_vexValue744 = claircore.Package{Name: "libjpeg-turbo", Kind: 1}

var rhel_vexValue745 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2017-9614"}

var rhel_vexValue746 = claircore.Alias{Space: unique.Make("CVE"), Name: "2017-9614"}

var rhel_vexValue747 = "A flaw was found in nodejs-ini. If an attacker submits a malicious INI file to an application that parses it with ini.parse, they will pollute the prototype on the application. This can be exploited further depending on the context."

var rhel_vexValue748 = "https://access.redhat.com/security/cve/CVE-2020-7788 https://nvd.nist.gov/vuln/detail/CVE-2020-7788 https://www.cve.org/CVERecord?id=CVE-2020-7788 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-7788.json https://access.redhat.com/errata/RHSA-2021:0548"

var rhel_vexValue749 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2020-7788"}

var rhel_vexValue750 = claircore.Alias{Space: unique.Make("CVE"), Name: "2020-7788"}

var rhel_vexValue751 = "https://access.redhat.com/security/cve/CVE-2020-7788 https://nvd.nist.gov/vuln/detail/CVE-2020-7788 https://www.cve.org/CVERecord?id=CVE-2020-7788 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-7788.json https://access.redhat.com/errata/RHSA-2021:0549"

var rhel_vexValue752 = "https://access.redhat.com/security/cve/CVE-2020-7788 https://nvd.nist.gov/vuln/detail/CVE-2020-7788 https://www.cve.org/CVERecord?id=CVE-2020-7788 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-7788.json https://access.redhat.com/errata/RHSA-2021:0551"

var rhel_vexValue753 = "https://access.redhat.com/security/cve/CVE-2020-7788 https://nvd.nist.gov/vuln/detail/CVE-2020-7788 https://www.cve.org/CVERecord?id=CVE-2020-7788 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-7788.json https://access.redhat.com/errata/RHSA-2022:0246"

var rhel_vexValue754 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue34, rhel_vexValue35, rhel_vexValue7, rhel_vexValue18, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue755 = claircore.Repository{Name: "cpe:2.3:a:redhat:rhel_eus:8.4:*:appstream:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue754}

var rhel_vexValue756 = "https://access.redhat.com/security/cve/CVE-2020-7788 https://nvd.nist.gov/vuln/detail/CVE-2020-7788 https://www.cve.org/CVERecord?id=CVE-2020-7788 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-7788.json"

var rhel_vexValue757 = cpe.Value{V: "rhel_e6s", Kind: 3}

var rhel_vexValue758 = cpe.Value{V: "8\\.10", Kind: 3}

var rhel_vexValue759 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue757, rhel_vexValue758, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue760 = claircore.Repository{Name: "cpe:2.3:a:redhat:rhel_e6s:8.10:*:*:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue759}

var rhel_vexValue761 = claircore.Package{Name: "nodejs", Kind: 1, Module: "nodejs:16"}

var rhel_vexValue762 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue129, rhel_vexValue218, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue763 = claircore.Repository{Name: "cpe:2.3:a:redhat:rhel_aus:8.6:*:*:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue762}

var rhel_vexValue764 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue34, rhel_vexValue594, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue765 = claircore.Repository{Name: "cpe:2.3:a:redhat:rhel_eus:8.8:*:*:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue764}

var rhel_vexValue766 = "A flaw was found in libsolv. A buffer overflow vulnerability in the pool_disabled_solvable function allows attackers to cause a denial of service. The highest threat from this vulnerability is to system availability."

var rhel_vexValue767 = "https://access.redhat.com/security/cve/CVE-2021-33929 https://nvd.nist.gov/vuln/detail/CVE-2021-33929 https://www.cve.org/CVERecord?id=CVE-2021-33929 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-33929.json https://access.redhat.com/errata/RHSA-2021:4060"

var rhel_vexValue768 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2021-33929"}

var rhel_vexValue769 = claircore.Alias{Space: unique.Make("CVE"), Name: "2021-33929"}

var rhel_vexValue770 = "https://access.redhat.com/security/cve/CVE-2021-33929 https://nvd.nist.gov/vuln/detail/CVE-2021-33929 https://www.cve.org/CVERecord?id=CVE-2021-33929 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-33929.json https://access.redhat.com/errata/RHSA-2022:5498"

var rhel_vexValue771 = "https://access.redhat.com/security/cve/CVE-2021-33929 https://nvd.nist.gov/vuln/detail/CVE-2021-33929 https://www.cve.org/CVERecord?id=CVE-2021-33929 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-33929.json"

var rhel_vexValue772 = "A flaw was found in the Curl package. This flaw allows an attacker to insert cookies into a running program using libcurl if the specific series of conditions are met."

var rhel_vexValue773 = "https://access.redhat.com/security/cve/CVE-2023-38546 https://nvd.nist.gov/vuln/detail/CVE-2023-38546 https://www.cve.org/CVERecord?id=CVE-2023-38546 https://security.access.redhat.com/data/csaf/v2/vex-feed/2023/cve-2023-38546.json https://access.redhat.com/errata/RHSA-2023:5700"

var rhel_vexValue774 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2023-38546"}

var rhel_vexValue775 = claircore.Alias{Space: unique.Make("CVE"), Name: "2023-38546"}

var rhel_vexValue776 = "https://access.redhat.com/security/cve/CVE-2023-38546 https://nvd.nist.gov/vuln/detail/CVE-2023-38546 https://www.cve.org/CVERecord?id=CVE-2023-38546 https://security.access.redhat.com/data/csaf/v2/vex-feed/2023/cve-2023-38546.json https://access.redhat.com/errata/RHSA-2023:5763"

var rhel_vexValue777 = "https://access.redhat.com/security/cve/CVE-2023-38546 https://nvd.nist.gov/vuln/detail/CVE-2023-38546 https://www.cve.org/CVERecord?id=CVE-2023-38546 https://security.access.redhat.com/data/csaf/v2/vex-feed/2023/cve-2023-38546.json https://access.redhat.com/errata/RHSA-2023:6292"

var rhel_vexValue778 = "https://access.redhat.com/security/cve/CVE-2023-38546 https://nvd.nist.gov/vuln/detail/CVE-2023-38546 https://www.cve.org/CVERecord?id=CVE-2023-38546 https://security.access.redhat.com/data/csaf/v2/vex-feed/2023/cve-2023-38546.json https://access.redhat.com/errata/RHSA-2023:6745"

var rhel_vexValue779 = "https://access.redhat.com/security/cve/CVE-2023-38546 https://nvd.nist.gov/vuln/detail/CVE-2023-38546 https://www.cve.org/CVERecord?id=CVE-2023-38546 https://security.access.redhat.com/data/csaf/v2/vex-feed/2023/cve-2023-38546.json https://access.redhat.com/errata/RHSA-2023:7540"

var rhel_vexValue780 = "https://access.redhat.com/security/cve/CVE-2023-38546 https://nvd.nist.gov/vuln/detail/CVE-2023-38546 https://www.cve.org/CVERecord?id=CVE-2023-38546 https://security.access.redhat.com/data/csaf/v2/vex-feed/2023/cve-2023-38546.json https://access.redhat.com/errata/RHSA-2024:1601"

var rhel_vexValue781 = "https://access.redhat.com/security/cve/CVE-2023-38546 https://nvd.nist.gov/vuln/detail/CVE-2023-38546 https://www.cve.org/CVERecord?id=CVE-2023-38546 https://security.access.redhat.com/data/csaf/v2/vex-feed/2023/cve-2023-38546.json"

var rhel_vexValue782 = "A flaw was found in nodejs-y18n. There is a prototype pollution vulnerability in y18n's locale functionality. If an attacker is able to provide untrusted input via locale, they may be able to cause denial of service or in rare circumstances, impact to data integrity or confidentiality."

var rhel_vexValue783 = "https://access.redhat.com/security/cve/CVE-2020-7774 https://nvd.nist.gov/vuln/detail/CVE-2020-7774 https://www.cve.org/CVERecord?id=CVE-2020-7774 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-7774.json https://access.redhat.com/errata/RHSA-2020:5499"

var rhel_vexValue784 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2020-7774"}

var rhel_vexValue785 = claircore.Alias{Space: unique.Make("CVE"), Name: "2020-7774"}

var rhel_vexValue786 = "https://access.redhat.com/security/cve/CVE-2020-7774 https://nvd.nist.gov/vuln/detail/CVE-2020-7774 https://www.cve.org/CVERecord?id=CVE-2020-7774 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-7774.json https://access.redhat.com/errata/RHSA-2021:0548"

var rhel_vexValue787 = "https://access.redhat.com/security/cve/CVE-2020-7774 https://nvd.nist.gov/vuln/detail/CVE-2020-7774 https://www.cve.org/CVERecord?id=CVE-2020-7774 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-7774.json https://access.redhat.com/errata/RHSA-2021:0551"

var rhel_vexValue788 = "https://access.redhat.com/security/cve/CVE-2020-7774 https://nvd.nist.gov/vuln/detail/CVE-2020-7774 https://www.cve.org/CVERecord?id=CVE-2020-7774 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-7774.json"

var rhel_vexValue789 = claircore.Package{Name: "openshift-logging/kibana6-rhel8", Kind: 4}

var rhel_vexValue790 = cpe.Value{V: "logging", Kind: 3}

var rhel_vexValue791 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue790, rhel_vexValue180, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue792 = claircore.Repository{Name: "cpe:2.3:a:redhat:logging:5:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue791}

var rhel_vexValue793 = claircore.Package{Name: "rhosdt/jaeger-all-in-one-rhel8", Kind: 4}

var rhel_vexValue794 = cpe.Value{V: "openshift_distributed_tracing", Kind: 3}

var rhel_vexValue795 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue794, rhel_vexValue504, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue796 = claircore.Repository{Name: "cpe:2.3:a:redhat:openshift_distributed_tracing:2:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue795}

var rhel_vexValue797 = "This affects the package npm-user-validate before 1.0.1. The regex that validates user emails took exponentially longer to process long input strings beginning with @ characters."

var rhel_vexValue798 = "https://access.redhat.com/security/cve/CVE-2020-7754 https://nvd.nist.gov/vuln/detail/CVE-2020-7754 https://www.cve.org/CVERecord?id=CVE-2020-7754 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-7754.json https://access.redhat.com/errata/RHSA-2021:0548"

var rhel_vexValue799 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2020-7754"}

var rhel_vexValue800 = claircore.Alias{Space: unique.Make("CVE"), Name: "2020-7754"}

var rhel_vexValue801 = "https://access.redhat.com/security/cve/CVE-2020-7754 https://nvd.nist.gov/vuln/detail/CVE-2020-7754 https://www.cve.org/CVERecord?id=CVE-2020-7754 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-7754.json https://access.redhat.com/errata/RHSA-2021:0549"

var rhel_vexValue802 = "https://access.redhat.com/security/cve/CVE-2020-7754 https://nvd.nist.gov/vuln/detail/CVE-2020-7754 https://www.cve.org/CVERecord?id=CVE-2020-7754 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-7754.json https://access.redhat.com/errata/RHSA-2021:0551"

var rhel_vexValue803 = "https://access.redhat.com/security/cve/CVE-2020-7754 https://nvd.nist.gov/vuln/detail/CVE-2020-7754 https://www.cve.org/CVERecord?id=CVE-2020-7754 https://security.access.redhat.com/data/csaf/v2/vex-feed/2020/cve-2020-7754.json"

var rhel_vexValue804 = "A vulnerability was found in libjpeg-turbo that could allow a remote attacker to execute arbitrary code on the system, which is caused by an integer overflow, which leads to subsequent heap corruption."

var rhel_vexValue805 = "https://access.redhat.com/security/cve/CVE-2019-2201 https://nvd.nist.gov/vuln/detail/CVE-2019-2201 https://www.cve.org/CVERecord?id=CVE-2019-2201 https://security.access.redhat.com/data/csaf/v2/vex-feed/2019/cve-2019-2201.json"

var rhel_vexValue806 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2019-2201"}

var rhel_vexValue807 = claircore.Alias{Space: unique.Make("CVE"), Name: "2019-2201"}

var rhel_vexValue808 = "Missing character filtering has been discovered in Python. When folding a long comment in an email header containing exclusively unfoldable characters, the parenthesis would not be preserved. This could be used for injecting headers into email messages where addresses are user-controlled and not sanitized."

var rhel_vexValue809 = "https://access.redhat.com/security/cve/CVE-2025-11468 https://nvd.nist.gov/vuln/detail/CVE-2025-11468 https://www.cve.org/CVERecord?id=CVE-2025-11468 https://security.access.redhat.com/data/csaf/v2/vex-feed/2025/cve-2025-11468.json https://access.redhat.com/errata/RHSA-2026:7661"

var rhel_vexValue810 = claircore.Package{Name: "python3", Kind: 2}

var rhel_vexValue811 = cpe.Value{V: "hummingbird", Kind: 3}

var rhel_vexValue812 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue811, rhel_vexValue675, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue813 = claircore.Repository{Name: "cpe:2.3:a:redhat:hummingbird:1:*:*:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue812}

var rhel_vexValue814 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2025-11468"}

var rhel_vexValue815 = claircore.Alias{Space: unique.Make("CVE"), Name: "2025-11468"}

var rhel_vexValue816 = "https://access.redhat.com/security/cve/CVE-2025-11468 https://nvd.nist.gov/vuln/detail/CVE-2025-11468 https://www.cve.org/CVERecord?id=CVE-2025-11468 https://security.access.redhat.com/data/csaf/v2/vex-feed/2025/cve-2025-11468.json"

var rhel_vexValue817 = claircore.Package{Name: "python3", Kind: 1}

var rhel_vexValue818 = cpe.Value{V: "enterprise_linux_eus", Kind: 3}

var rhel_vexValue819 = cpe.Value{V: "10\\.0", Kind: 3}

var rhel_vexValue820 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue3, rhel_vexValue4, rhel_vexValue818, rhel_vexValue819, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue821 = claircore.Repository{Name: "cpe:2.3:o:redhat:enterprise_linux_eus:10.0:*:*:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue820}

var rhel_vexValue822 = cpe.Value{V: "10\\.1", Kind: 3}

var rhel_vexValue823 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue3, rhel_vexValue4, rhel_vexValue16, rhel_vexValue822, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue824 = claircore.Repository{Name: "cpe:2.3:o:redhat:enterprise_linux:10.1:*:*:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue823}

var rhel_vexValue825 = cpe.Value{V: "10\\.2", Kind: 3}

var rhel_vexValue826 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue3, rhel_vexValue4, rhel_vexValue16, rhel_vexValue825, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue827 = claircore.Repository{Name: "cpe:2.3:o:redhat:enterprise_linux:10.2:*:*:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue826}

var rhel_vexValue828 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue34, rhel_vexValue271, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue829 = claircore.Repository{Name: "cpe:2.3:a:redhat:rhel_eus:9.0:*:*:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue828}

var rhel_vexValue830 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue34, rhel_vexValue279, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue831 = claircore.Repository{Name: "cpe:2.3:a:redhat:rhel_eus:9.2:*:*:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue830}

var rhel_vexValue832 = cpe.Value{V: "9\\.4", Kind: 3}

var rhel_vexValue833 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue34, rhel_vexValue832, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue834 = claircore.Repository{Name: "cpe:2.3:a:redhat:rhel_eus:9.4:*:*:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue833}

var rhel_vexValue835 = cpe.Value{V: "9\\.6", Kind: 3}

var rhel_vexValue836 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue34, rhel_vexValue835, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue837 = claircore.Repository{Name: "cpe:2.3:a:redhat:rhel_eus:9.6:*:*:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue836}

var rhel_vexValue838 = "A flaw was found in Spring Security. When using RegexRequestMatcher, an easy misconfiguration can bypass some servlet containers. Applications using RegexRequestMatcher with `.` in the regular expression are possibly vulnerable to an authorization bypass."

var rhel_vexValue839 = "https://access.redhat.com/security/cve/CVE-2022-22978 https://nvd.nist.gov/vuln/detail/CVE-2022-22978 https://www.cve.org/CVERecord?id=CVE-2022-22978 https://security.access.redhat.com/data/csaf/v2/vex-feed/2022/cve-2022-22978.json https://access.redhat.com/errata/RHSA-2023:3299"

var rhel_vexValue840 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2022-22978"}

var rhel_vexValue841 = claircore.Alias{Space: unique.Make("CVE"), Name: "2022-22978"}

var rhel_vexValue842 = "https://access.redhat.com/security/cve/CVE-2022-22978 https://nvd.nist.gov/vuln/detail/CVE-2022-22978 https://www.cve.org/CVERecord?id=CVE-2022-22978 https://security.access.redhat.com/data/csaf/v2/vex-feed/2022/cve-2022-22978.json"

var rhel_vexValue843 = claircore.Package{Name: "jenkins", Kind: 1}

var rhel_vexValue844 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue228, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue845 = claircore.Repository{Name: "cpe:2.3:a:redhat:ocp_tools:*:*:*:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue844}

var rhel_vexValue846 = claircore.Repository{Name: "cpe:2.3:a:redhat:openshift:4:*:*:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue332}

var rhel_vexValue847 = "A flaw was found in the SSH channel integrity. By manipulating sequence numbers during the handshake, an attacker can remove the initial messages on the secure channel without causing a MAC failure. For example, an attacker could disable the ping extension and thus disable the new countermeasure in OpenSSH 9.5 against keystroke timing attacks."

var rhel_vexValue848 = "https://access.redhat.com/security/cve/CVE-2023-48795 https://nvd.nist.gov/vuln/detail/CVE-2023-48795 https://www.cve.org/CVERecord?id=CVE-2023-48795 https://security.access.redhat.com/data/csaf/v2/vex-feed/2023/cve-2023-48795.json https://access.redhat.com/errata/RHSA-2024:3634"

var rhel_vexValue849 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue228, rhel_vexValue393, rhel_vexValue7, rhel_vexValue52, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue850 = claircore.Repository{Name: "cpe:2.3:a:redhat:ocp_tools:4.14:*:el8:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue849}

var rhel_vexValue851 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2023-48795"}

var rhel_vexValue852 = claircore.Alias{Space: unique.Make("CVE"), Name: "2023-48795"}

var rhel_vexValue853 = "https://access.redhat.com/security/cve/CVE-2023-48795 https://nvd.nist.gov/vuln/detail/CVE-2023-48795 https://www.cve.org/CVERecord?id=CVE-2023-48795 https://security.access.redhat.com/data/csaf/v2/vex-feed/2023/cve-2023-48795.json https://access.redhat.com/errata/RHSA-2024:3635"

var rhel_vexValue854 = "https://access.redhat.com/security/cve/CVE-2023-48795 https://nvd.nist.gov/vuln/detail/CVE-2023-48795 https://www.cve.org/CVERecord?id=CVE-2023-48795 https://security.access.redhat.com/data/csaf/v2/vex-feed/2023/cve-2023-48795.json https://access.redhat.com/errata/RHSA-2024:3636"

var rhel_vexValue855 = "https://access.redhat.com/security/cve/CVE-2023-48795 https://nvd.nist.gov/vuln/detail/CVE-2023-48795 https://www.cve.org/CVERecord?id=CVE-2023-48795 https://security.access.redhat.com/data/csaf/v2/vex-feed/2023/cve-2023-48795.json https://access.redhat.com/errata/RHSA-2024:4597"

var rhel_vexValue856 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue228, rhel_vexValue396, rhel_vexValue7, rhel_vexValue52, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue857 = claircore.Repository{Name: "cpe:2.3:a:redhat:ocp_tools:4.15:*:el8:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue856}

var rhel_vexValue858 = "https://access.redhat.com/security/cve/CVE-2023-48795 https://nvd.nist.gov/vuln/detail/CVE-2023-48795 https://www.cve.org/CVERecord?id=CVE-2023-48795 https://security.access.redhat.com/data/csaf/v2/vex-feed/2023/cve-2023-48795.json"

var rhel_vexValue859 = claircore.Repository{Name: "cpe:2.3:a:redhat:ocp_tools:*:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue844}

var rhel_vexValue860 = claircore.Package{Name: "aap-cloud-metrics-collector-container", Kind: 4}

var rhel_vexValue861 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue205, rhel_vexValue504, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue862 = claircore.Repository{Name: "cpe:2.3:a:redhat:ansible_automation_platform:2:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue861}

var rhel_vexValue863 = claircore.Package{Name: "ansible-automation-platform-24/ansible-builder-rhel9", Kind: 4}

var rhel_vexValue864 = claircore.Package{Name: "ansible-automation-platform-24/de-supported-rhel8", Kind: 4}

var rhel_vexValue865 = claircore.Package{Name: "ansible-automation-platform-24/ee-dellemc-openmanage-rhel8", Kind: 4}

var rhel_vexValue866 = claircore.Package{Name: "ansible-automation-platform-24/ee-supported-rhel9", Kind: 4}

var rhel_vexValue867 = claircore.Package{Name: "ansible-automation-platform-24/platform-resource-runner-rhel8", Kind: 4}

var rhel_vexValue868 = claircore.Package{Name: "ansible-automation-platform-25/ee-cloud-services-rhel9", Kind: 4}

var rhel_vexValue869 = claircore.Package{Name: "ansible-automation-platform-25/ee-minimal-rhel8", Kind: 4}

var rhel_vexValue870 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue367, rhel_vexValue675, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue871 = claircore.Repository{Name: "cpe:2.3:a:redhat:cert_manager:1:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue870}

var rhel_vexValue872 = claircore.Package{Name: "container-native-virtualization/cluster-network-addons-operator", Kind: 4}

var rhel_vexValue873 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue375, rhel_vexValue331, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue874 = claircore.Repository{Name: "cpe:2.3:a:redhat:container_native_virtualization:4:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue873}

var rhel_vexValue875 = claircore.Package{Name: "openshift4-wincw/windows-machine-config-rhel9-operator", Kind: 4}

var rhel_vexValue876 = claircore.Package{Name: "openshift4/ose-alibaba-cloud-csi-driver-container-rhel8", Kind: 4}

var rhel_vexValue877 = claircore.Package{Name: "openshift4/ose-cli", Kind: 4}

var rhel_vexValue878 = claircore.Package{Name: "openshift4/ose-cli-artifacts", Kind: 4}

var rhel_vexValue879 = claircore.Package{Name: "openshift4/ose-deployer", Kind: 4}

var rhel_vexValue880 = claircore.Package{Name: "openshift4/ose-ibmcloud-cluster-api-controllers-rhel9", Kind: 4}

var rhel_vexValue881 = claircore.Package{Name: "openshift4/ose-node-problem-detector-rhel8", Kind: 4}

var rhel_vexValue882 = claircore.Package{Name: "openshift4/ose-csi-external-provisioner-rhel8", Kind: 4}

var rhel_vexValue883 = claircore.Package{Name: "openshift4/ose-vsphere-csi-driver-syncer-rhel8", Kind: 4}

var rhel_vexValue884 = claircore.Package{Name: "openshift4/egress-router-cni-rhel8", Kind: 4}

var rhel_vexValue885 = claircore.Package{Name: "openshift-builds/openshift-builds-webhook-rhel8", Kind: 4}

var rhel_vexValue886 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue693, rhel_vexValue675, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue887 = claircore.Repository{Name: "cpe:2.3:a:redhat:openshift_builds:1:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue886}

var rhel_vexValue888 = claircore.Package{Name: "ocs4/cephcsi-rhel8", Kind: 4}

var rhel_vexValue889 = claircore.Package{Name: "noobaa-core-container", Kind: 4}

var rhel_vexValue890 = claircore.Package{Name: "odf4/cephcsi-rhel9", Kind: 4}

var rhel_vexValue891 = claircore.Package{Name: "odf4/rook-ceph-rhel9-operator", Kind: 4}

var rhel_vexValue892 = claircore.Package{Name: "discovery-server-container", Kind: 4}

var rhel_vexValue893 = cpe.Value{V: "discovery", Kind: 3}

var rhel_vexValue894 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue893, rhel_vexValue675, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue895 = claircore.Repository{Name: "cpe:2.3:a:redhat:discovery:1:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue894}

var rhel_vexValue896 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue509, rhel_vexValue68, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue897 = claircore.Repository{Name: "cpe:2.3:a:redhat:advanced_cluster_security:3:*:*:*:*:*:*:*", Key: "rhcc-container-repository", CPE: rhel_vexValue896}

var rhel_vexValue898 = "A flaw was found in libsolv. A buffer overflow vulnerability in the prune_to_recommend function allows attackers to cause a denial of service. The highest threat from this vulnerability is to system availability."

var rhel_vexValue899 = "https://access.redhat.com/security/cve/CVE-2021-33938 https://nvd.nist.gov/vuln/detail/CVE-2021-33938 https://www.cve.org/CVERecord?id=CVE-2021-33938 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-33938.json https://access.redhat.com/errata/RHSA-2021:4060"

var rhel_vexValue900 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2021-33938"}

var rhel_vexValue901 = claircore.Alias{Space: unique.Make("CVE"), Name: "2021-33938"}

var rhel_vexValue902 = "https://access.redhat.com/security/cve/CVE-2021-33938 https://nvd.nist.gov/vuln/detail/CVE-2021-33938 https://www.cve.org/CVERecord?id=CVE-2021-33938 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-33938.json https://access.redhat.com/errata/RHSA-2022:5498"

var rhel_vexValue903 = "https://access.redhat.com/security/cve/CVE-2021-33938 https://nvd.nist.gov/vuln/detail/CVE-2021-33938 https://www.cve.org/CVERecord?id=CVE-2021-33938 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-33938.json"

var rhel_vexValue904 = "A flaw was found in OpenSSL. A remote attacker can exploit a stack buffer overflow vulnerability by supplying a crafted Cryptographic Message Syntax (CMS) message with an oversized Initialization Vector (IV) when parsing AuthEnvelopedData structures that use Authenticated Encryption with Associated Data (AEAD) ciphers such as AES-GCM. This can lead to a crash, causing a Denial of Service (DoS), or potentially allow for remote code execution."

var rhel_vexValue905 = "https://access.redhat.com/security/cve/CVE-2025-15467 https://nvd.nist.gov/vuln/detail/CVE-2025-15467 https://www.cve.org/CVERecord?id=CVE-2025-15467 https://security.access.redhat.com/data/csaf/v2/vex-feed/2025/cve-2025-15467.json https://access.redhat.com/errata/RHSA-2026:1472"

var rhel_vexValue906 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2025-15467"}

var rhel_vexValue907 = claircore.Alias{Space: unique.Make("CVE"), Name: "2025-15467"}

var rhel_vexValue908 = "https://access.redhat.com/security/cve/CVE-2025-15467 https://nvd.nist.gov/vuln/detail/CVE-2025-15467 https://www.cve.org/CVERecord?id=CVE-2025-15467 https://security.access.redhat.com/data/csaf/v2/vex-feed/2025/cve-2025-15467.json https://access.redhat.com/errata/RHSA-2026:1473"

var rhel_vexValue909 = "https://access.redhat.com/security/cve/CVE-2025-15467 https://nvd.nist.gov/vuln/detail/CVE-2025-15467 https://www.cve.org/CVERecord?id=CVE-2025-15467 https://security.access.redhat.com/data/csaf/v2/vex-feed/2025/cve-2025-15467.json https://access.redhat.com/errata/RHSA-2026:1496"

var rhel_vexValue910 = "https://access.redhat.com/security/cve/CVE-2025-15467 https://nvd.nist.gov/vuln/detail/CVE-2025-15467 https://www.cve.org/CVERecord?id=CVE-2025-15467 https://security.access.redhat.com/data/csaf/v2/vex-feed/2025/cve-2025-15467.json https://access.redhat.com/errata/RHSA-2026:1503"

var rhel_vexValue911 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue34, rhel_vexValue835, rhel_vexValue7, rhel_vexValue18, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue912 = claircore.Repository{Name: "cpe:2.3:a:redhat:rhel_eus:9.6:*:appstream:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue911}

var rhel_vexValue913 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue3, rhel_vexValue4, rhel_vexValue34, rhel_vexValue835, rhel_vexValue7, rhel_vexValue36, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue914 = claircore.Repository{Name: "cpe:2.3:o:redhat:rhel_eus:9.6:*:baseos:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue913}

var rhel_vexValue915 = "https://access.redhat.com/security/cve/CVE-2025-15467 https://nvd.nist.gov/vuln/detail/CVE-2025-15467 https://www.cve.org/CVERecord?id=CVE-2025-15467 https://security.access.redhat.com/data/csaf/v2/vex-feed/2025/cve-2025-15467.json https://access.redhat.com/errata/RHSA-2026:1519"

var rhel_vexValue916 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue3, rhel_vexValue4, rhel_vexValue34, rhel_vexValue832, rhel_vexValue7, rhel_vexValue36, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue917 = claircore.Repository{Name: "cpe:2.3:o:redhat:rhel_eus:9.4:*:baseos:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue916}

var rhel_vexValue918 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue34, rhel_vexValue832, rhel_vexValue7, rhel_vexValue18, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue919 = claircore.Repository{Name: "cpe:2.3:a:redhat:rhel_eus:9.4:*:appstream:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue918}

var rhel_vexValue920 = "https://access.redhat.com/security/cve/CVE-2025-15467 https://nvd.nist.gov/vuln/detail/CVE-2025-15467 https://www.cve.org/CVERecord?id=CVE-2025-15467 https://security.access.redhat.com/data/csaf/v2/vex-feed/2025/cve-2025-15467.json https://access.redhat.com/errata/RHSA-2026:1594"

var rhel_vexValue921 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue3, rhel_vexValue4, rhel_vexValue162, rhel_vexValue279, rhel_vexValue7, rhel_vexValue36, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue922 = claircore.Repository{Name: "cpe:2.3:o:redhat:rhel_e4s:9.2:*:baseos:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue921}

var rhel_vexValue923 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue162, rhel_vexValue279, rhel_vexValue7, rhel_vexValue18, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue924 = claircore.Repository{Name: "cpe:2.3:a:redhat:rhel_e4s:9.2:*:appstream:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue923}

var rhel_vexValue925 = "https://access.redhat.com/security/cve/CVE-2025-15467 https://nvd.nist.gov/vuln/detail/CVE-2025-15467 https://www.cve.org/CVERecord?id=CVE-2025-15467 https://security.access.redhat.com/data/csaf/v2/vex-feed/2025/cve-2025-15467.json https://access.redhat.com/errata/RHSA-2026:1733"

var rhel_vexValue926 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue3, rhel_vexValue4, rhel_vexValue162, rhel_vexValue271, rhel_vexValue7, rhel_vexValue36, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue927 = claircore.Repository{Name: "cpe:2.3:o:redhat:rhel_e4s:9.0:*:baseos:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue926}

var rhel_vexValue928 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue162, rhel_vexValue271, rhel_vexValue7, rhel_vexValue18, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue929 = claircore.Repository{Name: "cpe:2.3:a:redhat:rhel_e4s:9.0:*:appstream:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue928}

var rhel_vexValue930 = "https://access.redhat.com/security/cve/CVE-2025-15467 https://nvd.nist.gov/vuln/detail/CVE-2025-15467 https://www.cve.org/CVERecord?id=CVE-2025-15467 https://security.access.redhat.com/data/csaf/v2/vex-feed/2025/cve-2025-15467.json https://access.redhat.com/errata/RHSA-2026:7261"

var rhel_vexValue931 = "A flaw was found in the JUnit Jenkins plugin. The manipulation with an unknown input leads to a Cross-site scripting vulnerability, impacting the integrity. This flaw allows an attacker to inject arbitrary HTML and script code into the website."

var rhel_vexValue932 = "https://access.redhat.com/security/cve/CVE-2022-34176 https://nvd.nist.gov/vuln/detail/CVE-2022-34176 https://www.cve.org/CVERecord?id=CVE-2022-34176 https://security.access.redhat.com/data/csaf/v2/vex-feed/2022/cve-2022-34176.json https://access.redhat.com/errata/RHBA-2022:8582"

var rhel_vexValue933 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2022-34176"}

var rhel_vexValue934 = claircore.Alias{Space: unique.Make("CVE"), Name: "2022-34176"}

var rhel_vexValue935 = "https://access.redhat.com/security/cve/CVE-2022-34176 https://nvd.nist.gov/vuln/detail/CVE-2022-34176 https://www.cve.org/CVERecord?id=CVE-2022-34176 https://security.access.redhat.com/data/csaf/v2/vex-feed/2022/cve-2022-34176.json https://access.redhat.com/errata/RHSA-2022:6531"

var rhel_vexValue936 = "https://access.redhat.com/security/cve/CVE-2022-34176 https://nvd.nist.gov/vuln/detail/CVE-2022-34176 https://www.cve.org/CVERecord?id=CVE-2022-34176 https://security.access.redhat.com/data/csaf/v2/vex-feed/2022/cve-2022-34176.json https://access.redhat.com/errata/RHSA-2023:0017"

var rhel_vexValue937 = "https://access.redhat.com/security/cve/CVE-2022-34176 https://nvd.nist.gov/vuln/detail/CVE-2022-34176 https://www.cve.org/CVERecord?id=CVE-2022-34176 https://security.access.redhat.com/data/csaf/v2/vex-feed/2022/cve-2022-34176.json"

var rhel_vexValue938 = "A flaw was found in the EventSource NPM Package. The description from the source states the following message: \"Exposure of Sensitive Information to an Unauthorized Actor.\" This flaw allows an attacker to steal the user's credentials and then use the credentials to access the legitimate website."

var rhel_vexValue939 = "https://access.redhat.com/security/cve/CVE-2022-1650 https://nvd.nist.gov/vuln/detail/CVE-2022-1650 https://www.cve.org/CVERecord?id=CVE-2022-1650 https://security.access.redhat.com/data/csaf/v2/vex-feed/2022/cve-2022-1650.json https://access.redhat.com/errata/RHBA-2022:5747"

var rhel_vexValue940 = claircore.Package{Name: "dotnet-runtime-6.0", Kind: 2}

var rhel_vexValue941 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2022-1650"}

var rhel_vexValue942 = claircore.Alias{Space: unique.Make("CVE"), Name: "2022-1650"}

var rhel_vexValue943 = claircore.Package{Name: "dotnet", Kind: 2}

var rhel_vexValue944 = claircore.Package{Name: "aspnetcore-runtime-6.0", Kind: 2}

var rhel_vexValue945 = claircore.Package{Name: "dotnet6.0", Kind: 2}

var rhel_vexValue946 = "https://access.redhat.com/security/cve/CVE-2022-1650 https://nvd.nist.gov/vuln/detail/CVE-2022-1650 https://www.cve.org/CVERecord?id=CVE-2022-1650 https://security.access.redhat.com/data/csaf/v2/vex-feed/2022/cve-2022-1650.json https://access.redhat.com/errata/RHBA-2022:5749"

var rhel_vexValue947 = "https://access.redhat.com/security/cve/CVE-2022-1650 https://nvd.nist.gov/vuln/detail/CVE-2022-1650 https://www.cve.org/CVERecord?id=CVE-2022-1650 https://security.access.redhat.com/data/csaf/v2/vex-feed/2022/cve-2022-1650.json https://access.redhat.com/errata/RHSA-2022:6057"

var rhel_vexValue948 = "https://access.redhat.com/security/cve/CVE-2022-1650 https://nvd.nist.gov/vuln/detail/CVE-2022-1650 https://www.cve.org/CVERecord?id=CVE-2022-1650 https://security.access.redhat.com/data/csaf/v2/vex-feed/2022/cve-2022-1650.json"

var rhel_vexValue949 = claircore.Package{Name: "rhacm2/console-rhel8", Kind: 4}

var rhel_vexValue950 = claircore.Package{Name: "rhacm2/kui-web-terminal-rhel8", Kind: 4}

var rhel_vexValue951 = claircore.Package{Name: "rhacm2/search-ui-rhel8", Kind: 4}

var rhel_vexValue952 = "libxslt through 1.1.33 allows bypass of a protection mechanism because callers of xsltCheckRead and xsltCheckWrite permit access even upon receiving a -1 error code. xsltCheckRead can return -1 for a crafted URL that is not actually invalid and is subsequently loaded."

var rhel_vexValue953 = "https://access.redhat.com/security/cve/CVE-2019-11068 https://nvd.nist.gov/vuln/detail/CVE-2019-11068 https://www.cve.org/CVERecord?id=CVE-2019-11068 https://security.access.redhat.com/data/csaf/v2/vex-feed/2019/cve-2019-11068.json https://access.redhat.com/errata/RHSA-2020:4005"

var rhel_vexValue954 = claircore.Package{Name: "libxslt", Kind: 2}

var rhel_vexValue955 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2019-11068"}

var rhel_vexValue956 = claircore.Alias{Space: unique.Make("CVE"), Name: "2019-11068"}

var rhel_vexValue957 = "https://access.redhat.com/security/cve/CVE-2019-11068 https://nvd.nist.gov/vuln/detail/CVE-2019-11068 https://www.cve.org/CVERecord?id=CVE-2019-11068 https://security.access.redhat.com/data/csaf/v2/vex-feed/2019/cve-2019-11068.json https://access.redhat.com/errata/RHSA-2020:4464"

var rhel_vexValue958 = "https://access.redhat.com/security/cve/CVE-2019-11068 https://nvd.nist.gov/vuln/detail/CVE-2019-11068 https://www.cve.org/CVERecord?id=CVE-2019-11068 https://security.access.redhat.com/data/csaf/v2/vex-feed/2019/cve-2019-11068.json"

var rhel_vexValue959 = claircore.Package{Name: "libxslt", Kind: 1}

var rhel_vexValue960 = cpe.Value{V: "10", Kind: 3}

var rhel_vexValue961 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue472, rhel_vexValue960, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue962 = claircore.Repository{Name: "cpe:2.3:a:redhat:openstack:10:*:*:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue961}

var rhel_vexValue963 = cpe.Value{V: "13", Kind: 3}

var rhel_vexValue964 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue472, rhel_vexValue963, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue965 = claircore.Repository{Name: "cpe:2.3:a:redhat:openstack:13:*:*:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue964}

var rhel_vexValue966 = cpe.Value{V: "14", Kind: 3}

var rhel_vexValue967 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue472, rhel_vexValue966, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue968 = claircore.Repository{Name: "cpe:2.3:a:redhat:openstack:14:*:*:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue967}

var rhel_vexValue969 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue472, rhel_vexValue133, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue970 = claircore.Repository{Name: "cpe:2.3:a:redhat:openstack:9:*:*:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue969}

var rhel_vexValue971 = cpe.Value{V: "storage", Kind: 3}

var rhel_vexValue972 = cpe.WFN{Attr: [11]cpe.Value{rhel_vexValue15, rhel_vexValue4, rhel_vexValue971, rhel_vexValue68, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7, rhel_vexValue7}}

var rhel_vexValue973 = claircore.Repository{Name: "cpe:2.3:a:redhat:storage:3:*:*:*:*:*:*:*", Key: "rhel-cpe-repository", CPE: rhel_vexValue972}

var rhel_vexValue974 = "A flaw was found in vim. The vulnerability occurs due to Illegal memory access and leads to an out-of-bounds write vulnerability in the ex_cmds function. This flaw allows an attacker to input a specially crafted file, leading to a crash or code execution."

var rhel_vexValue975 = "https://access.redhat.com/security/cve/CVE-2022-1785 https://nvd.nist.gov/vuln/detail/CVE-2022-1785 https://www.cve.org/CVERecord?id=CVE-2022-1785 https://security.access.redhat.com/data/csaf/v2/vex-feed/2022/cve-2022-1785.json https://access.redhat.com/errata/RHSA-2022:5813"

var rhel_vexValue976 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2022-1785"}

var rhel_vexValue977 = claircore.Alias{Space: unique.Make("CVE"), Name: "2022-1785"}

var rhel_vexValue978 = "https://access.redhat.com/security/cve/CVE-2022-1785 https://nvd.nist.gov/vuln/detail/CVE-2022-1785 https://www.cve.org/CVERecord?id=CVE-2022-1785 https://security.access.redhat.com/data/csaf/v2/vex-feed/2022/cve-2022-1785.json https://access.redhat.com/errata/RHSA-2022:5942"

var rhel_vexValue979 = "https://access.redhat.com/security/cve/CVE-2022-1785 https://nvd.nist.gov/vuln/detail/CVE-2022-1785 https://www.cve.org/CVERecord?id=CVE-2022-1785 https://security.access.redhat.com/data/csaf/v2/vex-feed/2022/cve-2022-1785.json"

var rhel_vexValue980 = "A flaw was found in nodejs. A denial of service is possible when the whitelist includes “localhost6”. When “localhost6” is not present in /etc/hosts, it is just an ordinary domain that is resolved via DNS over the network. If the attacker controls the victim's DNS server or can spoof its responses, the DNS rebinding protection can be bypassed by using the “localhost6” domain. As long as the attacker uses the “localhost6” domain, they can still apply the attack described in CVE-2018-7160."

var rhel_vexValue981 = "https://access.redhat.com/security/cve/CVE-2021-22884 https://nvd.nist.gov/vuln/detail/CVE-2021-22884 https://www.cve.org/CVERecord?id=CVE-2021-22884 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-22884.json https://access.redhat.com/errata/RHSA-2021:0734"

var rhel_vexValue982 = claircore.Alias{Space: unique.Make("https://access.redhat.com/security/cve/"), Name: "CVE-2021-22884"}

var rhel_vexValue983 = claircore.Alias{Space: unique.Make("CVE"), Name: "2021-22884"}

var rhel_vexValue984 = "https://access.redhat.com/security/cve/CVE-2021-22884 https://nvd.nist.gov/vuln/detail/CVE-2021-22884 https://www.cve.org/CVERecord?id=CVE-2021-22884 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-22884.json https://access.redhat.com/errata/RHSA-2021:0735"

var rhel_vexValue985 = "https://access.redhat.com/security/cve/CVE-2021-22884 https://nvd.nist.gov/vuln/detail/CVE-2021-22884 https://www.cve.org/CVERecord?id=CVE-2021-22884 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-22884.json https://access.redhat.com/errata/RHSA-2021:0738"

var rhel_vexValue986 = "https://access.redhat.com/security/cve/CVE-2021-22884 https://nvd.nist.gov/vuln/detail/CVE-2021-22884 https://www.cve.org/CVERecord?id=CVE-2021-22884 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-22884.json https://access.redhat.com/errata/RHSA-2021:0739"

var rhel_vexValue987 = "https://access.redhat.com/security/cve/CVE-2021-22884 https://nvd.nist.gov/vuln/detail/CVE-2021-22884 https://www.cve.org/CVERecord?id=CVE-2021-22884 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-22884.json https://access.redhat.com/errata/RHSA-2021:0740"

var rhel_vexValue988 = "https://access.redhat.com/security/cve/CVE-2021-22884 https://nvd.nist.gov/vuln/detail/CVE-2021-22884 https://www.cve.org/CVERecord?id=CVE-2021-22884 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-22884.json https://access.redhat.com/errata/RHSA-2021:0741"

var rhel_vexValue989 = "https://access.redhat.com/security/cve/CVE-2021-22884 https://nvd.nist.gov/vuln/detail/CVE-2021-22884 https://www.cve.org/CVERecord?id=CVE-2021-22884 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-22884.json https://access.redhat.com/errata/RHSA-2021:0744"

var rhel_vexValue990 = "https://access.redhat.com/security/cve/CVE-2021-22884 https://nvd.nist.gov/vuln/detail/CVE-2021-22884 https://www.cve.org/CVERecord?id=CVE-2021-22884 https://security.access.redhat.com/data/csaf/v2/vex-feed/2021/cve-2021-22884.json"
