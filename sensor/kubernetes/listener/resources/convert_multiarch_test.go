package resources

import (
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/images/utils"
	imageUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPopulateImageMetadataMultiArchChildDigest(t *testing.T) {
	// Realistic sha256 values for a multi-arch index (manifest list) digest and the
	// per-arch child manifest digests the container runtime would actually report.
	const (
		indexDigest = "sha256:a1b2c3d4e5f60718293a4b5c6d7e8f90112233445566778899aabbccddeeff00"
		amd64Child  = "sha256:00ffeeddccbbaa998877665544332211009f8e7d6c5b4a3928170615f4e3d2c1"
		arm64Child  = "sha256:0f1e2d3c4b5a69788796a5b4c3d2e1f00112233445566778899aabbccddeeff0"
		// configID models the bare image config digest a runtime reports for an image it
		// did not pull from a registry (locally built or preloaded).
		configID = "sha256:9f8e7d6c5b4a3928170615f4e3d2c100ffeeddccbbaa99887766554433221100"
	)

	const repo = "quay.io/example/multiarch"
	indexPin := repo + "@" + indexDigest

	// A registry/remote that differs from the spec pin, modeling a case where the
	// container runtime pulled the image from a different location (e.g. a mirror or
	// an unqualified-search-registry alias) than the spec references.
	const runtimeRepo = "mirror.example.com/platform/multiarch"

	cases := map[string]struct {
		multiArchFlag bool
		// unqualifiedFlag toggles features.UnqualifiedSearchRegistries, which enables
		// re-syncing the image name (registry/remote) with what the runtime reported.
		unqualifiedFlag bool
		// deployImage is the image reference pinned in the deployment/pod spec.
		deployImage string
		// imageID is the ID the container runtime reports in the container status.
		imageID string
		// expectedID is the digest expected to win on the resulting image record.
		expectedID string
		// expectedFullName is the expected image name FullName after processing.
		expectedFullName string
	}{
		"index digest pin, amd64 child reported, flag on": {
			multiArchFlag:    true,
			deployImage:      indexPin,
			imageID:          "docker-pullable://" + repo + "@" + amd64Child,
			expectedID:       amd64Child,
			expectedFullName: repo + "@" + amd64Child,
		},
		"index digest pin, arm64 child reported, flag on": {
			multiArchFlag:    true,
			deployImage:      indexPin,
			imageID:          "docker-pullable://" + repo + "@" + arm64Child,
			expectedID:       arm64Child,
			expectedFullName: repo + "@" + arm64Child,
		},
		"runtime digest equals spec digest is a no-op, flag on": {
			multiArchFlag:    true,
			deployImage:      indexPin,
			imageID:          "docker-pullable://" + repo + "@" + indexDigest,
			expectedID:       indexDigest,
			expectedFullName: indexPin,
		},
		"index digest pin, runtime reports different registry/remote, unqualified+flag on": {
			// With UnqualifiedSearchRegistries on, the multi-arch block first swaps in the
			// child digest, then updateImageWithNewerImageName re-points the name at the
			// registry/remote the runtime actually pulled from, re-normalizing the FullName
			// around the (now child) digest. The resulting name reflects the runtime
			// location carrying the child digest.
			multiArchFlag:    true,
			unqualifiedFlag:  true,
			deployImage:      indexPin,
			imageID:          "docker-pullable://" + runtimeRepo + "@" + amd64Child,
			expectedID:       amd64Child,
			expectedFullName: runtimeRepo + "@" + amd64Child,
		},
		"empty container image ID on index pin keeps index digest, flag on": {
			// A not-ready container reports an empty ImageID; the child-digest block must
			// short-circuit and leave the index digest and name untouched.
			multiArchFlag:    true,
			deployImage:      indexPin,
			imageID:          "",
			expectedID:       indexDigest,
			expectedFullName: indexPin,
		},
		"bare config digest is not a manifest, flag on": {
			// A bare config digest is not a manifest: no registry can serve it, so it must
			// never replace the pinned index digest even though it differs from the spec.
			multiArchFlag:    true,
			deployImage:      indexPin,
			imageID:          configID,
			expectedID:       indexDigest,
			expectedFullName: indexPin,
		},
		"docker-schemed bare config digest is not a manifest, flag on": {
			// Docker reports non-pullable image IDs as "docker://sha256:<config>". Removing
			// the scheme still leaves no repository, so the pin must be retained.
			multiArchFlag:    true,
			deployImage:      indexPin,
			imageID:          "docker://" + configID,
			expectedID:       indexDigest,
			expectedFullName: indexPin,
		},
		"index digest pin keeps index digest when flag off": {
			multiArchFlag:    false,
			deployImage:      indexPin,
			imageID:          "docker-pullable://" + repo + "@" + amd64Child,
			expectedID:       indexDigest,
			expectedFullName: indexPin,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			testutils.MustUpdateFeature(t, features.FlattenImageData, true)
			testutils.MustUpdateFeature(t, features.MultiArchWorkloadChildDigest, c.multiArchFlag)
			testutils.MustUpdateFeature(t, features.UnqualifiedSearchRegistries, c.unqualifiedFlag)

			img, err := imageUtils.GenerateImageFromString(c.deployImage)
			require.NoError(t, err)

			wrap := deploymentWrap{
				Deployment: &storage.Deployment{},
			}
			wrap.Containers = append(wrap.Containers, &storage.Container{
				Name:  "container-0",
				Image: img,
			})

			k8sPod := &v1.Pod{}
			k8sPod.Spec.Containers = append(k8sPod.Spec.Containers, v1.Container{Name: "container-0", Image: c.deployImage})
			k8sPod.Status.ContainerStatuses = append(k8sPod.Status.ContainerStatuses, v1.ContainerStatus{
				Name:    "container-0",
				ImageID: c.imageID,
			})

			wrap.populateImageMetadata(set.StringSet{}, k8sPod)

			result := wrap.GetDeployment().GetContainers()[0].GetImage()
			assert.Equal(t, c.expectedID, result.GetId())

			if c.expectedFullName != "" {
				assert.Equal(t, c.expectedFullName, result.GetName().GetFullName())
				// IdV2 is derived from the (full) image name and the winning digest; compute
				// the expectation independently from the result's name to avoid tautology.
				expectedName := &storage.ImageName{FullName: c.expectedFullName}
				assert.Equal(t, utils.NewImageV2ID(expectedName, c.expectedID), result.GetIdV2())
			}
		})
	}
}

func TestPopulateImageMetadataMultiArchMultipleContainers(t *testing.T) {
	// A pod with two containers where only one is pinned by a multi-arch index
	// digest. The index-pinned container's runtime child digest differs from its
	// spec digest and must be rewritten to the child digest, while the sibling
	// container whose runtime digest already equals its spec digest must be left
	// unchanged. This proves the child-digest rewrite matches per container name.
	testutils.MustUpdateFeature(t, features.MultiArchWorkloadChildDigest, true)

	const (
		indexDigest     = "sha256:a1b2c3d4e5f60718293a4b5c6d7e8f90112233445566778899aabbccddeeff00"
		amd64Child      = "sha256:00ffeeddccbbaa998877665544332211009f8e7d6c5b4a3928170615f4e3d2c1"
		singleArchChild = "sha256:fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"
	)

	const (
		multiArchPin  = "quay.io/example/multiarch@" + indexDigest
		singleArchPin = "quay.io/example/singlearch@" + singleArchChild
	)

	multiArchImg, err := imageUtils.GenerateImageFromString(multiArchPin)
	require.NoError(t, err)
	singleArchImg, err := imageUtils.GenerateImageFromString(singleArchPin)
	require.NoError(t, err)

	wrap := deploymentWrap{
		Deployment: &storage.Deployment{},
	}
	wrap.Containers = []*storage.Container{
		{Name: "multiarch", Image: multiArchImg},
		{Name: "singlearch", Image: singleArchImg},
	}

	pod := &v1.Pod{
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{Name: "multiarch", Image: multiArchPin},
				{Name: "singlearch", Image: singleArchPin},
			},
		},
		Status: v1.PodStatus{
			ContainerStatuses: []v1.ContainerStatus{
				// Runtime reports a per-arch child manifest that differs from the index pin.
				{Name: "multiarch", ImageID: "docker-pullable://quay.io/example/multiarch@" + amd64Child},
				// Runtime reports the same digest the single-arch container was pinned by.
				{Name: "singlearch", ImageID: "docker-pullable://quay.io/example/singlearch@" + singleArchChild},
			},
		},
	}

	wrap.populateImageMetadata(set.StringSet{}, pod)

	// The index-pinned container is rewritten to the per-arch child digest.
	assert.Equal(t, amd64Child, wrap.GetDeployment().GetContainers()[0].GetImage().GetId())
	// The single-arch container (runtime digest == spec digest) is unchanged.
	assert.Equal(t, singleArchChild, wrap.GetDeployment().GetContainers()[1].GetImage().GetId())
}

func TestPopulateImageMetadataMultiArchMultiplePods(t *testing.T) {
	// Multiple pods of a single deployment are visited newest first. These cases cover the
	// guards around the child-digest rewrite: older rollout revisions, a different
	// repository with the same digest, mixed-architecture replicas, cluster locality, and
	// the non-repeatable "first usable observation wins" behavior.
	const (
		indexDigest = "sha256:a1b2c3d4e5f60718293a4b5c6d7e8f90112233445566778899aabbccddeeff00"
		oldIndex    = "sha256:b1c2d3e4f5a60719283a4b5c6d7e8f90112233445566778899aabbccddeeff01"
		amd64Child  = "sha256:00ffeeddccbbaa998877665544332211009f8e7d6c5b4a3928170615f4e3d2c1"
		arm64Child  = "sha256:0f1e2d3c4b5a69788796a5b4c3d2e1f00112233445566778899aabbccddeeff0"
		oldChild    = "sha256:1111111111111111111111111111111111111111111111111111111111111111"
		configID    = "4444444444444444444444444444444444444444444444444444444444444444"
	)

	const (
		repo     = "quay.io/example/multiarch"
		oldRepo  = "quay.io/example/multiarch-old"
		indexPin = repo + "@" + indexDigest

		unrelatedLocal = "image-registry.openshift-image-registry.svc:5000/team/other@sha256:2222222222222222222222222222222222222222222222222222222222222222"
	)

	newerPod := metav1.NewTime(time.Now())
	olderPod := metav1.NewTime(time.Now().Add(-time.Hour))

	type testPod struct {
		spec    string
		imageID string
		created metav1.Time
	}

	cases := map[string]struct {
		deployImage          string
		unqualified          bool
		pods                 []testPod
		localImages          []string
		expectedID           string
		expectedName         string
		expectedRegistry     string
		expectedRemote       string
		expectPullable       bool
		expectedClusterLocal bool
	}{
		"older pod from a previous revision cannot replace the newest digest": {
			deployImage: indexPin,
			pods: []testPod{
				{spec: oldRepo + "@" + oldIndex, imageID: "docker-pullable://" + oldRepo + "@" + oldChild, created: olderPod},
				{spec: indexPin, imageID: "docker-pullable://" + repo + "@" + amd64Child, created: newerPod},
			},
			expectedID:   amd64Child,
			expectedName: repo + "@" + amd64Child,
		},
		"single old-revision pod cannot supply the child digest": {
			// The deployment is pinned to indexDigest but its only pod still carries an older
			// revision's digest. The spec-digest comparison alone must reject it, isolating
			// that check from the "already rewritten" check.
			deployImage: indexPin,
			pods: []testPod{
				{spec: oldRepo + "@" + oldIndex, imageID: "docker-pullable://" + oldRepo + "@" + oldChild, created: newerPod},
			},
			expectedID:   indexDigest,
			expectedName: indexPin,
		},
		"different repository with the same digest cannot supply the child digest": {
			// The image still carries the pinned digest, so the digest check passes; only the
			// repository comparison against the pinned reference rejects this pod. This also
			// guards that matching on repository plus digest did not loosen the check.
			deployImage: indexPin,
			pods: []testPod{
				{spec: oldRepo + "@" + indexDigest, imageID: "docker-pullable://" + oldRepo + "@" + amd64Child, created: newerPod},
			},
			expectedID:   indexDigest,
			expectedName: indexPin,
		},
		"a tagged pod reference with the same repository and pin can supply the child digest": {
			// Kubernetes accepts both repo:tag@digest and repo@digest for one image, and the
			// two parse to different full names. Pin matching must compare the normalized
			// repository and the digest, so the redundant tag does not disqualify this pod.
			deployImage: indexPin,
			pods: []testPod{
				{spec: repo + ":1.0@" + indexDigest, imageID: "docker-pullable://" + repo + "@" + amd64Child, created: newerPod},
			},
			expectedID:   amd64Child,
			expectedName: repo + "@" + amd64Child,
		},
		"an older replica reporting the same child from another repository cannot change the repository": {
			// Both pods match the pin and report the same child digest, but the older one
			// pulled it through a mirror. The first matching pod with a non-empty image ID
			// supplies the runtime name; a later pod must not flip the repository, and
			// therefore IdV2, even though the digest is identical.
			unqualified: true,
			deployImage: indexPin,
			pods: []testPod{
				{spec: indexPin, imageID: "docker-pullable://mirror.example.com/platform/multiarch@" + amd64Child, created: olderPod},
				{spec: indexPin, imageID: "docker-pullable://" + repo + "@" + amd64Child, created: newerPod},
			},
			expectedID:       amd64Child,
			expectedName:     repo + "@" + amd64Child,
			expectedRegistry: "quay.io",
			expectedRemote:   "example/multiarch",
			expectPullable:   true,
		},
		"mixed-arch replicas retain the newest observation within one build": {
			// One deployment with replicas on both architectures can only carry one image
			// per container in the current data model. Within a single build the newest pod's
			// digest wins; the rewrite is not repeatable across builds (see convert.go).
			// Representing both digests requires a schema change (out of scope).
			deployImage: indexPin,
			pods: []testPod{
				{spec: indexPin, imageID: "docker-pullable://" + repo + "@" + arm64Child, created: olderPod},
				{spec: indexPin, imageID: "docker-pullable://" + repo + "@" + amd64Child, created: newerPod},
			},
			expectedID:   amd64Child,
			expectedName: repo + "@" + amd64Child,
		},
		"cluster-local image stays cluster-local after the rewrite": {
			deployImage: indexPin,
			// localImages is keyed by the pinned reference, as the resolver builds it
			// before any rewrite has happened.
			localImages: []string{indexPin},
			// The second, older pod is processed after the newest pod rewrote the name, so
			// it exercises the spec-reference fallback in the locality lookup.
			pods: []testPod{
				{spec: indexPin, imageID: "docker-pullable://" + repo + "@" + amd64Child, created: olderPod},
				{spec: indexPin, imageID: "docker-pullable://" + repo + "@" + amd64Child, created: newerPod},
			},
			expectedID:           amd64Child,
			expectedName:         repo + "@" + amd64Child,
			expectedClusterLocal: true,
		},
		// The case of an already-rewritten image whose resolver-built localImages are keyed
		// by the child digest is covered by TestPopulateImageMetadataMultiArchResolverRebuild,
		// which models the rebuild on the same mutated wrap instead of a fresh wrap.
		"an unrelated older pod cannot make the image cluster-local": {
			// An older pod's container shares the container name but references an unrelated
			// cluster-local image. Locality must be decided from the pinned reference only,
			// otherwise a stale pod would flip the image to cluster-local.
			deployImage: indexPin,
			localImages: []string{unrelatedLocal},
			pods: []testPod{
				{spec: unrelatedLocal, imageID: "docker-pullable://" + unrelatedLocal, created: olderPod},
				{spec: indexPin, imageID: "docker-pullable://" + repo + "@" + amd64Child, created: newerPod},
			},
			expectedID:           amd64Child,
			expectedName:         repo + "@" + amd64Child,
			expectedClusterLocal: false,
		},
		"a same-revision pod with a bare config id cannot override the selected observation": {
			// Both pods match the pin, so neither is stale. Once the newest pod has recorded a
			// child digest, an older same-revision pod reporting only a config id must not
			// contribute pullability or the runtime image name: with unqualified search
			// registries on, that would blank registry/remote and mark the image not pullable.
			unqualified: true,
			deployImage: indexPin,
			pods: []testPod{
				{spec: indexPin, imageID: "docker://sha256:" + configID, created: olderPod},
				{spec: indexPin, imageID: "docker-pullable://" + repo + "@" + amd64Child, created: newerPod},
			},
			expectedID:       amd64Child,
			expectedName:     repo + "@" + amd64Child,
			expectedRegistry: "quay.io",
			expectedRemote:   "example/multiarch",
			expectPullable:   true,
		},
		"an arm64 replica cannot relabel the record selected from the amd64 pod": {
			// A same-revision replica on another architecture, pulled through a mirror, must
			// change neither the digest nor the repository of the record already selected.
			unqualified: true,
			deployImage: indexPin,
			pods: []testPod{
				{spec: indexPin, imageID: "docker-pullable://mirror.example.com/platform/multiarch@" + arm64Child, created: olderPod},
				{spec: indexPin, imageID: "docker-pullable://" + repo + "@" + amd64Child, created: newerPod},
			},
			expectedID:       amd64Child,
			expectedName:     repo + "@" + amd64Child,
			expectedRegistry: "quay.io",
			expectedRemote:   "example/multiarch",
			expectPullable:   true,
		},
		"an older pod may still supply the child digest when the newest reports only the index (invariant)": {
			// Nothing is locked until a child digest has actually been recorded. This passes
			// with and without the change; it guards against a future "first pod wins"
			// implementation and is not a regression guard for this one.
			deployImage: indexPin,
			pods: []testPod{
				{spec: indexPin, imageID: "docker-pullable://" + repo + "@" + arm64Child, created: olderPod},
				{spec: indexPin, imageID: "docker-pullable://" + repo + "@" + indexDigest, created: newerPod},
			},
			expectedID:   arm64Child,
			expectedName: repo + "@" + arm64Child,
		},
		"a cluster-local deployment stays cluster-local when only older-revision pods report": {
			// Locality is a property of the deployment's declared reference, not of any pod, so
			// a deployment whose only pod is on an older revision must still be reported local.
			deployImage: indexPin,
			localImages: []string{indexPin},
			pods: []testPod{
				{spec: oldRepo + "@" + oldIndex, imageID: "docker-pullable://" + oldRepo + "@" + oldChild, created: newerPod},
			},
			expectedID:           indexDigest,
			expectedName:         indexPin,
			expectedClusterLocal: true,
		},
		"runtime without a digest keeps the pinned digest (invariant)": {
			// A tag-form runtime image ID yields no digest both before and after the fix, so
			// this is a safety net rather than a regression guard.
			deployImage: indexPin,
			pods: []testPod{
				{spec: indexPin, imageID: "docker-pullable://" + repo + ":1.0", created: newerPod},
			},
			expectedID:   indexDigest,
			expectedName: indexPin,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			testutils.MustUpdateFeature(t, features.FlattenImageData, true)
			testutils.MustUpdateFeature(t, features.MultiArchWorkloadChildDigest, true)
			testutils.MustUpdateFeature(t, features.UnqualifiedSearchRegistries, c.unqualified)

			img, err := imageUtils.GenerateImageFromString(c.deployImage)
			require.NoError(t, err)

			wrap := deploymentWrap{
				Deployment: &storage.Deployment{},
			}
			wrap.Containers = []*storage.Container{
				{Name: "container-0", Image: img},
			}

			localImages := set.NewStringSet()
			for _, li := range c.localImages {
				localImages.Add(li)
			}

			pods := make([]*v1.Pod, 0, len(c.pods))
			for _, p := range c.pods {
				pods = append(pods, &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{CreationTimestamp: p.created},
					Spec: v1.PodSpec{
						Containers: []v1.Container{{Name: "container-0", Image: p.spec}},
					},
					Status: v1.PodStatus{
						ContainerStatuses: []v1.ContainerStatus{{Name: "container-0", ImageID: p.imageID}},
					},
				})
			}

			wrap.populateImageMetadata(localImages, pods...)

			result := wrap.GetDeployment().GetContainers()[0].GetImage()
			assert.Equal(t, c.expectedID, result.GetId())
			assert.Equal(t, c.expectedName, result.GetName().GetFullName())
			assert.Equal(t, c.expectedClusterLocal, result.GetIsClusterLocal())
			if c.expectedRegistry != "" {
				assert.Equal(t, c.expectedRegistry, result.GetName().GetRegistry())
				assert.Equal(t, c.expectedRemote, result.GetName().GetRemote())
			}
			if c.expectPullable {
				assert.False(t, result.GetNotPullable())
			}

			// Compute IdV2 independently from the expected name and digest.
			expectedName := &storage.ImageName{FullName: c.expectedName}
			assert.Equal(t, utils.NewImageV2ID(expectedName, c.expectedID), result.GetIdV2())
		})
	}
}

func TestPopulateImageMetadataMultiArchTagWorkloadUnaffected(t *testing.T) {
	// A by-tag workload has no index digest to expand, so enabling the flag must leave
	// the resulting image record identical, including the digest, full name, and IdV2.
	const (
		repo      = "quay.io/example/multiarch"
		tagRef    = repo + ":1.0"
		newDigest = "sha256:00ffeeddccbbaa998877665544332211009f8e7d6c5b4a3928170615f4e3d2c1"
		oldDigest = "sha256:1111111111111111111111111111111111111111111111111111111111111111"
	)

	testutils.MustUpdateFeature(t, features.FlattenImageData, true)
	testutils.MustUpdateFeature(t, features.UnqualifiedSearchRegistries, false)

	run := func(flag bool) *storage.ContainerImage {
		testutils.MustUpdateFeature(t, features.MultiArchWorkloadChildDigest, flag)

		img, err := imageUtils.GenerateImageFromString(tagRef)
		require.NoError(t, err)

		wrap := deploymentWrap{
			Deployment: &storage.Deployment{},
		}
		wrap.Containers = []*storage.Container{
			{Name: "container-0", Image: img},
		}

		pods := []*v1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{CreationTimestamp: metav1.NewTime(time.Now().Add(-time.Hour))},
				Spec:       v1.PodSpec{Containers: []v1.Container{{Name: "container-0", Image: tagRef}}},
				Status: v1.PodStatus{ContainerStatuses: []v1.ContainerStatus{
					{Name: "container-0", ImageID: "docker-pullable://" + repo + "@" + oldDigest},
				}},
			},
			{
				ObjectMeta: metav1.ObjectMeta{CreationTimestamp: metav1.NewTime(time.Now())},
				Spec:       v1.PodSpec{Containers: []v1.Container{{Name: "container-0", Image: tagRef}}},
				Status: v1.PodStatus{ContainerStatuses: []v1.ContainerStatus{
					{Name: "container-0", ImageID: "docker-pullable://" + repo + "@" + newDigest},
				}},
			},
		}

		wrap.populateImageMetadata(set.NewStringSet(), pods...)
		return wrap.GetDeployment().GetContainers()[0].GetImage()
	}

	withFlag := run(true)
	withoutFlag := run(false)

	protoassert.Equal(t, withoutFlag, withFlag)
	assert.Equal(t, newDigest, withFlag.GetId())
	assert.Equal(t, tagRef, withFlag.GetName().GetFullName())
}

func TestPopulateImageMetadataMultiArchResolverRebuild(t *testing.T) {
	// BuildDeploymentWithDependencies calls populateDataFromPods again on the same stored,
	// already-mutated deploymentWrap. These cases run populateImageMetadata twice on one wrap
	// to prove that the second pass neither lets a stale pod contribute nor undoes a
	// legitimate rewrite.
	const (
		indexDigest = "sha256:a1b2c3d4e5f60718293a4b5c6d7e8f90112233445566778899aabbccddeeff00"
		oldIndex    = "sha256:b1c2d3e4f5a60719283a4b5c6d7e8f90112233445566778899aabbccddeeff01"
		amd64Child  = "sha256:00ffeeddccbbaa998877665544332211009f8e7d6c5b4a3928170615f4e3d2c1"
		staleChild  = "sha256:1111111111111111111111111111111111111111111111111111111111111111"
		oldChild    = "sha256:3333333333333333333333333333333333333333333333333333333333333333"
	)
	const (
		repo     = "quay.io/example/multiarch"
		indexPin = repo + "@" + indexDigest
	)

	newerPod := metav1.NewTime(time.Now())
	olderPod := metav1.NewTime(time.Now().Add(-time.Hour))

	type testPod struct {
		spec    string
		imageID string
		created metav1.Time
	}

	cases := map[string]struct {
		unqualified          bool
		localImages          []string
		pods                 []testPod
		expectedID           string
		expectedName         string
		expectedRegistry     string
		expectedRemote       string
		expectedClusterLocal bool
	}{
		"older pod cannot supply a child digest on the resolver rebuild": {
			// The older pod carries the pinned digest under a different repository. With
			// unqualified search registries on, the first pass used to rewrite the image's
			// registry and remote to that older pod's, which made the second pass see the
			// stale pod's reference as the one the deployment declared and accept its child
			// digest. Matching against the captured pin, which no pass can mutate, is what
			// keeps both passes rejecting it.
			unqualified: true,
			pods: []testPod{
				{spec: indexPin, imageID: "docker-pullable://" + indexPin, created: newerPod},
				{spec: "docker.io/example/multiarch@" + indexDigest, imageID: "docker-pullable://docker.io/example/multiarch@" + staleChild, created: olderPod},
			},
			expectedID:   indexDigest,
			expectedName: indexPin,
		},
		"a legitimate rewrite survives the rebuild and is not re-applied (invariant)": {
			// Holds with and without the fix, so this is a safety net rather than a
			// regression guard: it pins down that a second pass is idempotent.
			pods: []testPod{
				{spec: indexPin, imageID: "docker-pullable://" + repo + "@" + amd64Child, created: newerPod},
			},
			expectedID:   amd64Child,
			expectedName: repo + "@" + amd64Child,
		},
		"cluster-local survives the rebuild (invariant)": {
			// The resolver keys localImages by the pinned reference; after the first pass the
			// image name has been rewritten, so the second pass must still find the pin.
			// This already held before the fix, via the pod-spec fallback that the fix
			// replaced with the captured pin, so it guards the behavior, not the change.
			localImages: []string{indexPin},
			pods: []testPod{
				{spec: indexPin, imageID: "docker-pullable://" + repo + "@" + amd64Child, created: newerPod},
			},
			expectedID:           amd64Child,
			expectedName:         repo + "@" + amd64Child,
			expectedClusterLocal: true,
		},
		"locality keyed by the child reference resolves after the rebuild (invariant)": {
			// localImages holds only the child reference, which no pass can know before the
			// rewrite. The first pass cannot report locality; the second must find it through
			// the now-rewritten live name. This closes a coverage gap rather than guarding
			// this change: every other fixture keys localImages by the pin, so the pinned-key
			// fallback alone could satisfy them. It passes before and after the fix.
			localImages: []string{repo + "@" + amd64Child},
			pods: []testPod{
				{spec: indexPin, imageID: "docker-pullable://" + repo + "@" + amd64Child, created: newerPod},
			},
			expectedID:           amd64Child,
			expectedName:         repo + "@" + amd64Child,
			expectedClusterLocal: true,
		},
		"the selected repository survives the rebuild": {
			// Two pods match the pin and report the same child digest from different pull
			// repositories. The repository chosen by the first pass must be the one still
			// reported after the second pass re-runs over the already-mutated wrap.
			unqualified: true,
			pods: []testPod{
				{spec: indexPin, imageID: "docker-pullable://" + repo + "@" + amd64Child, created: newerPod},
				{spec: indexPin, imageID: "docker-pullable://mirror.example.com/platform/multiarch@" + amd64Child, created: olderPod},
			},
			expectedID:       amd64Child,
			expectedName:     repo + "@" + amd64Child,
			expectedRegistry: "quay.io",
			expectedRemote:   "example/multiarch",
		},
		"an older pod cannot overwrite the runtime image name": {
			// With unqualified search registries on, a stale pod must not re-point the
			// registry/remote at the location its own runtime reported.
			unqualified: true,
			pods: []testPod{
				{spec: indexPin, imageID: "docker-pullable://" + repo + "@" + amd64Child, created: newerPod},
				{spec: "quay.io/example/multiarch-old@" + oldIndex, imageID: "docker-pullable://docker.io/other/multiarch@" + oldChild, created: olderPod},
			},
			expectedID:       amd64Child,
			expectedName:     repo + "@" + amd64Child,
			expectedRegistry: "quay.io",
			expectedRemote:   "example/multiarch",
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			testutils.MustUpdateFeature(t, features.FlattenImageData, true)
			testutils.MustUpdateFeature(t, features.MultiArchWorkloadChildDigest, true)
			testutils.MustUpdateFeature(t, features.UnqualifiedSearchRegistries, c.unqualified)

			img, err := imageUtils.GenerateImageFromString(indexPin)
			require.NoError(t, err)
			wrap := deploymentWrap{Deployment: &storage.Deployment{}}
			wrap.Containers = []*storage.Container{{Name: "container-0", Image: img}}

			localImages := set.NewStringSet()
			for _, li := range c.localImages {
				localImages.Add(li)
			}
			pods := make([]*v1.Pod, 0, len(c.pods))
			for _, p := range c.pods {
				pods = append(pods, &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{CreationTimestamp: p.created},
					Spec:       v1.PodSpec{Containers: []v1.Container{{Name: "container-0", Image: p.spec}}},
					Status: v1.PodStatus{ContainerStatuses: []v1.ContainerStatus{
						{Name: "container-0", ImageID: p.imageID},
					}},
				})
			}

			// Two passes on the same wrap, as the resolver rebuild does.
			wrap.populateImageMetadata(localImages, pods...)
			wrap.populateImageMetadata(localImages, pods...)

			result := wrap.GetDeployment().GetContainers()[0].GetImage()
			assert.Equal(t, c.expectedID, result.GetId())
			assert.Equal(t, c.expectedName, result.GetName().GetFullName())
			assert.Equal(t, c.expectedClusterLocal, result.GetIsClusterLocal())
			if c.expectedRegistry != "" {
				assert.Equal(t, c.expectedRegistry, result.GetName().GetRegistry())
				assert.Equal(t, c.expectedRemote, result.GetName().GetRemote())
			}
			expectedName := &storage.ImageName{FullName: c.expectedName}
			assert.Equal(t, utils.NewImageV2ID(expectedName, c.expectedID), result.GetIdV2())
		})
	}
}

func TestRuntimeChildDigest(t *testing.T) {
	const (
		child = "sha256:00ffeeddccbbaa998877665544332211009f8e7d6c5b4a3928170615f4e3d2c1"
		hex64 = "9f8e7d6c5b4a3928170615f4e3d2c100ffeeddccbbaa99887766554433221100"
	)

	cases := map[string]struct {
		imageID  string
		expected string
	}{
		"repository path component named sha256 is not mistaken for the digest": {
			imageID:  "quay.io/example/sha256:1.0@" + child,
			expected: child,
		},
		"docker-pullable digest reference": {
			imageID:  "docker-pullable://quay.io/example/multiarch@" + child,
			expected: child,
		},
		"bare config digest has no repository": {
			imageID: "sha256:" + hex64,
		},
		"docker-schemed bare config digest": {
			imageID: "docker://sha256:" + hex64,
		},
		"bare hex id without algorithm": {
			imageID: hex64,
		},
		"tag-only reference carries no digest": {
			imageID: "quay.io/example/multiarch:1.0",
		},
		"empty": {
			imageID: "",
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, c.expected, runtimeChildDigest(c.imageID))
		})
	}
}

func TestPopulateImageMetadataMultiArchConsistentObservation(t *testing.T) {
	// The digest, the repository it was read from and its pullability must all come from one
	// pod observation, and the answer must not change when the same wrap is built again.
	const (
		indexDigest = "sha256:a1b2c3d4e5f60718293a4b5c6d7e8f90112233445566778899aabbccddeeff00"
		amd64Child  = "sha256:00ffeeddccbbaa998877665544332211009f8e7d6c5b4a3928170615f4e3d2c1"
	)
	const (
		repo       = "quay.io/example/multiarch"
		mirrorRepo = "mirror.example.com/platform/multiarch"
		indexPin   = repo + "@" + indexDigest
	)

	newerPod := metav1.NewTime(time.Now())
	olderPod := metav1.NewTime(time.Now().Add(-time.Hour))

	type testPod struct {
		imageID string
		created metav1.Time
	}

	cases := map[string]struct {
		pods             []testPod
		expectedID       string
		expectedName     string
		expectedRegistry string
		expectedRemote   string
		expectedPullable bool
	}{
		"a child reported by an older pod supersedes an index observation from a mirror": {
			// The newest pod only confirms the index, but from a mirror, and is not pullable.
			// The older pod reports the child from the pinned repository. Taking the digest
			// from the older pod and the repository from the newest used to publish
			// mirror.example.com/platform/multiarch@<child>.
			pods: []testPod{
				{imageID: "docker://" + mirrorRepo + "@" + indexDigest, created: newerPod},
				{imageID: "docker-pullable://" + repo + "@" + amd64Child, created: olderPod},
			},
			expectedID:       amd64Child,
			expectedName:     repo + "@" + amd64Child,
			expectedRegistry: "quay.io",
			expectedRemote:   "example/multiarch",
			expectedPullable: true,
		},
		"index-only observations still supply the mirror metadata and keep the pin (invariant)": {
			// No pod reports a child, so the newest index observation is the fallback. This
			// passes with and without the change; it guards against over-correcting.
			pods: []testPod{
				{imageID: "docker-pullable://" + mirrorRepo + "@" + indexDigest, created: newerPod},
				{imageID: "docker-pullable://" + repo + "@" + indexDigest, created: olderPod},
			},
			expectedID:       indexDigest,
			expectedName:     mirrorRepo + "@" + indexDigest,
			expectedRegistry: "mirror.example.com",
			expectedRemote:   "platform/multiarch",
			expectedPullable: true,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			testutils.MustUpdateFeature(t, features.FlattenImageData, true)
			testutils.MustUpdateFeature(t, features.MultiArchWorkloadChildDigest, true)
			testutils.MustUpdateFeature(t, features.UnqualifiedSearchRegistries, true)

			build := func(passes int) *storage.ContainerImage {
				img, err := imageUtils.GenerateImageFromString(indexPin)
				require.NoError(t, err)
				wrap := deploymentWrap{Deployment: &storage.Deployment{}}
				wrap.Containers = []*storage.Container{{Name: "container-0", Image: img}}

				pods := make([]*v1.Pod, 0, len(c.pods))
				for _, p := range c.pods {
					pods = append(pods, &v1.Pod{
						ObjectMeta: metav1.ObjectMeta{CreationTimestamp: p.created},
						Spec:       v1.PodSpec{Containers: []v1.Container{{Name: "container-0", Image: indexPin}}},
						Status: v1.PodStatus{ContainerStatuses: []v1.ContainerStatus{
							{Name: "container-0", ImageID: p.imageID},
						}},
					})
				}
				for range passes {
					wrap.populateImageMetadata(set.NewStringSet(), pods...)
				}
				return wrap.GetDeployment().GetContainers()[0].GetImage()
			}

			single := build(1)
			assert.Equal(t, c.expectedID, single.GetId())
			assert.Equal(t, c.expectedName, single.GetName().GetFullName())
			assert.Equal(t, c.expectedRegistry, single.GetName().GetRegistry())
			assert.Equal(t, c.expectedRemote, single.GetName().GetRemote())
			assert.Equal(t, c.expectedPullable, !single.GetNotPullable())
			expectedName := &storage.ImageName{FullName: c.expectedName}
			assert.Equal(t, utils.NewImageV2ID(expectedName, c.expectedID), single.GetIdV2())

			// Building again over the same wrap must give the identical record.
			protoassert.Equal(t, single, build(2))
		})
	}
}

func TestDeploymentWrapCloneKeepsPinnedImages(t *testing.T) {
	const (
		indexDigest = "sha256:a1b2c3d4e5f60718293a4b5c6d7e8f90112233445566778899aabbccddeeff00"
		amd64Child  = "sha256:00ffeeddccbbaa998877665544332211009f8e7d6c5b4a3928170615f4e3d2c1"
	)
	const (
		repo        = "quay.io/example/multiarch"
		indexPin    = repo + "@" + indexDigest
		otherMirror = "other-mirror.example.com/platform/multiarch"
	)

	testutils.MustUpdateFeature(t, features.FlattenImageData, true)
	testutils.MustUpdateFeature(t, features.MultiArchWorkloadChildDigest, true)
	testutils.MustUpdateFeature(t, features.UnqualifiedSearchRegistries, true)

	newPod := func(imageID string) *v1.Pod {
		return &v1.Pod{
			Spec: v1.PodSpec{Containers: []v1.Container{{Name: "container-0", Image: indexPin}}},
			Status: v1.PodStatus{ContainerStatuses: []v1.ContainerStatus{
				{Name: "container-0", ImageID: imageID},
			}},
		}
	}

	img, err := imageUtils.GenerateImageFromString(indexPin)
	require.NoError(t, err)
	orig := &deploymentWrap{Deployment: &storage.Deployment{}}
	orig.Containers = []*storage.Container{{Name: "container-0", Image: img}}

	orig.populateImageMetadata(set.NewStringSet(), newPod("docker-pullable://"+repo+"@"+amd64Child))
	require.Equal(t, amd64Child, orig.GetDeployment().GetContainers()[0].GetImage().GetId(), "setup: image must be rewritten")

	clone := orig.Clone()

	t.Run("the clone records the declared reference, not the rewritten one", func(t *testing.T) {
		assert.Equal(t, pinnedImageRef{fullName: indexPin, repo: repo, digest: indexDigest}, clone.pinnedImages["container-0"])
	})

	t.Run("the clone's pins are a distinct map from the original's", func(t *testing.T) {
		clone.pinnedImages["container-0"] = pinnedImageRef{digest: "mutated"}
		clone.pinnedImages["extra"] = pinnedImageRef{digest: "extra"}
		assert.Equal(t, pinnedImageRef{fullName: indexPin, repo: repo, digest: indexDigest}, orig.pinnedImages["container-0"])
		assert.NotContains(t, orig.pinnedImages, "extra")
		// Restore, so the behavioral check below runs against an unmodified clone.
		clone.pinnedImages = map[string]pinnedImageRef{"container-0": orig.pinnedImages["container-0"]}
	})

	t.Run("a build over the clone behaves like a build over the original (invariant)", func(t *testing.T) {
		// The pod still declares the original index and now reports the same child from
		// another mirror. Because it still matches the pin, it contributes metadata on both.
		pod := newPod("docker-pullable://" + otherMirror + "@" + amd64Child)
		clone.populateImageMetadata(set.NewStringSet(), pod)
		orig.populateImageMetadata(set.NewStringSet(), pod)

		got := clone.GetDeployment().GetContainers()[0].GetImage()
		assert.Equal(t, amd64Child, got.GetId())
		assert.Equal(t, "other-mirror.example.com", got.GetName().GetRegistry())
		assert.Equal(t, "platform/multiarch", got.GetName().GetRemote())
		protoassert.Equal(t, orig.GetDeployment().GetContainers()[0].GetImage(), got)
	})
}
