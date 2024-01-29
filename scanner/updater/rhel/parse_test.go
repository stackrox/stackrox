package rhel

import (
	"context"
	"net/url"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/zlog"
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
			configFunc:        func(_ any) error { return nil },
			expectedVulnCount: 192,
		},
		{
			name:              "ignore unpatched path",
			fileName:          "testdata/rhel-8-rpm-unpatched.xml",
			configFunc:        func(c any) error { return nil },
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
		s, err := url.ParseQuery(vuln.Severity)
		if err != nil {
			t.Fatalf("invalid severity: %s", vuln.Severity)
		}
		if got, want := s.Get("cvss3_score"), "7.5"; got != want {
			t.Fatalf("unexpected CVSS3 score: got: %s, want: %s", got, want)
		}
		if got, want := s.Get("cvss3_vector"), "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:N/A:N"; got != want {
			t.Fatalf("unexpected CVSS3 vector: got: %s, want: %s", got, want)
		}
		if got, want := s.Get("cvss2_score"), ""; got != want {
			t.Fatalf("unexpected CVSS2 score: got: %s, want: %s", got, want)
		}
		if got, want := s.Get("severity"), "Important"; got != want {
			t.Fatalf("unexpected severity: got: %s, want: %s", got, want)
		}
	}

	const (
		base      = "cpe:/a:redhat:enterprise_linux:8"
		appstream = "cpe:/a:redhat:enterprise_linux:8::appstream"
	)
	if count[base] != 15 || count[appstream] != 15 {
		t.Fatalf("got: %v vulnerabilities with, want 15 of each", count)
	}
}

func TestAllKnownOpenShift4CPEs(t *testing.T) {
	table := []struct {
		cpe      string
		expected []string
	}{
		{
			cpe: "cpe:/a:redhat:openshift:4.14",
			expected: []string{
				"cpe:/a:redhat:openshift:4.0",
				"cpe:/a:redhat:openshift:4.1",
				"cpe:/a:redhat:openshift:4.2",
				"cpe:/a:redhat:openshift:4.3",
				"cpe:/a:redhat:openshift:4.4",
				"cpe:/a:redhat:openshift:4.5",
				"cpe:/a:redhat:openshift:4.6",
				"cpe:/a:redhat:openshift:4.7",
				"cpe:/a:redhat:openshift:4.8",
				"cpe:/a:redhat:openshift:4.9",
				"cpe:/a:redhat:openshift:4.10",
				"cpe:/a:redhat:openshift:4.11",
				"cpe:/a:redhat:openshift:4.12",
				"cpe:/a:redhat:openshift:4.13",
			},
		},
		{
			cpe: "cpe:/a:redhat:openshift:4.15::el8",
			expected: []string{
				"cpe:/a:redhat:openshift:4.0::el8",
				"cpe:/a:redhat:openshift:4.1::el8",
				"cpe:/a:redhat:openshift:4.2::el8",
				"cpe:/a:redhat:openshift:4.3::el8",
				"cpe:/a:redhat:openshift:4.4::el8",
				"cpe:/a:redhat:openshift:4.5::el8",
				"cpe:/a:redhat:openshift:4.6::el8",
				"cpe:/a:redhat:openshift:4.7::el8",
				"cpe:/a:redhat:openshift:4.8::el8",
				"cpe:/a:redhat:openshift:4.9::el8",
				"cpe:/a:redhat:openshift:4.10::el8",
				"cpe:/a:redhat:openshift:4.11::el8",
				"cpe:/a:redhat:openshift:4.12::el8",
				"cpe:/a:redhat:openshift:4.13::el8",
				"cpe:/a:redhat:openshift:4.14::el8",
			},
		},
		{
			cpe: "cpe:/a:redhat:openshift:4.15::el9",
			expected: []string{
				"cpe:/a:redhat:openshift:4.0::el9",
				"cpe:/a:redhat:openshift:4.1::el9",
				"cpe:/a:redhat:openshift:4.2::el9",
				"cpe:/a:redhat:openshift:4.3::el9",
				"cpe:/a:redhat:openshift:4.4::el9",
				"cpe:/a:redhat:openshift:4.5::el9",
				"cpe:/a:redhat:openshift:4.6::el9",
				"cpe:/a:redhat:openshift:4.7::el9",
				"cpe:/a:redhat:openshift:4.8::el9",
				"cpe:/a:redhat:openshift:4.9::el9",
				"cpe:/a:redhat:openshift:4.10::el9",
				"cpe:/a:redhat:openshift:4.11::el9",
				"cpe:/a:redhat:openshift:4.12::el9",
				"cpe:/a:redhat:openshift:4.13::el9",
				"cpe:/a:redhat:openshift:4.14::el9",
			},
		},
	}

	for _, test := range table {
		t.Run(test.cpe, func(t *testing.T) {
			cpes, err := allKnownOpenShift4CPEs(test.cpe)
			if err != nil {
				t.Fatal(err)
			}

			if !cmp.Equal(cpes, test.expected) {
				t.Fatal(cmp.Diff(cpes, test.expected))
			}
		})
	}
}
