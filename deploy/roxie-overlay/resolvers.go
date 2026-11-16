package main

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
)

type ResolveResult struct {
	Tag      string
	Location imageLocation
}

type TagResolver interface {
	Resolve(image string) (ResolveResult, error)
}

// BaseTag resolves the base tag once and shares it. If baseTag is empty,
// it auto-resolves from recent git history on first access.
type BaseTag struct {
	registry string
	tag      string
	// overlayTag is excluded from auto-resolution: its locally built main image would
	// qualify as a base, making images missing at the overlay "fall back" to the overlay.
	overlayTag string
}

func NewBaseTag(registry, baseTag, overlayTag string) *BaseTag {
	return &BaseTag{registry: registry, tag: baseTag, overlayTag: overlayTag}
}

// Peek returns the base tag if it has already been resolved, without triggering resolution.
func (b *BaseTag) Peek() (string, bool) {
	return b.tag, b.tag != ""
}

func (b *BaseTag) Get() (string, error) {
	if b.tag != "" {
		return b.tag, nil
	}
	tag, err := autoResolveBase(b.registry, b.overlayTag)
	if err != nil {
		return "", err
	}
	b.tag = tag
	return b.tag, nil
}

// baseFallbackResolver resolves an image tag by checking if the overlay
// tag exists, falling back to the shared base tag if it does not.
// The result is always the main-format tag; tagTransformer, if set, maps it to the
// image's actual tag for the lookup (e.g. the operator bundle's "v<operator tag>").
type baseFallbackResolver struct {
	base           *BaseTag
	registry       string
	overlayTag     string
	tagTransformer func(string) string
}

func NewBaseFallbackResolver(base *BaseTag, registry, overlayTag string, tagTransformer func(string) string) TagResolver {
	return &baseFallbackResolver{
		base:           base,
		registry:       registry,
		overlayTag:     overlayTag,
		tagTransformer: tagTransformer,
	}
}

func (r *baseFallbackResolver) locate(image, tag string) (imageLocation, error) {
	if r.tagTransformer != nil {
		tag = r.tagTransformer(tag)
	}
	return locateImageRef(r.registry, image, tag)
}

func (r *baseFallbackResolver) Resolve(image string) (ResolveResult, error) {
	loc, err := r.locate(image, r.overlayTag)
	if err != nil {
		return ResolveResult{}, err
	}
	if loc != imageNotFound {
		return ResolveResult{Tag: r.overlayTag, Location: loc}, nil
	}

	tag, err := r.base.Get()
	if err != nil {
		return ResolveResult{}, err
	}
	loc, err = r.locate(image, tag)
	if err != nil {
		return ResolveResult{}, err
	}
	return ResolveResult{Tag: tag, Location: loc}, nil
}

// fixedTagResolver uses a user-provided tag as is and only reports where the image is available.
// Used for collector & fact images.
type fixedTagResolver struct {
	registry string
	tag      string
}

func NewFixedTagResolver(registry, tag string) TagResolver {
	return &fixedTagResolver{registry: registry, tag: tag}
}

func (r *fixedTagResolver) Resolve(image string) (ResolveResult, error) {
	loc, err := locateImageRef(r.registry, image, r.tag)
	if err != nil {
		return ResolveResult{}, err
	}
	return ResolveResult{Tag: r.tag, Location: loc}, nil
}

// autoResolveBase walks recent git history to find the newest commit
// whose main image exists locally or in the remote registry.
func autoResolveBase(registry, exclude string) (string, error) {
	log.Printf("auto-resolving base tag...")

	// --first-parent stays on this branch's line and the one it branched from,
	// skipping commits of merged-in side branches.
	out, err := exec.Command("git", "log", "--first-parent", "--format=%h", "-50").Output()
	if err != nil {
		return "", fmt.Errorf("git log: %w", err)
	}

	for sha := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		if sha == "" {
			continue
		}
		tagOut, err := exec.Command("git", "describe", "--tags", "--abbrev=10", "--long", "--exclude", "*-nightly-*", sha).Output()
		if err != nil {
			continue
		}
		tag := strings.TrimSpace(string(tagOut))
		if tag == "" || tag == exclude {
			continue
		}
		loc, err := locateImageRef(registry, mainImage.name(), tag)
		if err != nil {
			return "", err
		}
		if loc != imageNotFound {
			log.Printf("✔  auto-resolved base: %s", tag)
			return tag, nil
		}
	}

	return "", fmt.Errorf("could not auto-resolve base tag: no recent commit has a published main image")
}
