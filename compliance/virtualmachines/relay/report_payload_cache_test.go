package relay

import (
	"fmt"
	"testing"
	"time"

	relaytest "github.com/stackrox/rox/compliance/virtualmachines/relay/testutils"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stretchr/testify/require"
)

const (
	key1 = "k"
	key2 = "a"
	key3 = "b"
)

func TestReportPayloadCache_LRUByUpdatedAt_EvictionScenario(t *testing.T) {
	t.Parallel()

	const maxSlots = 2
	ttl := 24 * time.Hour
	c := newReportPayloadCache(maxSlots, ttl)

	vm0, vm1, vm2 := "vm0", "vm1", "vm2"

	t0 := time.Unix(100, 0)
	t1 := time.Unix(200, 0)
	t2 := time.Unix(300, 0)
	t3 := time.Unix(400, 0)
	t4 := time.Unix(500, 0)

	payload0 := relaytest.NewTestVMReport("cid0")
	payload1 := relaytest.NewTestVMReport("cid1")
	payload2 := cloneVMReport(t, payload0)
	payload2.DiscoveredData.OsVersion = "changed"
	payload3 := relaytest.NewTestVMReport("cid3")

	c.Upsert(vm0, payload0, t0)
	c.Upsert(vm1, payload1, t1)

	// Identical payload for vm0 — no recency / updatedAt change.
	c.Upsert(vm0, cloneVMReport(t, payload0), t2)

	c.Upsert(vm0, payload2, t3)

	c.Upsert(vm2, payload3, t4)

	got0, ok0 := c.Get(vm0, t4)
	require.Truef(t, ok0, "expected vm0 cache lookup to be a hit, but got a miss")
	require.Truef(t, got0.EqualVT(payload2), "expected vm0 cached payload to match payload2, but it did not")

	got1, ok1 := c.Get(vm1, t4)
	require.Falsef(t, ok1, "expected vm1 cache lookup to be a miss, but got a hit")
	require.Nilf(t, got1, "expected vm1 cached payload to be nil, got %v", got1)

	got2, ok2 := c.Get(vm2, t4)
	require.Truef(t, ok2, "expected vm2 cache lookup to be a hit, but got a miss")
	require.Truef(t, got2.EqualVT(payload3), "expected vm2 cached payload to match payload3, but it did not")
}

func TestReportPayloadCache_Upsert_StoresClone(t *testing.T) {
	t.Parallel()

	c := newReportPayloadCache(4, time.Hour)
	now := time.Unix(20, 0)
	r := relaytest.NewTestVMReport("upsert-clone")
	expected := r.CloneVT()

	c.Upsert(key1, r, now)
	r.DiscoveredData.OsVersion = "mutated-after-upsert"

	got, ok := c.Get(key1, now)
	require.Truef(t, ok, "expected a cache hit, but got a miss")
	require.Truef(t, got.EqualVT(expected), "expected cached payload to match the pre-mutation clone, but it did not")
}

func TestReportPayloadCache_Get_ReturnsCachedReference(t *testing.T) {
	t.Parallel()

	c := newReportPayloadCache(4, time.Hour)
	now := time.Unix(10, 0)
	r := relaytest.NewTestVMReport("x")
	c.Upsert(key1, r, now)

	out, ok := c.Get(key1, now)
	require.Truef(t, ok, "expected initial cache lookup to be a hit, but got a miss")
	require.Truef(t, out.EqualVT(r), "expected initial cached payload to match inserted payload, but it did not")
	changedOsVersion := "mutated"
	out.DiscoveredData.OsVersion = changedOsVersion

	out2, ok2 := c.Get(key1, now)
	require.Truef(t, ok2, "expected second cache lookup to be a hit, but got a miss")
	got := out2.GetDiscoveredData().GetOsVersion()
	require.Equalf(t, changedOsVersion, got, "expected second cached OS version %q, got %q", changedOsVersion, got)
}

func TestReportPayloadCache_Get_TTLExpiry_DoesNotEvict(t *testing.T) {
	t.Parallel()

	c := newReportPayloadCache(4, 10*time.Minute)
	base := time.Unix(1000, 0)
	r := relaytest.NewTestVMReport("x")
	c.Upsert(key1, r, base)

	_, ok := c.Get(key1, base.Add(10*time.Minute).Add(-time.Nanosecond))
	require.Truef(t, ok, "expected pre-expiry lookup to be a hit, but got a miss")

	got, ok := c.Get(key1, base.Add(10*time.Minute))
	require.Truef(t, ok, "expected expiry-boundary lookup to still be a hit, but got a miss")
	require.Truef(t, got.EqualVT(r), "expected cached payload at ttl boundary to match inserted payload, but it did not")
	require.Equalf(t, 1, c.Len(), "expected expired entry to remain cached until sweep/upsert, got len=%d", c.Len())

	evictions := c.SweepExpired(base.Add(10 * time.Minute))
	require.Lenf(t, evictions, 1, "expected sweep eviction count %d, got %d", 1, len(evictions))
	require.Equalf(
		t,
		key1,
		evictions[0].resourceID,
		"expected ttl eviction resourceID %q, got %q",
		key1,
		evictions[0].resourceID,
	)
	require.Equalf(
		t,
		10*time.Minute,
		evictions[0].residency,
		"expected ttl eviction residency %s, got %s",
		10*time.Minute,
		evictions[0].residency,
	)
	require.Equalf(
		t,
		10*time.Minute,
		evictions[0].lifetime,
		"expected ttl eviction lifetime %s, got %s",
		10*time.Minute,
		evictions[0].lifetime,
	)
	require.Equalf(t, 0, c.Len(), "expected cache length after sweep %d, got %d", 0, c.Len())
}

func TestReportPayloadCache_Upsert_ExpiresAtMostBudgetPerInsert(t *testing.T) {
	t.Parallel()

	const staleEntries = 20
	ttl := 10 * time.Minute
	c := newReportPayloadCache(64, ttl)
	base := time.Unix(5000, 0)
	for i := range staleEntries {
		c.Upsert(fmt.Sprintf("k-%02d", i), relaytest.NewTestVMReport(fmt.Sprintf("%02d", i)), base.Add(time.Duration(i)*time.Second))
	}
	require.Equalf(t, staleEntries, c.Len(), "expected stale entries count %d, got %d", staleEntries, c.Len())

	now := base.Add(ttl + time.Minute)
	evictions := c.Upsert("fresh", relaytest.NewTestVMReport("fresh"), now)

	require.Lenf(
		t,
		evictions,
		upsertExpiredEvictionPerInsert,
		"expected per-insert eviction cap %d, got %d",
		upsertExpiredEvictionPerInsert,
		len(evictions),
	)
	require.Equalf(
		t,
		staleEntries-upsertExpiredEvictionPerInsert+1,
		c.Len(),
		"expected cache length after bounded upsert sweep %d, got %d",
		staleEntries-upsertExpiredEvictionPerInsert+1,
		c.Len(),
	)

	remaining := c.SweepExpired(now)
	require.Lenf(
		t,
		remaining,
		staleEntries-upsertExpiredEvictionPerInsert,
		"expected remaining stale evictions %d, got %d",
		staleEntries-upsertExpiredEvictionPerInsert,
		len(remaining),
	)
	require.Equalf(t, 1, c.Len(), "expected cache to retain only fresh entry, got len=%d", c.Len())
	_, ok := c.Get("fresh", now)
	require.Truef(t, ok, "expected fresh entry lookup to be a hit, but got a miss")
}

func TestReportPayloadCache_LenAndCapacity(t *testing.T) {
	t.Parallel()

	c := newReportPayloadCache(5, time.Hour)
	require.Equalf(t, 5, c.Capacity(), "expected cache capacity %d, got %d", 5, c.Capacity())
	require.Equalf(t, 0, c.Len(), "expected initial cache length %d, got %d", 0, c.Len())

	now := time.Unix(10, 0)
	c.Upsert("x", relaytest.NewTestVMReport("x"), now)
	require.Equalf(t, 1, c.Len(), "expected cache length after one insert %d, got %d", 1, c.Len())
}

func TestReportPayloadCache_MaxSlotsNonPositive_NewKeyUpsertNoOps(t *testing.T) {
	t.Parallel()

	c := newReportPayloadCache(0, time.Hour)
	now := time.Unix(50, 0)
	c.Upsert("new", relaytest.NewTestVMReport("new"), now)
	require.Equalf(
		t,
		0,
		c.Len(),
		"expected zero-capacity cache length after insert %d, got %d",
		0,
		c.Len(),
	)
	_, ok := c.Get("new", now)
	require.Falsef(t, ok, "expected zero-capacity cache lookup to be a miss, but got a hit")

	cNeg := newReportPayloadCache(-3, time.Hour)
	cNeg.Upsert("other", relaytest.NewTestVMReport("other"), now)
	require.Equalf(
		t,
		0,
		cNeg.Len(),
		"expected negative-capacity cache length after insert %d, got %d",
		0,
		cNeg.Len(),
	)
}

func TestReportPayloadCache_SweepExpired_NoWork_UnchangedLen(t *testing.T) {
	t.Parallel()

	ttl := time.Hour
	now := time.Unix(5000, 0)

	t.Run("empty cache", func(t *testing.T) {
		t.Parallel()
		c := newReportPayloadCache(4, ttl)
		require.Equalf(t, 0, c.Len(), "expected empty-cache initial length %d, got %d", 0, c.Len())
		ev := c.SweepExpired(now)
		require.Emptyf(t, ev, "expected no empty-cache sweep evictions, got %v", ev)
		require.Equalf(t, 0, c.Len(), "expected empty-cache post-sweep length %d, got %d", 0, c.Len())
	})

	t.Run("all entries still within TTL", func(t *testing.T) {
		t.Parallel()
		c := newReportPayloadCache(4, ttl)
		base := time.Unix(6000, 0)
		c.Upsert(key2, relaytest.NewTestVMReport(key2), base)
		c.Upsert(key3, relaytest.NewTestVMReport(key3), base.Add(30*time.Minute))
		require.Equalf(
			t,
			2,
			c.Len(),
			"expected within-ttl cache length after inserts %d, got %d",
			2,
			c.Len(),
		)

		sweepAt := base.Add(20 * time.Minute)
		ev := c.SweepExpired(sweepAt)
		require.Emptyf(t, ev, "expected no within-ttl sweep evictions, got %v", ev)
		require.Equalf(
			t,
			2,
			c.Len(),
			"expected within-ttl cache length after sweep %d, got %d",
			2,
			c.Len(),
		)
		_, okA := c.Get(key2, sweepAt)
		require.Truef(t, okA, "expected within-ttl lookup for a to be a hit, but got a miss")
		_, okB := c.Get(key3, sweepAt)
		require.Truef(t, okB, "expected within-ttl lookup for b to be a hit, but got a miss")
	})
}

func TestReportPayloadCache_SweepExpired_MultipleExpired_OrderedOldestUpdatedFirst(t *testing.T) {
	t.Parallel()

	ttl := time.Hour
	c := newReportPayloadCache(4, ttl)
	base := time.Unix(8000, 0)
	rOld := relaytest.NewTestVMReport("older")
	rNew := relaytest.NewTestVMReport("newer")
	c.Upsert("older", rOld, base)
	c.Upsert("newer", rNew, base.Add(20*time.Minute))

	sweepAt := base.Add(90 * time.Minute)
	evictions := c.SweepExpired(sweepAt)
	require.Lenf(t, evictions, 2, "expected sweep eviction count %d, got %d", 2, len(evictions))
	require.Equalf(
		t,
		"older",
		evictions[0].resourceID,
		"expected first eviction resourceID %q, got %q",
		"older",
		evictions[0].resourceID,
	)
	require.Equalf(
		t,
		90*time.Minute,
		evictions[0].residency,
		"expected first eviction residency %s, got %s",
		90*time.Minute,
		evictions[0].residency,
	)
	require.Equalf(
		t,
		90*time.Minute,
		evictions[0].lifetime,
		"expected first eviction lifetime %s, got %s",
		90*time.Minute,
		evictions[0].lifetime,
	)
	require.Equalf(
		t,
		"newer",
		evictions[1].resourceID,
		"expected second eviction resourceID %q, got %q",
		"newer",
		evictions[1].resourceID,
	)
	require.Equalf(
		t,
		70*time.Minute,
		evictions[1].residency,
		"expected second eviction residency %s, got %s",
		70*time.Minute,
		evictions[1].residency,
	)
	require.Equalf(
		t,
		70*time.Minute,
		evictions[1].lifetime,
		"expected second eviction lifetime %s, got %s",
		70*time.Minute,
		evictions[1].lifetime,
	)
	require.Equalf(t, 0, c.Len(), "expected cache length after sweep %d, got %d", 0, c.Len())
}

func TestReportPayloadCache_Remove_ReturnsEvictionDurations(t *testing.T) {
	t.Parallel()

	c := newReportPayloadCache(4, time.Hour)
	base := time.Unix(1000, 0)
	r := relaytest.NewTestVMReport("r")
	c.Upsert(key1, r, base)

	ev, removed := c.Remove(key1, base.Add(7*time.Minute+3*time.Second))
	require.Truef(t, removed, "expected remove(existing key) to return true, but got false")
	require.Equalf(
		t,
		7*time.Minute+3*time.Second,
		ev.residency,
		"expected remove(existing key) residency %s, got %s",
		7*time.Minute+3*time.Second,
		ev.residency,
	)
	require.Equalf(
		t,
		7*time.Minute+3*time.Second,
		ev.lifetime,
		"expected remove(existing key) lifetime %s, got %s",
		7*time.Minute+3*time.Second,
		ev.lifetime,
	)
	require.Equalf(
		t,
		0,
		c.Len(),
		"expected cache length after remove(existing key) %d, got %d",
		0,
		c.Len(),
	)

	ev2, removed2 := c.Remove(key1, base)
	require.Falsef(t, removed2, "expected remove(missing key) to return false, but got true")
	require.Equalf(t, payloadEviction{}, ev2, "expected zero eviction for missing remove, got %+v", ev2)
}

func TestReportPayloadCache_SweepExpired_KeepsFreshRemovesExpired(t *testing.T) {
	t.Parallel()

	ttl := time.Hour
	c := newReportPayloadCache(4, ttl)
	base := time.Unix(2000, 0)

	oldReport := relaytest.NewTestVMReport("old")
	freshReport := relaytest.NewTestVMReport("fresh")
	c.Upsert("old", oldReport, base)
	c.Upsert("fresh", freshReport, base.Add(30*time.Minute))

	sweepAt := base.Add(80 * time.Minute)
	evictions := c.SweepExpired(sweepAt)

	require.Lenf(t, evictions, 1, "expected sweep eviction count %d, got %d", 1, len(evictions))
	require.Equalf(
		t,
		"old",
		evictions[0].resourceID,
		"expected evicted resourceID %q, got %q",
		"old",
		evictions[0].resourceID,
	)
	require.Equalf(
		t,
		80*time.Minute,
		evictions[0].residency,
		"expected evicted residency %s, got %s",
		80*time.Minute,
		evictions[0].residency,
	)
	require.Equalf(
		t,
		80*time.Minute,
		evictions[0].lifetime,
		"expected evicted lifetime %s, got %s",
		80*time.Minute,
		evictions[0].lifetime,
	)

	require.Equalf(t, 1, c.Len(), "expected cache length after sweep %d, got %d", 1, c.Len())
	gotFresh, ok := c.Get("fresh", sweepAt)
	require.Truef(t, ok, "expected fresh entry lookup to be a hit, but got a miss")
	require.Truef(t, gotFresh.EqualVT(freshReport), "expected fresh payload to match freshReport, but it did not")

	_, okOld := c.Get("old", sweepAt)
	require.Falsef(t, okOld, "expected old entry lookup to be a miss, but got a hit")
}

func TestReportPayloadCache_SweepExpired_ThenUpsert_EvictsLRUByUpdatedAt(t *testing.T) {
	t.Parallel()

	const maxSlots = 2
	ttl := time.Hour
	c := newReportPayloadCache(maxSlots, ttl)

	base := time.Unix(100, 0)
	payloadA := relaytest.NewTestVMReport(key2)
	payloadB := relaytest.NewTestVMReport(key3)
	payloadC := relaytest.NewTestVMReport("c")
	payloadD := relaytest.NewTestVMReport("d")

	c.Upsert(key2, payloadA, base)
	c.Upsert(key3, payloadB, base.Add(30*time.Minute))

	evictions := c.SweepExpired(base.Add(80 * time.Minute))
	require.Lenf(t, evictions, 1, "expected post-sweep eviction count %d, got %d", 1, len(evictions))
	require.Equalf(
		t,
		key2,
		evictions[0].resourceID,
		"expected post-sweep evicted resourceID %q, got %q",
		key2,
		evictions[0].resourceID,
	)

	// Fill slots again; inserting "d" evicts "b" by least-recently-updated (not TTL — "b" is still valid).
	c.Upsert("c", payloadC, base.Add(85*time.Minute))
	now := base.Add(86 * time.Minute)
	c.Upsert("d", payloadD, now)

	_, okA := c.Get(key2, now)
	require.Falsef(t, okA, "expected lookup for a to be a miss, but got a hit")
	_, okB := c.Get(key3, now)
	require.Falsef(t, okB, "expected lookup for b to be a miss, but got a hit")
	gotC, okC := c.Get("c", now)
	require.Truef(t, okC, "expected lookup for c to be a hit, but got a miss")
	require.Truef(t, gotC.EqualVT(payloadC), "expected payload for c to match payloadC, but it did not")
	gotD, okD := c.Get("d", now)
	require.Truef(t, okD, "expected lookup for d to be a hit, but got a miss")
	require.Truef(t, gotD.EqualVT(payloadD), "expected payload for d to match payloadD, but it did not")
}

func TestReportPayloadCache_Get_DoesNotPromoteRecency(t *testing.T) {
	t.Parallel()

	c := newReportPayloadCache(2, time.Hour)
	t0 := time.Unix(1, 0)
	t1 := time.Unix(2, 0)
	t2 := time.Unix(3, 0)

	c.Upsert(key2, relaytest.NewTestVMReport(key2), t0)
	c.Upsert(key3, relaytest.NewTestVMReport(key3), t1)

	// Read "a" (oldest by updatedAt) — must not move it to MRU.
	_, ok := c.Get(key2, t1)
	require.Truef(t, ok, "expected lookup for a before eviction to be a hit, but got a miss")

	// Adding "c" should evict LRU by updatedAt ("a"), not "b".
	c.Upsert("c", relaytest.NewTestVMReport("c"), t2)

	_, okA := c.Get(key2, t2)
	require.Falsef(t, okA, "expected lookup for a after eviction to be a miss, but got a hit")
	_, okB := c.Get(key3, t2)
	require.Truef(t, okB, "expected lookup for b after eviction to be a hit, but got a miss")
	_, okC := c.Get("c", t2)
	require.Truef(t, okC, "expected lookup for c after insert to be a hit, but got a miss")
}

func cloneVMReport(t *testing.T, r *v1.VMReport) *v1.VMReport {
	t.Helper()
	return r.CloneVT()
}
