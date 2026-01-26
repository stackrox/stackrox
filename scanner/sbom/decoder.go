package sbom

import (
	"github.com/quay/claircore/alpine"
	"github.com/quay/claircore/aws"
	"github.com/quay/claircore/debian"
	"github.com/quay/claircore/nodejs"
	"github.com/quay/claircore/oracle"
	"github.com/quay/claircore/photon"
	"github.com/quay/claircore/purl"
	"github.com/quay/claircore/python"
	"github.com/quay/claircore/rhel"
	"github.com/quay/claircore/rhel/rhcc"
	"github.com/quay/claircore/ruby"
	"github.com/quay/claircore/sbom/spdx"
	"github.com/quay/claircore/suse"
	"github.com/quay/claircore/ubuntu"
)

func NewPURLRegistry() *purl.Registry {
	reg := purl.NewRegistry()

	// Distro-based ecosystems with fixed namespaces.
	reg.RegisterPurlType(rhel.PURLType, rhel.PURLNamespace, rhel.ParseRPMPURL)
	reg.RegisterPurlType(suse.PURLType, suse.PURLNamespace, suse.ParsePURL)
	reg.RegisterPurlType(oracle.PURLType, oracle.PURLNamespace, oracle.ParsePURL)
	reg.RegisterPurlType(photon.PURLType, photon.PURLNamespace, photon.ParsePURL)
	reg.RegisterPurlType(aws.PURLType, aws.PURLNamespace, aws.ParsePURL)
	reg.RegisterPurlType(alpine.PURLType, alpine.PURLNamespace, alpine.ParsePURL)
	reg.RegisterPurlType(debian.PURLType, debian.PURLNamespace, debian.ParsePURL)
	reg.RegisterPurlType(ubuntu.PURLType, ubuntu.PURLNamespace, ubuntu.ParsePURL)

	// Language ecosystems without namespaces.
	reg.RegisterPurlType(python.PURLType, purl.NoneNamespace, python.ParsePURL)
	reg.RegisterPurlType(nodejs.PURLType, purl.NoneNamespace, nodejs.ParsePURL)
	reg.RegisterPurlType(ruby.PURLType, purl.NoneNamespace, ruby.ParsePURL)
	reg.RegisterPurlType(rhcc.PURLType, purl.NoneNamespace, rhcc.ParseOCIPURL)

	// NOTE: golang and maven PURLs use variable namespaces (domain, groupId)
	// that cannot be pre-registered with the current purl.Registry key scheme.
	// These PURLs are gracefully skipped by the decoder.

	return reg
}

func NewSPDXDecoder(registry purl.Converter) *spdx.Decoder {
	return spdx.NewDefaultDecoder(spdx.WithDecoderPURLConverter(registry))
}
