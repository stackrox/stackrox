package cache

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestCacheWithUnifiedDeferral(t *testing.T) {
	t.Setenv(features.UnifiedCVEDeferral.EnvVar(), "true")
	if !features.UnifiedCVEDeferral.Enabled() {
		t.Skipf("%s=false. Skipping test", features.UnifiedCVEDeferral.EnvVar())
	}

	testCache(t)
}

func TestCacheWithoutUnifiedDeferral(t *testing.T) {
	t.Setenv(features.UnifiedCVEDeferral.EnvVar(), "false")
	if features.UnifiedCVEDeferral.Enabled() {
		t.Skipf("%s=true. Skipping test", features.UnifiedCVEDeferral.EnvVar())
	}

	testCache(t)
}

func testCache(t *testing.T) {
	img := fixtures.GetImage()
	registry := img.GetName().GetRegistry()
	remote := img.GetName().GetRemote()
	tag := img.GetName().GetTag()
	vuln1 := img.Scan.Components[0].Vulns[0].GetCve()
	vuln2 := img.Scan.Components[0].Vulns[1].GetCve()
	vuln3 := img.Scan.Components[0].Vulns[2].GetCve()

	req1 := fixtures.GetGlobalFPRequest(vuln1)
	req2 := fixtures.GetGlobalDeferralRequest(vuln2)
	if features.UnifiedCVEDeferral.Enabled() {
		req1 = fixtures.GetGlobalFPRequestV2(vuln1)
		req2 = fixtures.GetGlobalDeferralRequestV2(vuln2)
	}
	req1.Id = "req1"
	req2.Id = "req2"
	req3 := fixtures.GetImageScopeDeferralRequest(registry, remote, tag, vuln1)
	req3.Id = "req3"
	req4 := fixtures.GetImageScopeDeferralRequest(registry, remote, tag, vuln3)
	req4.Id = "req4"
	req5 := fixtures.GetImageScopeDeferralRequest("reg-2", "fake", tag, vuln3)
	req5.Id = "req5"
	req6 := fixtures.GetImageScopeFPRequest(registry, remote, ".*", vuln3)
	req5.Id = "req6"
	req7 := fixtures.GetImageScopeDeferralRequest(registry, remote, "", vuln3)
	req7.Id = "req7"

	cache := New()
	for _, req := range []*storage.VulnerabilityRequest{req1, req2, req3, req4, req5, req6, req7} {
		assert.True(t, cache.Add(req))
	}

	// Test that the vulnerability state is scoped to the smallest scope.
	response := cache.GetVulnsWithState(registry, remote, tag)
	assert.Len(t, response, 3)
	assert.Equal(t, storage.VulnerabilityState_DEFERRED, response[vuln1])
	assert.Equal(t, storage.VulnerabilityState_DEFERRED, response[vuln2])
	assert.Equal(t, storage.VulnerabilityState_DEFERRED, response[vuln3])

	// Test if the correct vul req is returned.
	vulnReqID := cache.GetEffectiveVulnReqIDForImage(img.GetName().GetRegistry(), img.GetName().GetRemote(), img.GetName().GetTag(), vuln1)
	assert.Equal(t, req3.GetId(), vulnReqID)
	vulnReqID = cache.GetEffectiveVulnReqIDForImage(img.GetName().GetRegistry(), img.GetName().GetRemote(), "fake", vuln1)
	assert.Equal(t, req1.GetId(), vulnReqID)

	// Remove the image scoped request for vuln1
	cache.Remove(req3.GetId())

	// Remove one of the request for vuln3, and verify that for removed scope the cve state is taken from req6 which
	// applies to all tags.
	cache.Remove(req4.GetId())
	response = cache.GetEffectiveVulnStateForImage(
		[]string{vuln3},
		req4.GetScope().GetImageScope().GetRegistry(),
		req4.GetScope().GetImageScope().GetRemote(),
		req4.GetScope().GetImageScope().GetTag())
	assert.Equal(t, storage.VulnerabilityState_FALSE_POSITIVE, response[vuln3])

	// Verify that for req5 scope cve is still deferred since the request still exists.
	response = cache.GetEffectiveVulnStateForImage(
		[]string{vuln3},
		req5.GetScope().GetImageScope().GetRegistry(),
		req5.GetScope().GetImageScope().GetRemote(),
		req5.GetScope().GetImageScope().GetTag())
	assert.Equal(t, storage.VulnerabilityState_DEFERRED, response[vuln3])

	// Verify that for req7 scope the cve is deferred since request still exists.
	response = cache.GetEffectiveVulnStateForImage(
		[]string{vuln3},
		req7.GetScope().GetImageScope().GetRegistry(),
		req7.GetScope().GetImageScope().GetRemote(),
		req7.GetScope().GetImageScope().GetTag())
	assert.Equal(t, storage.VulnerabilityState_DEFERRED, response[vuln3])

	// Verify that for req4 scope the cve is observed since the all tags request is removed.
	cache.Remove(req6.GetId())
	response = cache.GetEffectiveVulnStateForImage(
		[]string{vuln3},
		req4.GetScope().GetImageScope().GetRegistry(),
		req4.GetScope().GetImageScope().GetRemote(),
		req4.GetScope().GetImageScope().GetTag())
	assert.Equal(t, storage.VulnerabilityState_OBSERVED, response[vuln3])

	// Verify that for req7 scope the cve is deferred since request still exists.
	response = cache.GetEffectiveVulnStateForImage(
		[]string{vuln3},
		req7.GetScope().GetImageScope().GetRegistry(),
		req7.GetScope().GetImageScope().GetRemote(),
		req7.GetScope().GetImageScope().GetTag())
	assert.Equal(t, storage.VulnerabilityState_DEFERRED, response[vuln3])

	// Verify that for req5 scope, cves are still in deferred since the request still exists.
	response = cache.GetEffectiveVulnStateForImage(
		[]string{vuln3},
		req5.GetScope().GetImageScope().GetRegistry(),
		req5.GetScope().GetImageScope().GetRemote(),
		req5.GetScope().GetImageScope().GetTag())
	assert.Equal(t, storage.VulnerabilityState_DEFERRED, response[vuln3])

	// Verify that for req5 scope, cves are in observed since no request exists.
	cache.Remove(req5.GetId())
	response = cache.GetEffectiveVulnStateForImage(
		[]string{vuln3},
		req5.GetScope().GetImageScope().GetRegistry(),
		req5.GetScope().GetImageScope().GetRemote(),
		req5.GetScope().GetImageScope().GetTag())
	assert.Equal(t, storage.VulnerabilityState_OBSERVED, response[vuln3])

	// Verify that for req7 scope the cve is observed since no request exists.
	cache.Remove(req7.GetId())
	response = cache.GetEffectiveVulnStateForImage(
		[]string{vuln3},
		req7.GetScope().GetImageScope().GetRegistry(),
		req7.GetScope().GetImageScope().GetRemote(),
		req7.GetScope().GetImageScope().GetTag())
	assert.Equal(t, storage.VulnerabilityState_OBSERVED, response[vuln3])

}
