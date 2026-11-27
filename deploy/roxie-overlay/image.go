package main

import "fmt"

type image int

const (
	mainImage image = iota
	centralDBImage
	scannerV4Image
	scannerV4DBImage
	collectorImage
	factImage
	operatorBundleImage

	numImages // sentinel, keep last
)

// name returns the image's repository name within the registry.
func (i image) name() string {
	switch i {
	case mainImage:
		return "main"
	case centralDBImage:
		return "central-db"
	case scannerV4Image:
		return "scanner-v4"
	case scannerV4DBImage:
		return "scanner-v4-db"
	case collectorImage:
		return "collector"
	case factImage:
		return "fact"
	case operatorBundleImage:
		return "stackrox-operator-bundle"
	default:
		panic(fmt.Sprintf("unknown image %d", i))
	}
}

// mainVersioned reports whether the image is tagged with the main image version,
// unlike the independently versioned collector/fact.
func (i image) mainVersioned() bool {
	switch i {
	case collectorImage, factImage:
		return false
	default:
		return true
	}
}
