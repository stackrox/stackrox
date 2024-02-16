package manual

import (
	"time"

	"github.com/quay/claircore"
	"github.com/stackrox/rox/pkg/utils"
)

// vulns returns vulnerabilities not tracked by other means.
func (u *updater) vulns() []*claircore.Vulnerability {
	return []*claircore.Vulnerability{
		{
			// Vuln: CVE-2022-22963/GHSA-6v73-fgf6-w5j7
			// Reason: The vuln table has an entry for GHSA-6v73-fgf6-w5j7, but Scanner V4
			// may have trouble determining the groupID when pom.properties is missing.
			// Source: https://osv-vulnerabilities.storage.googleapis.com/Maven/GHSA-6v73-fgf6-w5j7.json
			Updater:            u.Name(),
			Name:               "CVE-2022-22963",
			Description:        "Spring Cloud Function Code Injection with a specially crafted SpEL as a routing expression",
			Issued:             mustParseTime(time.RFC3339, "2022-04-03T00:00:59Z"),
			Links:              "https://nvd.nist.gov/vuln/detail/CVE-2022-22963",
			Severity:           "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
			NormalizedSeverity: claircore.Critical,
			Package: &claircore.Package{
				Name:           "spring-cloud-function-context",
				Kind:           claircore.BINARY,
				RepositoryHint: "Maven",
			},
			FixedInVersion: "introduced=0&fixed=3.1.7",
			Repo: &claircore.Repository{
				Name: "maven",
				URI:  "https://repo1.maven.apache.org/maven2",
			},
		},
		{
			// Vuln: CVE-2022-22963/GHSA-6v73-fgf6-w5j7
			// Reason: Same as previous entry but with different vulnerable range.
			// Source: https://osv-vulnerabilities.storage.googleapis.com/Maven/GHSA-6v73-fgf6-w5j7.json
			Updater:            u.Name(),
			Name:               "CVE-2022-22963",
			Description:        "Spring Cloud Function Code Injection with a specially crafted SpEL as a routing expression",
			Issued:             mustParseTime(time.RFC3339, "2022-04-03T00:00:59Z"),
			Links:              "https://nvd.nist.gov/vuln/detail/CVE-2022-22963",
			Severity:           "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
			NormalizedSeverity: claircore.Critical,
			Package: &claircore.Package{
				Name:           "spring-cloud-function-context",
				Kind:           claircore.BINARY,
				RepositoryHint: "Maven",
			},
			FixedInVersion: "introduced=3.2.0&fixed=3.2.3",
			Repo: &claircore.Repository{
				Name: "maven",
				URI:  "https://repo1.maven.apache.org/maven2",
			},
		},

		{
			// Vuln: CVE-2022-22965/GHSA-36p3-wjmg-h94x (Spring4Shell)
			// Reason: The vuln table has an entry for GHSA-36p3-wjmg-h94x, but Scanner V4
			// may have trouble determining the groupID when pom.properties is missing.
			// Source: https://osv-vulnerabilities.storage.googleapis.com/Maven/GHSA-36p3-wjmg-h94x.json
			Updater:            u.Name(),
			Name:               "CVE-2022-22965",
			Description:        "Remote Code Execution in Spring Framework",
			Issued:             mustParseTime(time.RFC3339, "2022-03-31T18:30:50Z"),
			Links:              "https://nvd.nist.gov/vuln/detail/CVE-2022-22965",
			Severity:           "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
			NormalizedSeverity: claircore.Critical,
			Package: &claircore.Package{
				Name:           "spring-beans",
				Kind:           claircore.BINARY,
				RepositoryHint: "Maven",
			},
			FixedInVersion: "introduced=0&fixed=5.2.20.RELEASE",
			Repo: &claircore.Repository{
				Name: "maven",
				URI:  "https://repo1.maven.apache.org/maven2",
			},
		},
		{
			// Vuln: CVE-2022-22965/GHSA-36p3-wjmg-h94x (Spring4Shell)
			// Reason: Same as previous entry but with different vulnerable range.
			// Source: https://osv-vulnerabilities.storage.googleapis.com/Maven/GHSA-36p3-wjmg-h94x.json
			Updater:            u.Name(),
			Name:               "CVE-2022-22965",
			Description:        "Remote Code Execution in Spring Framework",
			Issued:             mustParseTime(time.RFC3339, "2022-03-31T18:30:50Z"),
			Links:              "https://nvd.nist.gov/vuln/detail/CVE-2022-22965",
			Severity:           "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
			NormalizedSeverity: claircore.Critical,
			Package: &claircore.Package{
				Name:           "spring-beans",
				Kind:           claircore.BINARY,
				RepositoryHint: "Maven",
			},
			FixedInVersion: "introduced=5.3.0&fixed=5.3.18",
			Repo: &claircore.Repository{
				Name: "maven",
				URI:  "https://repo1.maven.apache.org/maven2",
			},
		},
		{
			// Vuln: CVE-2022-22965/GHSA-36p3-wjmg-h94x (Spring4Shell)
			// Reason: Same as previous entries but with different package name.
			// Source: https://osv-vulnerabilities.storage.googleapis.com/Maven/GHSA-36p3-wjmg-h94x.json
			Updater:            u.Name(),
			Name:               "CVE-2022-22965",
			Description:        "Remote Code Execution in Spring Framework",
			Issued:             mustParseTime(time.RFC3339, "2022-03-31T18:30:50Z"),
			Links:              "https://nvd.nist.gov/vuln/detail/CVE-2022-22965",
			Severity:           "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
			NormalizedSeverity: claircore.Critical,
			Package: &claircore.Package{
				Name:           "spring-webmvc",
				Kind:           claircore.BINARY,
				RepositoryHint: "Maven",
			},
			FixedInVersion: "introduced=0&fixed=5.2.20.RELEASE",
			Repo: &claircore.Repository{
				Name: "maven",
				URI:  "https://repo1.maven.apache.org/maven2",
			},
		},
		{
			// Vuln: CVE-2022-22965/GHSA-36p3-wjmg-h94x (Spring4Shell)
			// Reason: Same as previous entry but with different vulnerable range.
			// Source: https://osv-vulnerabilities.storage.googleapis.com/Maven/GHSA-36p3-wjmg-h94x.json
			Updater:            u.Name(),
			Name:               "CVE-2022-22965",
			Description:        "Remote Code Execution in Spring Framework",
			Issued:             mustParseTime(time.RFC3339, "2022-03-31T18:30:50Z"),
			Links:              "https://nvd.nist.gov/vuln/detail/CVE-2022-22965",
			Severity:           "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
			NormalizedSeverity: claircore.Critical,
			Package: &claircore.Package{
				Name:           "spring-webmvc",
				Kind:           claircore.BINARY,
				RepositoryHint: "Maven",
			},
			FixedInVersion: "introduced=5.3.0&fixed=5.3.18",
			Repo: &claircore.Repository{
				Name: "maven",
				URI:  "https://repo1.maven.apache.org/maven2",
			},
		},
		{
			// Vuln: CVE-2022-22965/GHSA-36p3-wjmg-h94x (Spring4Shell)
			// Reason: Same as previous entries but with different package name.
			// Source: https://osv-vulnerabilities.storage.googleapis.com/Maven/GHSA-36p3-wjmg-h94x.json
			Updater:            u.Name(),
			Name:               "CVE-2022-22965",
			Description:        "Remote Code Execution in Spring Framework",
			Issued:             mustParseTime(time.RFC3339, "2022-03-31T18:30:50Z"),
			Links:              "https://nvd.nist.gov/vuln/detail/CVE-2022-22965",
			Severity:           "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
			NormalizedSeverity: claircore.Critical,
			Package: &claircore.Package{
				Name:           "spring-boot-starter-web",
				Kind:           claircore.BINARY,
				RepositoryHint: "Maven",
			},
			FixedInVersion: "introduced=0&fixed=2.5.12",
			Repo: &claircore.Repository{
				Name: "maven",
				URI:  "https://repo1.maven.apache.org/maven2",
			},
		},
		{
			// Vuln: CVE-2022-22965/GHSA-36p3-wjmg-h94x (Spring4Shell)
			// Reason: Same as previous entry but with different vulnerable range.
			// Source: https://osv-vulnerabilities.storage.googleapis.com/Maven/GHSA-36p3-wjmg-h94x.json
			Updater:            u.Name(),
			Name:               "CVE-2022-22965",
			Description:        "Remote Code Execution in Spring Framework",
			Issued:             mustParseTime(time.RFC3339, "2022-03-31T18:30:50Z"),
			Links:              "https://nvd.nist.gov/vuln/detail/CVE-2022-22965",
			Severity:           "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
			NormalizedSeverity: claircore.Critical,
			Package: &claircore.Package{
				Name:           "spring-boot-starter-web",
				Kind:           claircore.BINARY,
				RepositoryHint: "Maven",
			},
			FixedInVersion: "introduced=2.6.0&fixed=2.6.6",
			Repo: &claircore.Repository{
				Name: "maven",
				URI:  "https://repo1.maven.apache.org/maven2",
			},
		},
		{
			// Vuln: CVE-2022-22965/GHSA-36p3-wjmg-h94x (Spring4Shell)
			// Reason: Same as previous entries but with different package name.
			// Source: https://osv-vulnerabilities.storage.googleapis.com/Maven/GHSA-36p3-wjmg-h94x.json
			Updater:            u.Name(),
			Name:               "CVE-2022-22965",
			Description:        "Remote Code Execution in Spring Framework",
			Issued:             mustParseTime(time.RFC3339, "2022-03-31T18:30:50Z"),
			Links:              "https://nvd.nist.gov/vuln/detail/CVE-2022-22965",
			Severity:           "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
			NormalizedSeverity: claircore.Critical,
			Package: &claircore.Package{
				Name:           "spring-webflux",
				Kind:           claircore.BINARY,
				RepositoryHint: "Maven",
			},
			FixedInVersion: "introduced=0&fixed=5.2.20.RELEASE",
			Repo: &claircore.Repository{
				Name: "maven",
				URI:  "https://repo1.maven.apache.org/maven2",
			},
		},
		{
			// Vuln: CVE-2022-22965/GHSA-36p3-wjmg-h94x (Spring4Shell)
			// Reason: Same as previous entry but with different vulnerable range.
			// Source: https://osv-vulnerabilities.storage.googleapis.com/Maven/GHSA-36p3-wjmg-h94x.json
			Updater:            u.Name(),
			Name:               "CVE-2022-22965",
			Description:        "Remote Code Execution in Spring Framework",
			Issued:             mustParseTime(time.RFC3339, "2022-03-31T18:30:50Z"),
			Links:              "https://nvd.nist.gov/vuln/detail/CVE-2022-22965",
			Severity:           "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
			NormalizedSeverity: claircore.Critical,
			Package: &claircore.Package{
				Name:           "spring-webflux",
				Kind:           claircore.BINARY,
				RepositoryHint: "Maven",
			},
			FixedInVersion: "introduced=5.3.0&fixed=5.3.18",
			Repo: &claircore.Repository{
				Name: "maven",
				URI:  "https://repo1.maven.apache.org/maven2",
			},
		},
		{
			// Vuln: CVE-2022-22965/GHSA-36p3-wjmg-h94x (Spring4Shell)
			// Reason: Same as previous entries but with different package name.
			// Source: https://osv-vulnerabilities.storage.googleapis.com/Maven/GHSA-36p3-wjmg-h94x.json
			Updater:            u.Name(),
			Name:               "CVE-2022-22965",
			Description:        "Remote Code Execution in Spring Framework",
			Issued:             mustParseTime(time.RFC3339, "2022-03-31T18:30:50Z"),
			Links:              "https://nvd.nist.gov/vuln/detail/CVE-2022-22965",
			Severity:           "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
			NormalizedSeverity: claircore.Critical,
			Package: &claircore.Package{
				Name:           "spring-boot-starter-webflux",
				Kind:           claircore.BINARY,
				RepositoryHint: "Maven",
			},
			FixedInVersion: "introduced=0&fixed=2.5.12",
			Repo: &claircore.Repository{
				Name: "maven",
				URI:  "https://repo1.maven.apache.org/maven2",
			},
		},
		{
			// Vuln: CVE-2022-22965/GHSA-36p3-wjmg-h94x (Spring4Shell)
			// Reason: Same as previous entry but with different vulnerable range.
			// Source: https://osv-vulnerabilities.storage.googleapis.com/Maven/GHSA-36p3-wjmg-h94x.json
			Updater:            u.Name(),
			Name:               "CVE-2022-22965",
			Description:        "Remote Code Execution in Spring Framework",
			Issued:             mustParseTime(time.RFC3339, "2022-03-31T18:30:50Z"),
			Links:              "https://nvd.nist.gov/vuln/detail/CVE-2022-22965",
			Severity:           "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
			NormalizedSeverity: claircore.Critical,
			Package: &claircore.Package{
				Name:           "spring-boot-starter-webflux",
				Kind:           claircore.BINARY,
				RepositoryHint: "Maven",
			},
			FixedInVersion: "introduced=2.6.0&fixed=2.6.6",
			Repo: &claircore.Repository{
				Name: "maven",
				URI:  "https://repo1.maven.apache.org/maven2",
			},
		},

		{
			// Vuln: CVE-2022-22978/GHSA-hh32-7344-cg2f
			// Reason: The vuln table has an entry for GHSA-hh32-7344-cg2f, but Scanner V4
			// may have trouble determining the groupID when pom.properties is missing.
			// Source: https://osv-vulnerabilities.storage.googleapis.com/Maven/GHSA-hh32-7344-cg2f.json
			Updater:            u.Name(),
			Name:               "CVE-2022-22978",
			Description:        "Authorization bypass in Spring Security",
			Issued:             mustParseTime(time.RFC3339, "2022-05-20T00:00:39Z"),
			Links:              "https://nvd.nist.gov/vuln/detail/CVE-2022-22978",
			Severity:           "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
			NormalizedSeverity: claircore.Critical,
			Package: &claircore.Package{
				Name:           "spring-security-core",
				Kind:           claircore.BINARY,
				RepositoryHint: "Maven",
			},
			FixedInVersion: "introduced=0&fixed=5.5.7",
			Repo: &claircore.Repository{
				Name: "maven",
				URI:  "https://repo1.maven.apache.org/maven2",
			},
		},
		{
			// Vuln: CVE-2022-22978/GHSA-hh32-7344-cg2f
			// Reason: Same as previous entry but with different vulnerable range.
			// Source: https://osv-vulnerabilities.storage.googleapis.com/Maven/GHSA-hh32-7344-cg2f.json
			Updater:            u.Name(),
			Name:               "CVE-2022-22978",
			Description:        "Authorization bypass in Spring Security",
			Issued:             mustParseTime(time.RFC3339, "2022-05-20T00:00:39Z"),
			Links:              "https://nvd.nist.gov/vuln/detail/CVE-2022-22978",
			Severity:           "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
			NormalizedSeverity: claircore.Critical,
			Package: &claircore.Package{
				Name:           "spring-security-core",
				Kind:           claircore.BINARY,
				RepositoryHint: "Maven",
			},
			FixedInVersion: "introduced=5.6.0&fixed=5.6.4",
			Repo: &claircore.Repository{
				Name: "maven",
				URI:  "https://repo1.maven.apache.org/maven2",
			},
		},
	}
}

func mustParseTime(layout, value string) time.Time {
	t, err := time.Parse(layout, value)
	utils.CrashOnError(err)
	return t
}
