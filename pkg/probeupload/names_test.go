package probeupload

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidModuleVersion_Valid(t *testing.T) {
	t.Parallel()

	values := []string{
		"1123dde0458e72a49880b06922e135dbcd36fb784fed530ab84ddfa8924e5c05",
		"95eb0815c4e7b59e0e5d0e53adb1a4faa5d5d902ad4caef2a27ed57a7f6260c3",
		"612dd2ee06b660e728292de9393e18c81a88f347ec52a39207c5166b5302b656",
		"a409284ad5be9a95bfd65b9eac6f179094d5b36af9a6ba3548fa98ee4d23a7a5",
		"7c30b6f295bae9ccf8695982687d871847dfecd12a1cfbc3edcfa93ceec6b5dc",
		"f7bd36bc2f3299306385c1270805fa3705af934acd37c6d2395dbba567dd3c58",
		"0.0.1",
		"1.2.3",
		"10.5.155",
		"1.0.0-rc1",
		"2.3.4-rc123",
	}

	for _, v := range values {
		assert.Truef(t, IsValidModuleVersion(v), "%q should be a valid module version", v)
	}
}

func TestIsValidModuleVersion_Invalid(t *testing.T) {
	t.Parallel()

	values := []string{
		"",
		"0.0",
		".2.3",
		"2.3.",
		"10.5.155.123",
		"1.0.0-rc",
		"2.0.0-invalid",
		"1.0.0-rc1-rc4",
		"2.0.0-rc1invalid",
		"latest",
		"1123dde0458e72a49880b06922e135dbcd36fb784fed530ab84ddfa8924e5c050",
		"95eb0815c4e7b59e0e5d0e53adb1a4",
		"612dd2ee06b660e728292de9393e18c81a88f347ec52a39207c5166b5302b6g6",
	}

	for _, v := range values {
		assert.Falsef(t, IsValidModuleVersion(v), "%q should not be a valid module version", v)
	}
}

func TestIsValidProbeName_Valid(t *testing.T) {
	t.Parallel()

	values := []string{
		"collector-4.9.24-coreos.ko.gz",
		"collector-ebpf-4.19.76-12371.89.0-cos.o.gz",
		"collector-4.15.0-1012-azure.ko.gz",
		"collector-ebpf-4.15.0-1061-azure.o.gz",
		"collector-4.4.79-k8s.ko.gz",
		"collector-4.4.81-1.el7.elrepo.x86_64.ko.gz",
	}

	for _, v := range values {
		assert.Truef(t, IsValidProbeName(v), "%q should be a valid probe name", v)
	}
}

func TestIsValidProbeName_Invalid(t *testing.T) {
	t.Parallel()

	values := []string{
		"",
		"collector-4.9.24-coreos.o.gz",
		"collector-ebpf-4.19.76-12371.89.0-cos.o",
		"collector.ko.gz",
		"collector-ebpf-4.15.0-1061-azure.ko.gz",
		"collector-.ko.gz",
		"/collector-4.4.81-1.el7.elrepo.x86_64.ko.gz",
		"somfie",
	}

	for _, v := range values {
		assert.Falsef(t, IsValidProbeName(v), "%q should be an invalid probe name", v)
	}
}

func TestIsValidFilePath_Valid(t *testing.T) {
	t.Parallel()

	values := []string{
		"1123dde0458e72a49880b06922e135dbcd36fb784fed530ab84ddfa8924e5c05/collector-4.9.24-coreos.ko.gz",
		"95eb0815c4e7b59e0e5d0e53adb1a4faa5d5d902ad4caef2a27ed57a7f6260c3/collector-ebpf-4.19.76-12371.89.0-cos.o.gz",
		"612dd2ee06b660e728292de9393e18c81a88f347ec52a39207c5166b5302b656/collector-4.15.0-1012-azure.ko.gz",
		"a409284ad5be9a95bfd65b9eac6f179094d5b36af9a6ba3548fa98ee4d23a7a5/collector-ebpf-4.15.0-1061-azure.o.gz",
		"7c30b6f295bae9ccf8695982687d871847dfecd12a1cfbc3edcfa93ceec6b5dc/collector-4.4.79-k8s.ko.gz",
		"f7bd36bc2f3299306385c1270805fa3705af934acd37c6d2395dbba567dd3c58/collector-4.4.81-1.el7.elrepo.x86_64.ko.gz",
	}

	for _, v := range values {
		assert.Truef(t, IsValidFilePath(v), "%q should be a valid probe file path", v)
	}
}

func TestIsValidFilePath_Invalid(t *testing.T) {
	t.Parallel()

	values := []string{
		"",
		"/1123dde0458e72a49880b06922e135dbcd36fb784fed530ab84ddfa8924e5c05/collector-4.9.24-coreos.ko.gz",
		"collector-ebpf-4.19.76-12371.89.0-cos.o.gz",
		"612dd2ee06b660e728292de9393e18c81a88f347ec52a39207c5166b5302b656/collector-4.15.0-1012-azure.o.gz",
		"a409284ad5be9a95bfd65b9eac6f179094d5b36af9a6ba3548fa98ee4d23a7a5",
		"7c30b6f295bae9ccf8695982687d871847dfecd12a1cfbc3edcfa93ceec6b5dc/",
		"somefile",
		"somepath/somefile",
	}

	for _, v := range values {
		assert.Falsef(t, IsValidFilePath(v), "%q should be an invalid probe file path", v)
	}
}
