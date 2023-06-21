package test

import (
	"github.com/quay/claircore"
)

var manuallyTestVulns = []*claircore.Vulnerability{
	{
		Updater:     "manual",
		Name:        "GHSA-cj7v-27pg-wf7q",
		Description: "URI use within Jetty's HttpURI class can parse invalid URIs such as http://localhost;/path as having an authority with a host of localhost;A URIs of the type http://localhost;/path should be interpreted to be either invalid or as localhost; to be the userinfo and no host. However, HttpURI.host returns localhost; which is definitely wrong.",
		//Issued:             time.Parse("2006-01-02 15:04", "2022-07-07T20:55:34Z"),
		Links:              "https://github.com/github/advisory-database/blob/main/advisories/github-reviewed/2022/07/GHSA-cj7v-27pg-wf7q/GHSA-cj7v-27pg-wf7q.json",
		Severity:           "CVSS:3.1/AV:N/AC:L/PR:H/UI:N/S:U/C:N/I:L/A:N",
		NormalizedSeverity: claircore.Low,
		Package: &claircore.Package{
			Name: "org.eclipse.jetty:jetty-http",
			Kind: claircore.BINARY,
		},
		FixedInVersion: "fixed=9.4.47&introduced=0",
		Repo: &claircore.Repository{
			Name: "maven",
		},
	}}
