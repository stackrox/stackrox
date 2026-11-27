package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	flag "github.com/spf13/pflag"
)

var imageRegistry string

func init() {
	log.SetFlags(0)
}

func main() {
	baseFlag := flag.String("base", "", "base tag to fall back to when overlay images are missing (auto-resolved if omitted)")
	flag.StringVar(&imageRegistry, "registry", "quay.io/stackrox-io", "image registry to look up images in")
	collectorTag := flag.String("collector-tag", "", "collector image tag to deploy instead of the operator's default")
	factTag := flag.String("fact-tag", "", "fact image tag to deploy instead of the operator's default")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: roxie-overlay [--registry <registry>] [--base <tag>] [--collector-tag <tag>] [--fact-tag <tag>] [<overlay-tag>]\n\n")
		fmt.Fprintf(os.Stderr, "If <overlay-tag> is omitted, MAIN_IMAGE_TAG is used.\n")
		fmt.Fprintf(os.Stderr, "If --base is omitted, the base tag is auto-resolved from recent git history.\n")
		fmt.Fprintf(os.Stderr, "Collector and fact use the deployed operator's defaults unless --collector-tag/--fact-tag are given.\n")
		fmt.Fprintf(os.Stderr, "Locally built collector/fact images are only transferred to the cluster when set explicitly\n")
		fmt.Fprintf(os.Stderr, "(see https://github.com/stackrox/roxie/issues/355).\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	var overlayTag string
	switch flag.NArg() {
	case 0:
		overlayTag = os.Getenv("MAIN_IMAGE_TAG")
		if overlayTag == "" {
			log.Fatalf("no overlay tag given and MAIN_IMAGE_TAG is not set")
		}
		log.Printf("Using overlay tag %s from the environment variable MAIN_IMAGE_TAG", overlayTag)
	case 1:
		overlayTag = flag.Arg(0)
	default:
		flag.Usage()
		os.Exit(1)
	}

	// Cancel in-flight lookups on Ctrl-C/SIGTERM instead of being killed mid-call.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, overlayTag, *baseFlag, *collectorTag, *factTag); err != nil {
		log.Fatal("Error: ", err)
	}
}

// resolution collects the outcome of resolving all images.
type resolution struct {
	rows      []tableRow
	tags      map[image]string
	onOverlay []string
	onBase    []string
}

func run(ctx context.Context, overlayTag, baseTag, collectorTag, factTag string) error {
	// Lazily evaluated base tag.
	base := NewBaseTag(imageRegistry, baseTag, overlayTag)

	resolvers := imageResolvers(base, overlayTag, collectorTag, factTag)
	res, err := resolveImages(ctx, resolvers, overlayTag)
	if err != nil {
		return err
	}

	printTable(res.rows, overlayTag)

	log.Println()
	if err := checkResolution(res, overlayTag); err != nil {
		return err
	}

	log.Println()
	return writeConfig(os.Stdout, imageRegistry, res.tags, base, overlayTag)
}

// Collector/fact are only included when a tag is given explicitly. Otherwise the deployed operator
// bundle's baked-in COLLECTOR_VERSION/FACT_VERSION apply: an overlay bundle carries those of the
// tree it was built from, a base bundle its own published ones.
func imageResolvers(base *BaseTag, overlayTag, collectorTag, factTag string) map[image]TagResolver {
	baseFallback := NewBaseFallbackResolver(base, imageRegistry, overlayTag, nil)
	resolvers := map[image]TagResolver{
		mainImage:           baseFallback,
		centralDBImage:      baseFallback,
		scannerV4Image:      baseFallback,
		scannerV4DBImage:    baseFallback,
		operatorBundleImage: NewBaseFallbackResolver(base, imageRegistry, overlayTag, toBundleTag),
	}
	if collectorTag != "" {
		resolvers[collectorImage] = NewFixedTagResolver(imageRegistry, collectorTag)
	}
	if factTag != "" {
		resolvers[factImage] = NewFixedTagResolver(imageRegistry, factTag)
	}
	return resolvers
}

func resolveImages(ctx context.Context, resolvers map[image]TagResolver, overlayTag string) (resolution, error) {
	res := resolution{tags: map[image]string{}}
	// Iterate in image declaration order for a stable table order.
	for img := range numImages {
		resolver, ok := resolvers[img]
		if !ok {
			continue
		}
		result, err := resolver.Resolve(ctx, img.name())
		if err != nil {
			return resolution{}, err
		}
		res.tags[img] = result.Tag
		res.rows = append(res.rows, tableRow{
			Image:    img,
			Tag:      result.Tag,
			Location: result.Location,
		})
		// Track overlay vs. base origin for the tag-mixing warning; collector/fact tags come from
		// neither, so they need to be excluded here.
		if img.mainVersioned() {
			if result.Tag == overlayTag {
				res.onOverlay = append(res.onOverlay, img.name())
			} else {
				res.onBase = append(res.onBase, img.name())
			}
		}
	}
	return res, nil
}
