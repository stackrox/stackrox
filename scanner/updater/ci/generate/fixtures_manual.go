package main

import (
	time "time"

	claircore "github.com/quay/claircore"
)

// manual.json.zst native fixture relationships, transcribed from the checked-in CI bundle.
// Kept independent from expected results in scanner/e2etests and backend QA.
// Scenarios: TestImage Spring/Log4j/Drools/Tomcat/Jackson images, including
// sandbox-scannerremovejar and spring-CVE-2022-22978. These are the existing
// manual updater relationships, not additional CVEs invented for count checks.
var manualFixtures = []operation{
	{Member: "manual.json.zst", Updater: "stackrox-manual", Vulnerabilities: []*claircore.Vulnerability{
		{Updater: "stackrox-manual", Name: "CVE-2022-22963", Description: "Spring Cloud Function Code Injection with a specially crafted SpEL as a routing expression", Issued: time.Date(2022, 4, 3, 0, 0, 59, 0, time.UTC), Links: "https://nvd.nist.gov/vuln/detail/CVE-2022-22963", Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 5, Package: &manualValue0, Repo: &manualValue1, FixedInVersion: "introduced=0&fixed=3.1.7"},
		{Updater: "stackrox-manual", Name: "CVE-2022-22963", Description: "Spring Cloud Function Code Injection with a specially crafted SpEL as a routing expression", Issued: time.Date(2022, 4, 3, 0, 0, 59, 0, time.UTC), Links: "https://nvd.nist.gov/vuln/detail/CVE-2022-22963", Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 5, Package: &manualValue0, Repo: &manualValue1, FixedInVersion: "introduced=3.2.0&fixed=3.2.3"},
		{Updater: "stackrox-manual", Name: "CVE-2022-22965", Description: "Remote Code Execution in Spring Framework", Issued: time.Date(2022, 3, 31, 18, 30, 50, 0, time.UTC), Links: "https://nvd.nist.gov/vuln/detail/CVE-2022-22965", Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 5, Package: &manualValue2, Repo: &manualValue1, FixedInVersion: "introduced=0&fixed=5.2.20.RELEASE"},
		{Updater: "stackrox-manual", Name: "CVE-2022-22965", Description: "Remote Code Execution in Spring Framework", Issued: time.Date(2022, 3, 31, 18, 30, 50, 0, time.UTC), Links: "https://nvd.nist.gov/vuln/detail/CVE-2022-22965", Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 5, Package: &manualValue2, Repo: &manualValue1, FixedInVersion: "introduced=5.3.0&fixed=5.3.18"},
		{Updater: "stackrox-manual", Name: "CVE-2022-22965", Description: "Remote Code Execution in Spring Framework", Issued: time.Date(2022, 3, 31, 18, 30, 50, 0, time.UTC), Links: "https://nvd.nist.gov/vuln/detail/CVE-2022-22965", Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 5, Package: &manualValue3, Repo: &manualValue1, FixedInVersion: "introduced=0&fixed=5.2.20.RELEASE"},
		{Updater: "stackrox-manual", Name: "CVE-2022-22965", Description: "Remote Code Execution in Spring Framework", Issued: time.Date(2022, 3, 31, 18, 30, 50, 0, time.UTC), Links: "https://nvd.nist.gov/vuln/detail/CVE-2022-22965", Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 5, Package: &manualValue3, Repo: &manualValue1, FixedInVersion: "introduced=5.3.0&fixed=5.3.18"},
		{Updater: "stackrox-manual", Name: "CVE-2022-22965", Description: "Remote Code Execution in Spring Framework", Issued: time.Date(2022, 3, 31, 18, 30, 50, 0, time.UTC), Links: "https://nvd.nist.gov/vuln/detail/CVE-2022-22965", Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 5, Package: &manualValue4, Repo: &manualValue1, FixedInVersion: "introduced=0&fixed=2.5.12"},
		{Updater: "stackrox-manual", Name: "CVE-2022-22965", Description: "Remote Code Execution in Spring Framework", Issued: time.Date(2022, 3, 31, 18, 30, 50, 0, time.UTC), Links: "https://nvd.nist.gov/vuln/detail/CVE-2022-22965", Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 5, Package: &manualValue4, Repo: &manualValue1, FixedInVersion: "introduced=2.6.0&fixed=2.6.6"},
		{Updater: "stackrox-manual", Name: "CVE-2022-22965", Description: "Remote Code Execution in Spring Framework", Issued: time.Date(2022, 3, 31, 18, 30, 50, 0, time.UTC), Links: "https://nvd.nist.gov/vuln/detail/CVE-2022-22965", Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 5, Package: &manualValue5, Repo: &manualValue1, FixedInVersion: "introduced=0&fixed=5.2.20.RELEASE"},
		{Updater: "stackrox-manual", Name: "CVE-2022-22965", Description: "Remote Code Execution in Spring Framework", Issued: time.Date(2022, 3, 31, 18, 30, 50, 0, time.UTC), Links: "https://nvd.nist.gov/vuln/detail/CVE-2022-22965", Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 5, Package: &manualValue5, Repo: &manualValue1, FixedInVersion: "introduced=5.3.0&fixed=5.3.18"},
		{Updater: "stackrox-manual", Name: "CVE-2022-22965", Description: "Remote Code Execution in Spring Framework", Issued: time.Date(2022, 3, 31, 18, 30, 50, 0, time.UTC), Links: "https://nvd.nist.gov/vuln/detail/CVE-2022-22965", Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 5, Package: &manualValue6, Repo: &manualValue1, FixedInVersion: "introduced=0&fixed=2.5.12"},
		{Updater: "stackrox-manual", Name: "CVE-2022-22965", Description: "Remote Code Execution in Spring Framework", Issued: time.Date(2022, 3, 31, 18, 30, 50, 0, time.UTC), Links: "https://nvd.nist.gov/vuln/detail/CVE-2022-22965", Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 5, Package: &manualValue6, Repo: &manualValue1, FixedInVersion: "introduced=2.6.0&fixed=2.6.6"},
		{Updater: "stackrox-manual", Name: "CVE-2022-22978", Description: "Authorization bypass in Spring Security", Issued: time.Date(2022, 5, 20, 0, 0, 39, 0, time.UTC), Links: "https://nvd.nist.gov/vuln/detail/CVE-2022-22978", Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 5, Package: &manualValue7, Repo: &manualValue1, FixedInVersion: "introduced=0&fixed=5.5.7"},
		{Updater: "stackrox-manual", Name: "CVE-2022-22978", Description: "Authorization bypass in Spring Security", Issued: time.Date(2022, 5, 20, 0, 0, 39, 0, time.UTC), Links: "https://nvd.nist.gov/vuln/detail/CVE-2022-22978", Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 5, Package: &manualValue7, Repo: &manualValue1, FixedInVersion: "introduced=5.6.0&fixed=5.6.4"},
		{Updater: "stackrox-manual", Name: "CVE-2022-29885", Description: manualValue8, Issued: time.Date(2022, 5, 12, 8, 15, 7, 0, time.UTC), Links: "https://nvd.nist.gov/vuln/detail/CVE-2022-29885", Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", Package: &manualValue9, Repo: &manualValue1, FixedInVersion: "introduced=8.5.38&fixed=8.5.79"},
		{Updater: "stackrox-manual", Name: "CVE-2022-29885", Description: manualValue8, Issued: time.Date(2022, 5, 12, 8, 15, 7, 0, time.UTC), Links: "https://nvd.nist.gov/vuln/detail/CVE-2022-29885", Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", Package: &manualValue9, Repo: &manualValue1, FixedInVersion: "introduced=9.0.13&fixed=9.0.63"},
		{Updater: "stackrox-manual", Name: "CVE-2022-29885", Description: manualValue8, Issued: time.Date(2022, 5, 12, 8, 15, 7, 0, time.UTC), Links: "https://nvd.nist.gov/vuln/detail/CVE-2022-29885", Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", Package: &manualValue9, Repo: &manualValue1, FixedInVersion: "introduced=10.0.0&fixed=10.0.21"},
		{Updater: "stackrox-manual", Name: "CVE-2023-28708", Description: manualValue10, Issued: time.Date(2023, 3, 22, 11, 15, 10, 0, time.UTC), Links: "https://nvd.nist.gov/vuln/detail/CVE-2023-28708", Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:U/C:L/I:N/A:N", NormalizedSeverity: 3, Package: &manualValue9, Repo: &manualValue1, FixedInVersion: "introduced=8.5.0&fixed=8.5.86"},
		{Updater: "stackrox-manual", Name: "CVE-2023-28708", Description: manualValue10, Issued: time.Date(2023, 3, 22, 11, 15, 10, 0, time.UTC), Links: "https://nvd.nist.gov/vuln/detail/CVE-2023-28708", Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:U/C:L/I:N/A:N", NormalizedSeverity: 3, Package: &manualValue9, Repo: &manualValue1, FixedInVersion: "introduced=9.0.0&fixed=9.0.72"},
		{Updater: "stackrox-manual", Name: "CVE-2023-28708", Description: manualValue10, Issued: time.Date(2023, 3, 22, 11, 15, 10, 0, time.UTC), Links: "https://nvd.nist.gov/vuln/detail/CVE-2023-28708", Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:U/C:L/I:N/A:N", NormalizedSeverity: 3, Package: &manualValue9, Repo: &manualValue1, FixedInVersion: "introduced=10.1.0&fixed=10.1.6"},
		{Updater: "stackrox-manual", Name: "CVE-2025-24813", Description: "Apache Tomcat: Potential RCE and/or information disclosure and/or information corruption with partial PUT", Issued: time.Date(2025, 3, 10, 18, 31, 56, 0, time.UTC), Links: "https://nvd.nist.gov/vuln/detail/CVE-2025-24813", Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 5, Package: &manualValue11, Repo: &manualValue1, FixedInVersion: "introduced=9.0.0.M1&fixed=9.0.99"},
		{Updater: "stackrox-manual", Name: "CVE-2025-24813", Description: "Apache Tomcat: Potential RCE and/or information disclosure and/or information corruption with partial PUT", Issued: time.Date(2025, 3, 10, 18, 31, 56, 0, time.UTC), Links: "https://nvd.nist.gov/vuln/detail/CVE-2025-24813", Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 5, Package: &manualValue9, Repo: &manualValue1, FixedInVersion: "introduced=9.0.0.M1&fixed=9.0.99"},
		{Updater: "stackrox-manual", Name: "CVE-2025-24813", Description: "Apache Tomcat: Potential RCE and/or information disclosure and/or information corruption with partial PUT", Issued: time.Date(2025, 3, 10, 18, 31, 56, 0, time.UTC), Links: "https://nvd.nist.gov/vuln/detail/CVE-2025-24813", Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 5, Package: &manualValue9, Repo: &manualValue1, FixedInVersion: "introduced=10.1.0-M1&fixed=10.1.35"},
		{Updater: "stackrox-manual", Name: "CVE-2025-24813", Description: "Apache Tomcat: Potential RCE and/or information disclosure and/or information corruption with partial PUT", Issued: time.Date(2025, 3, 10, 18, 31, 56, 0, time.UTC), Links: "https://nvd.nist.gov/vuln/detail/CVE-2025-24813", Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", NormalizedSeverity: 5, Package: &manualValue9, Repo: &manualValue1, FixedInVersion: "introduced=11.0.0-M1&fixed=11.0.3"},
	}},
}
var manualValue0 = claircore.Package{Name: "spring-cloud-function-context", Kind: 2}

var manualValue1 = claircore.Repository{Name: "maven", URI: "https://repo1.maven.apache.org/maven2"}

var manualValue2 = claircore.Package{Name: "spring-beans", Kind: 2}

var manualValue3 = claircore.Package{Name: "spring-webmvc", Kind: 2}

var manualValue4 = claircore.Package{Name: "spring-boot-starter-web", Kind: 2}

var manualValue5 = claircore.Package{Name: "spring-webflux", Kind: 2}

var manualValue6 = claircore.Package{Name: "spring-boot-starter-webflux", Kind: 2}

var manualValue7 = claircore.Package{Name: "spring-security-core", Kind: 2}

var manualValue8 = "The documentation of Apache Tomcat 10.1.0-M1 to 10.1.0-M14, 10.0.0-M1 to 10.0.20, 9.0.13 to 9.0.62 and 8.5.38 to 8.5.78 for the EncryptInterceptor incorrectly stated it enabled Tomcat clustering to run over an untrusted network. This was not correct. While the EncryptInterceptor does provide confidentiality and integrity protection, it does not protect against all risks associated with running over any untrusted network, particularly DoS risks."

var manualValue9 = claircore.Package{Name: "org.apache.tomcat-embed-core:tomcat-embed-core", Kind: 2}

var manualValue10 = "When using the RemoteIpFilter with requests received from a reverse proxy via HTTP that include the X-Forwarded-Proto header set to https, session cookies created by Apache Tomcat 11.0.0-M1 to 11.0.0.-M2, 10.1.0-M1 to 10.1.5, 9.0.0-M1 to 9.0.71 and 8.5.0 to 8.5.85 did not include the secure attribute. This could result in the user agent transmitting the session cookie over an insecure channel."

var manualValue11 = claircore.Package{Name: "tomcat-embed-core", Kind: 2}
