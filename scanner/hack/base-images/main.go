package main

import (
	"flag"
	"fmt"
	"localhost/jvdm/image-puller-poc/jsoncache"
	"localhost/jvdm/image-puller-poc/registry"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

type BaseImage struct {
	Name         string
	LayerDigests []string
}

type baseImageResolver struct {
	client   *registry.Client
	authfile string // path to docker config.json (optional)
	maxTags  int
}

func newResolver(client *registry.Client, authfile string, maxTags int) *baseImageResolver {
	return &baseImageResolver{client: client, authfile: authfile, maxTags: maxTags}
}

func (r *baseImageResolver) fromRefs(refs []string) ([]BaseImage, error) {
	out := make([]BaseImage, 0, len(refs))
	for _, ref := range refs {
		payload, err := r.client.InspectRef(ref, r.authfile, true)
		if err != nil {
			return nil, err
		}
		out = append(out, BaseImage{
			Name:         ref,
			LayerDigests: append([]string(nil), payload.Layers...),
		})
	}
	return out, nil
}

// fromRepo returns the latest maxTags images from a repository path,
// selecting by Created timestamp after probing up to maxProbe candidate tags.
func (r *baseImageResolver) fromRepo(repoPath string, maxProbe int) ([]BaseImage, error) {
	tags, err := r.client.ListTags(repoPath)
	if err != nil {
		return nil, err
	}
	if len(tags) == 0 {
		return nil, nil
	}

	// Rank tags by "semver-ish" first, then lexicographically, to pick candidates.
	candidates := make([]string, len(tags))
	copy(candidates, tags)
	sort.SliceStable(candidates, func(i, j int) bool {
		ai := semverKey(candidates[i])
		aj := semverKey(candidates[j])
		if ai.valid != aj.valid {
			return ai.valid // valid semver ranks before non-semver
		}
		if ai.major != aj.major {
			return ai.major > aj.major
		}
		if ai.minor != aj.minor {
			return ai.minor > aj.minor
		}
		if ai.patch != aj.patch {
			return ai.patch > aj.patch
		}
		// If both are equal (or both non-semver), fall back to lex desc
		return candidates[i] > candidates[j]
	})
	if len(candidates) > maxProbe {
		candidates = candidates[:maxProbe]
	}

	type inspected struct {
		created time.Time
		tag     string
		layers  []string
	}
	ins := make([]inspected, 0, len(candidates))
	for _, tag := range candidates {
		ref := fmt.Sprintf("%s:%s", repoPath, tag)
		payload, err := r.client.InspectRef(ref, r.authfile, true)
		if err != nil {
			return nil, err
		}
		created := time.Time{}
		if payload.Created != nil {
			created = payload.Created.UTC()
		}
		ins = append(ins, inspected{
			created: created,
			tag:     tag,
			layers:  append([]string(nil), payload.Layers...),
		})
	}

	sort.SliceStable(ins, func(i, j int) bool {
		return ins[i].created.After(ins[j].created)
	})
	if len(ins) > r.maxTags {
		ins = ins[:r.maxTags]
	}

	out := make([]BaseImage, 0, len(ins))
	for _, it := range ins {
		out = append(out, BaseImage{
			Name:         fmt.Sprintf("%s:%s", repoPath, it.tag),
			LayerDigests: it.layers,
		})
	}
	return out, nil
}

// detectBaseImageFromLayers returns the candidate with the longest prefix match.
func detectBaseImageFromLayers(targetLayers []string, candidates []BaseImage) *BaseImage {
	var best *BaseImage
	bestLen := -1
	for idx := range candidates {
		b := &candidates[idx]
		n := len(b.LayerDigests)
		if n == 0 || n > len(targetLayers) {
			continue
		}
		match := true
		for i := 0; i < n; i++ {
			if targetLayers[i] != b.LayerDigests[i] {
				match = false
				break
			}
		}
		if match && n > bestLen {
			best = b
			bestLen = n
		}
	}
	return best
}

type arrayFlag []string

func (a *arrayFlag) String() string { return strings.Join(*a, ",") }
func (a *arrayFlag) Set(s string) error {
	*a = append(*a, s)
	return nil
}

type semver struct {
	valid               bool
	major, minor, patch int
}

var semverRe = regexp.MustCompile(`^v?(\d+)(?:\.(\d+))?(?:\.(\d+))?(?:[+-].*)?$`)

func semverKey(tag string) semver {
	m := semverRe.FindStringSubmatch(tag)
	if m == nil {
		return semver{valid: false}
	}
	atoi := func(s string) int {
		if s == "" {
			return 0
		}
		n := 0
		for i := 0; i < len(s); i++ {
			if s[i] < '0' || s[i] > '9' {
				return 0
			}
			n = n*10 + int(s[i]-'0')
		}
		return n
	}
	return semver{valid: true, major: atoi(m[1]), minor: atoi(m[2]), patch: atoi(m[3])}
}

func parsePlatform(p string) (v1.Platform, error) {
	parts := strings.SplitN(p, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return v1.Platform{}, fmt.Errorf("invalid platform %q, expected os/arch", p)
	}
	return v1.Platform{OS: parts[0], Architecture: parts[1]}, nil
}

func keychainFromAuthfile(authfile string) authn.Keychain {
	if authfile == "" {
		return authn.DefaultKeychain
	}
	// Expect path to .../config.json. Set DOCKER_CONFIG to its directory so
	// go-containerregistry's default keychain picks it up.
	dir := filepath.Dir(authfile)
	_ = os.Setenv("DOCKER_CONFIG", dir)
	return authn.DefaultKeychain
}

func main() {
	var (
		bases     arrayFlag
		baseRepos arrayFlag

		platformStr string
		authFile    string
		cacheDir    string
		dropTail    int
		maxTags     int
		maxProbe    int
	)

	flag.Var(&bases, "base", "candidate base image ref (repeatable)")
	flag.Var(&baseRepos, "base-repo", "registry repository path without tag/digest (repeatable), e.g. docker.io/library/ubuntu")

	flag.StringVar(&platformStr, "platform", "linux/amd64", "platform os/arch")
	flag.StringVar(&authFile, "auth-file", "", "path to Docker config.json for registry auth")
	flag.StringVar(&cacheDir, "cache-dir", "", "path to the layer cache directory")
	flag.IntVar(&dropTail, "drop-tail", 1, "treat last N layers as app layers")
	flag.IntVar(&maxTags, "max-tags", 10, "how many latest tags to use per --base-repo")
	flag.IntVar(&maxProbe, "max-probe", 50, "upper bound of tags to inspect per repo before selecting latest")
	flag.Parse()

	if cacheDir == "" {
		log.Fatal("missing cacheDir")
	}

	if len(flag.Args()) == 0 {
		log.Fatal("at least one target is required")
	}

	plt, err := parsePlatform(platformStr)
	if err != nil {
		log.Fatal(err)
	}

	cache := jsoncache.New(cacheDir)
	keychain := keychainFromAuthfile(authFile)
	client := registry.NewClient(cache, keychain, plt)
	resolver := newResolver(client, authFile, maxTags)

	// Build base candidates from explicit refs
	allBases, err := resolver.fromRefs(bases)
	if err != nil {
		log.Fatal(err)
	}
	// Expand base repos to recent tags
	for _, repo := range baseRepos {
		log.Printf("expanding repo: %s", repo)
		more, err := resolver.fromRepo(repo, maxProbe)
		if err != nil {
			log.Fatalf("failed to resolve tags from repo: %v", err)
		}
		allBases = append(allBases, more...)
	}

	// Analyze each target
	log.Printf("%v", os.Args)
	for _, ref := range flag.Args() {
		fmt.Printf("🕵️  %s\n", ref)
		payload, err := client.InspectRef(ref, authFile, true)
		if err != nil {
			log.Fatal(err)
		}
		targetLayers := append([]string(nil), payload.Layers...)
		if dropTail > 0 && dropTail < len(targetLayers) {
			targetLayers = targetLayers[:len(targetLayers)-dropTail]
		} else if dropTail >= len(targetLayers) {
			targetLayers = nil
		}

		best := detectBaseImageFromLayers(targetLayers, allBases)
		if best != nil {
			fmt.Printf("✅ %s\n", best.Name)
		} else {
			fmt.Println("❌ no match")
		}
	}
}
