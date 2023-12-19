package rhel

import (
	"context"
	"encoding/xml"
	"os"
	"testing"
	"time"

	"github.com/quay/goval-parser/oval"
	"github.com/quay/zlog"

	"github.com/quay/claircore/libvuln/driver"
)

func TestCVEDefFromUnpatched(t *testing.T) {
	ctx := context.Background()
	var table = []struct {
		name              string
		fileName          string
		configFunc        driver.ConfigUnmarshaler
		expectedVulnCount int
		ignoreUnpatched   bool
	}{
		{
			name:              "default path",
			fileName:          "testdata/rhel-8-rpm-unpatched.xml",
			configFunc:        func(_ interface{}) error { return nil },
			expectedVulnCount: 192,
		},
		{
			name:              "ignore unpatched path",
			fileName:          "testdata/rhel-8-rpm-unpatched.xml",
			configFunc:        func(c interface{}) error { return nil },
			ignoreUnpatched:   true,
			expectedVulnCount: 0,
		},
	}

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			ctx := zlog.Test(ctx, t)

			f, err := os.Open(test.fileName)
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			u, err := NewUpdater("rhel-8-unpatched-updater", 8, "file:///dev/null", test.ignoreUnpatched)
			if err != nil {
				t.Fatal(err)
			}

			u.Configure(ctx, test.configFunc, nil)

			vulns, err := u.Parse(ctx, f)
			if err != nil {
				t.Fatal(err)
			}
			if len(vulns) != test.expectedVulnCount {
				t.Fatalf("was expecting %d vulns, but got %d", test.expectedVulnCount, len(vulns))
			}
		})
	}
}

func TestParse(t *testing.T) {
	t.Parallel()
	ctx := zlog.Test(context.Background(), t)

	u, err := NewUpdater(`rhel-3-updater`, 3, "file:///dev/null", false)
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Open("testdata/com.redhat.rhsa-20201980.xml")
	if err != nil {
		t.Fatal(err)
	}

	vs, err := u.Parse(ctx, f)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("found %d vulnerabilities", len(vs))
	// 15 packages, 2 cpes = 30 vulnerabilities
	if got, want := len(vs), 30; got != want {
		t.Fatalf("got: %d vulnerabilities, want: %d vulnerabilities", got, want)
	}
	count := make(map[string]int)
	for _, vuln := range vs {
		count[vuln.Repo.Name]++
	}

	const (
		base      = "cpe:/a:redhat:enterprise_linux:8"
		appstream = "cpe:/a:redhat:enterprise_linux:8::appstream"
	)
	if count[base] != 15 || count[appstream] != 15 {
		t.Fatalf("got: %v vulnerabilities with, want 15 of each", count)
	}
}

// Here's a giant restructured struct for reference and tests.
var ovalDef = oval.Definition{
	XMLName: xml.Name{Space: "http://oval.mitre.org/XMLSchema/oval-definitions-5", Local: "definition"},
	ID:      "oval:com.redhat.rhsa:def:20100401",
	Class:   "patch",
	Title:   "RHSA-2010:0401: tetex security update (Moderate)",
	Affecteds: []oval.Affected{
		{
			XMLName:   xml.Name{Space: "http://oval.mitre.org/XMLSchema/oval-definitions-5", Local: "affected"},
			Family:    "unix",
			Platforms: []string{"Red Hat Enterprise Linux 3"},
		},
	},
	References: []oval.Reference{
		{
			XMLName: xml.Name{Space: "http://oval.mitre.org/XMLSchema/oval-definitions-5", Local: "reference"},
			Source:  "RHSA",
			RefID:   "RHSA-2010:0401",
			RefURL:  "https://access.redhat.com/errata/RHSA-2010:0401",
		},
		{
			XMLName: xml.Name{Space: "http://oval.mitre.org/XMLSchema/oval-definitions-5", Local: "reference"},
			Source:  "CVE",
			RefID:   "CVE-2007-5935",
			RefURL:  "https://access.redhat.com/security/cve/CVE-2007-5935",
		},
		{
			XMLName: xml.Name{Space: "http://oval.mitre.org/XMLSchema/oval-definitions-5", Local: "reference"},
			Source:  "CVE",
			RefID:   "CVE-2009-0791",
			RefURL:  "https://access.redhat.com/security/cve/CVE-2009-0791",
		},
		{
			XMLName: xml.Name{Space: "http://oval.mitre.org/XMLSchema/oval-definitions-5", Local: "reference"},
			Source:  "CVE",
			RefID:   "CVE-2009-3609",
			RefURL:  "https://access.redhat.com/security/cve/CVE-2009-3609",
		},
		{
			XMLName: xml.Name{Space: "http://oval.mitre.org/XMLSchema/oval-definitions-5", Local: "reference"},
			Source:  "CVE",
			RefID:   "CVE-2010-0739",
			RefURL:  "https://access.redhat.com/security/cve/CVE-2010-0739",
		},
		{
			XMLName: xml.Name{Space: "http://oval.mitre.org/XMLSchema/oval-definitions-5", Local: "reference"},
			Source:  "CVE",
			RefID:   "CVE-2010-0827",
			RefURL:  "https://access.redhat.com/security/cve/CVE-2010-0827",
		},
		{
			XMLName: xml.Name{Space: "http://oval.mitre.org/XMLSchema/oval-definitions-5", Local: "reference"},
			Source:  "CVE",
			RefID:   "CVE-2010-1440",
			RefURL:  "https://access.redhat.com/security/cve/CVE-2010-1440",
		},
	},
	Description: "teTeX is an implementation of TeX. TeX takes a text file and a set of\nformatting commands as input, and creates a typesetter-independent DeVice\nIndependent (DVI) file as output.\n\nA buffer overflow flaw was found in the way teTeX processed virtual font\nfiles when converting DVI files into PostScript. An attacker could create a\nmalicious DVI file that would cause the dvips executable to crash or,\npotentially, execute arbitrary code. (CVE-2010-0827)\n\nMultiple integer overflow flaws were found in the way teTeX processed\nspecial commands when converting DVI files into PostScript. An attacker\ncould create a malicious DVI file that would cause the dvips executable to\ncrash or, potentially, execute arbitrary code. (CVE-2010-0739,\nCVE-2010-1440)\n\nA stack-based buffer overflow flaw was found in the way teTeX processed DVI\nfiles containing HyperTeX references with long titles, when converting them\ninto PostScript. An attacker could create a malicious DVI file that would\ncause the dvips executable to crash. (CVE-2007-5935)\n\nteTeX embeds a copy of Xpdf, an open source Portable Document Format (PDF)\nfile viewer, to allow adding images in PDF format to the generated PDF\ndocuments. The following issues affect Xpdf code:\n\nMultiple integer overflow flaws were found in Xpdf. If a local user\ngenerated a PDF file from a TeX document, referencing a specially-crafted\nPDF file, it would cause Xpdf to crash or, potentially, execute arbitrary\ncode with the privileges of the user running pdflatex. (CVE-2009-0791,\nCVE-2009-3609)\n\nAll users of tetex are advised to upgrade to these updated packages, which\ncontain backported patches to correct these issues.",
	Advisory: oval.Advisory{
		XMLName:  xml.Name{Space: "http://oval.mitre.org/XMLSchema/oval-definitions-5", Local: "advisory"},
		Severity: "Moderate",
		Cves: []oval.Cve{
			{
				XMLName: xml.Name{Space: "http://oval.mitre.org/XMLSchema/oval-definitions-5", Local: "cve"},
				CveID:   "CVE-2007-5935",
				Cvss2:   "",
				Cvss3:   "",
				Cwe:     "",
				Impact:  "low",
				Href:    "https://access.redhat.com/security/cve/CVE-2007-5935",
				Public:  "20071017",
			},
			{
				XMLName: xml.Name{Space: "http://oval.mitre.org/XMLSchema/oval-definitions-5", Local: "cve"},
				CveID:   "CVE-2009-0791",
				Cvss2:   "5.8/AV:A/AC:L/Au:N/C:P/I:P/A:P",
				Cvss3:   "",
				Cwe:     "CWE-190",
				Impact:  "",
				Href:    "https://access.redhat.com/security/cve/CVE-2009-0791",
				Public:  "20090519",
			},
			{
				XMLName: xml.Name{Space: "http://oval.mitre.org/XMLSchema/oval-definitions-5", Local: "cve"},
				CveID:   "CVE-2009-3609",
				Cvss2:   "2.1/AV:L/AC:L/Au:N/C:N/I:N/A:P",
				Cvss3:   "",
				Cwe:     "CWE-190",
				Impact:  "low",
				Href:    "https://access.redhat.com/security/cve/CVE-2009-3609",
				Public:  "20091014",
			},
			{
				XMLName: xml.Name{Space: "http://oval.mitre.org/XMLSchema/oval-definitions-5", Local: "cve"},
				CveID:   "CVE-2010-0739",
				Cvss2:   "6.8/AV:N/AC:M/Au:N/C:P/I:P/A:P",
				Cvss3:   "",
				Cwe:     "CWE-190",
				Impact:  "",
				Href:    "https://access.redhat.com/security/cve/CVE-2010-0739",
				Public:  "20100412",
			},
			{
				XMLName: xml.Name{Space: "http://oval.mitre.org/XMLSchema/oval-definitions-5", Local: "cve"},
				CveID:   "CVE-2010-0827",
				Cvss2:   "6.8/AV:N/AC:M/Au:N/C:P/I:P/A:P",
				Cvss3:   "",
				Cwe:     "",
				Impact:  "",
				Href:    "https://access.redhat.com/security/cve/CVE-2010-0827",
				Public:  "20100325",
			},
			{
				XMLName: xml.Name{Space: "http://oval.mitre.org/XMLSchema/oval-definitions-5", Local: "cve"},
				CveID:   "CVE-2010-1440",
				Cvss2:   "6.8/AV:N/AC:M/Au:N/C:P/I:P/A:P",
				Cvss3:   "",
				Cwe:     "CWE-190",
				Impact:  "",
				Href:    "https://access.redhat.com/security/cve/CVE-2010-1440",
				Public:  "20100503",
			},
		},
		Bugzillas: []oval.Bugzilla{
			{
				XMLName: xml.Name{Space: "http://oval.mitre.org/XMLSchema/oval-definitions-5", Local: "bugzilla"},
				ID:      "368591",
				URL:     "https://bugzilla.redhat.com/368591",
				Title:   "CVE-2007-5935 dvips -z buffer overflow with long href",
			},
			{
				XMLName: xml.Name{Space: "http://oval.mitre.org/XMLSchema/oval-definitions-5", Local: "bugzilla"},
				ID:      "491840",
				URL:     "https://bugzilla.redhat.com/491840",
				Title:   "CVE-2009-0791 xpdf: multiple integer overflows",
			},
			{
				XMLName: xml.Name{Space: "http://oval.mitre.org/XMLSchema/oval-definitions-5", Local: "bugzilla"},
				ID:      "526893",
				URL:     "https://bugzilla.redhat.com/526893",
				Title:   "CVE-2009-3609 xpdf/poppler: ImageStream::ImageStream integer overflow",
			},
			{
				XMLName: xml.Name{Space: "http://oval.mitre.org/XMLSchema/oval-definitions-5", Local: "bugzilla"},
				ID:      "572914",
				URL:     "https://bugzilla.redhat.com/572914",
				Title:   "CVE-2010-0827 tetex, texlive: Buffer overflow flaw by processing virtual font files",
			},
			{
				XMLName: xml.Name{Space: "http://oval.mitre.org/XMLSchema/oval-definitions-5", Local: "bugzilla"},
				ID:      "572941",
				URL:     "https://bugzilla.redhat.com/572941",
				Title:   "CVE-2010-0739 tetex, texlive: Integer overflow by processing special commands",
			},
			{
				XMLName: xml.Name{Space: "http://oval.mitre.org/XMLSchema/oval-definitions-5", Local: "bugzilla"},
				ID:      "586819",
				URL:     "https://bugzilla.redhat.com/586819",
				Title:   "CVE-2010-1440 tetex, texlive: Integer overflow by processing special commands",
			},
		},
		AffectedCPEList: []string{"cpe:/o:redhat:enterprise_linux:3"},
		Refs:            []oval.Ref(nil),
		Bugs:            []oval.Bug(nil),
		Issued: oval.Date{
			Date: time.Date(2010, 5, 6, 0, 0, 0, 0, time.UTC),
		},
		Updated: oval.Date{
			Date: time.Date(2010, 5, 6, 0, 0, 0, 0, time.UTC),
		},
	},
	Debian: oval.Debian{XMLName: xml.Name{Space: "", Local: ""}, MoreInfo: "", Date: oval.Date{Date: time.Time{}}},
	Criteria: oval.Criteria{
		XMLName:  xml.Name{Space: "http://oval.mitre.org/XMLSchema/oval-definitions-5", Local: "criteria"},
		Operator: "AND",
		Criterias: []oval.Criteria{
			{
				XMLName:  xml.Name{Space: "http://oval.mitre.org/XMLSchema/oval-definitions-5", Local: "criteria"},
				Operator: "OR",
				Criterias: []oval.Criteria{
					{
						XMLName:   xml.Name{Space: "http://oval.mitre.org/XMLSchema/oval-definitions-5", Local: "criteria"},
						Operator:  "AND",
						Criterias: []oval.Criteria(nil),
						Criterions: []oval.Criterion{
							{
								XMLName: xml.Name{Space: "http://oval.mitre.org/XMLSchema/oval-definitions-5", Local: "criterion"},
								Negate:  false,
								TestRef: "oval:com.redhat.rhsa:tst:20100401001",
								Comment: "tetex-xdvi is earlier than 0:1.0.7-67.19",
							},
							{
								XMLName: xml.Name{Space: "http://oval.mitre.org/XMLSchema/oval-definitions-5", Local: "criterion"},
								Negate:  false,
								TestRef: "oval:com.redhat.rhsa:tst:20060160004",
								Comment: "tetex-xdvi is signed with Red Hat master key",
							},
						},
					},
					{
						XMLName:   xml.Name{Space: "http://oval.mitre.org/XMLSchema/oval-definitions-5", Local: "criteria"},
						Operator:  "AND",
						Criterias: []oval.Criteria(nil),
						Criterions: []oval.Criterion{
							{
								XMLName: xml.Name{Space: "http://oval.mitre.org/XMLSchema/oval-definitions-5", Local: "criterion"},
								Negate:  false,
								TestRef: "oval:com.redhat.rhsa:tst:20100401003",
								Comment: "tetex-fonts is earlier than 0:1.0.7-67.19",
							},
							{
								XMLName: xml.Name{Space: "http://oval.mitre.org/XMLSchema/oval-definitions-5", Local: "criterion"},
								Negate:  false,
								TestRef: "oval:com.redhat.rhsa:tst:20060160012",
								Comment: "tetex-fonts is signed with Red Hat master key",
							},
						},
					},
					{
						XMLName:   xml.Name{Space: "http://oval.mitre.org/XMLSchema/oval-definitions-5", Local: "criteria"},
						Operator:  "AND",
						Criterias: []oval.Criteria(nil),
						Criterions: []oval.Criterion{
							{
								XMLName: xml.Name{Space: "http://oval.mitre.org/XMLSchema/oval-definitions-5", Local: "criterion"},
								Negate:  false,
								TestRef: "oval:com.redhat.rhsa:tst:20100401005",
								Comment: "tetex-dvips is earlier than 0:1.0.7-67.19",
							},
							{
								XMLName: xml.Name{Space: "http://oval.mitre.org/XMLSchema/oval-definitions-5", Local: "criterion"},
								Negate:  false,
								TestRef: "oval:com.redhat.rhsa:tst:20060160008",
								Comment: "tetex-dvips is signed with Red Hat master key",
							},
						},
					},
					{
						XMLName:   xml.Name{Space: "http://oval.mitre.org/XMLSchema/oval-definitions-5", Local: "criteria"},
						Operator:  "AND",
						Criterias: []oval.Criteria(nil),
						Criterions: []oval.Criterion{
							{
								XMLName: xml.Name{Space: "http://oval.mitre.org/XMLSchema/oval-definitions-5", Local: "criterion"},
								Negate:  false,
								TestRef: "oval:com.redhat.rhsa:tst:20100401007",
								Comment: "tetex is earlier than 0:1.0.7-67.19",
							},
							{
								XMLName: xml.Name{Space: "http://oval.mitre.org/XMLSchema/oval-definitions-5", Local: "criterion"},
								Negate:  false,
								TestRef: "oval:com.redhat.rhsa:tst:20060160002",
								Comment: "tetex is signed with Red Hat master key",
							},
						},
					},
					{
						XMLName:   xml.Name{Space: "http://oval.mitre.org/XMLSchema/oval-definitions-5", Local: "criteria"},
						Operator:  "AND",
						Criterias: []oval.Criteria(nil),
						Criterions: []oval.Criterion{
							{
								XMLName: xml.Name{Space: "http://oval.mitre.org/XMLSchema/oval-definitions-5", Local: "criterion"},
								Negate:  false,
								TestRef: "oval:com.redhat.rhsa:tst:20100401009",
								Comment: "tetex-afm is earlier than 0:1.0.7-67.19",
							},
							{
								XMLName: xml.Name{Space: "http://oval.mitre.org/XMLSchema/oval-definitions-5", Local: "criterion"},
								Negate:  false,
								TestRef: "oval:com.redhat.rhsa:tst:20060160010",
								Comment: "tetex-afm is signed with Red Hat master key",
							},
						},
					},
					{
						XMLName:   xml.Name{Space: "http://oval.mitre.org/XMLSchema/oval-definitions-5", Local: "criteria"},
						Operator:  "AND",
						Criterias: []oval.Criteria(nil),
						Criterions: []oval.Criterion{
							{
								XMLName: xml.Name{Space: "http://oval.mitre.org/XMLSchema/oval-definitions-5", Local: "criterion"},
								Negate:  false,
								TestRef: "oval:com.redhat.rhsa:tst:20100401011",
								Comment: "tetex-latex is earlier than 0:1.0.7-67.19",
							},
							{
								XMLName: xml.Name{Space: "http://oval.mitre.org/XMLSchema/oval-definitions-5", Local: "criterion"},
								Negate:  false,
								TestRef: "oval:com.redhat.rhsa:tst:20060160006",
								Comment: "tetex-latex is signed with Red Hat master key",
							},
						},
					},
				},
				Criterions: []oval.Criterion(nil),
			},
		},
		Criterions: []oval.Criterion{
			{
				XMLName: xml.Name{Space: "http://oval.mitre.org/XMLSchema/oval-definitions-5", Local: "criterion"},
				Negate:  false,
				TestRef: "oval:com.redhat.rhba:tst:20070026003",
				Comment: "Red Hat Enterprise Linux 3 is installed",
			},
		},
	},
}
