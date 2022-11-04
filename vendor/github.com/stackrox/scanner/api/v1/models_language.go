package v1

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/stackrox/scanner/cpe"
	"github.com/stackrox/scanner/database"
	"github.com/stackrox/scanner/ext/versionfmt/language"
	v1 "github.com/stackrox/scanner/generated/scanner/api/v1"
	"github.com/stackrox/scanner/pkg/component"
)

// These are possible package prefixes or suffixes. Package managers sometimes annotate
// the packages with these e.g. urllib-python
var possiblePythonPrefixesOrSuffixes = []string{
	"python", "python2", "python3",
}

// languageFeatureValue is the value for a map of language features.
type languageFeatureValue struct {
	name, version, layer string
}

// getIgnoredLanguageComponents returns a map of language components to ignore.
// The map keeps track of the language components that were ignored because
// they came from the package manager.
//
// It is assumed the given layersToComponents is sorted in lowest (base layer) to highest layer.
// This is because modifications to the files in later layers may not have a package manager change associated (e.g. chown a JAR).
func getIgnoredLanguageComponents(layersToComponents []*component.LayerToComponents) map[string]languageFeatureValue {
	ignoredLanguageComponents := make(map[string]languageFeatureValue)
	for _, layerToComponent := range layersToComponents {
		for _, c := range layerToComponent.Components {
			if c.FromPackageManager {
				ignoredLanguageComponents[c.Location] = languageFeatureValue{
					name:    c.Name,
					version: c.Version,
				}
			}
		}
	}

	return ignoredLanguageComponents
}

// getLanguageData returns all application (language) features in the given layer.
// This data includes features which were introduced in lower (parent) layers.
// Since an image is based on a layered-filesystem, this function recognizes when files/locations
// have been removed, and does not return features from files from lower (parent) levels which have been deleted
// in higher (child) levels.
//
// A returned feature's AddedBy is the first (parent) layer that introduced the feature. For example,
// if a file was modified between layers in a way that the features it describes are untouched
// (ex: chown, touch), then the higher layer's features from that file are unused.
//
// A known issue is if a file defines multiple features, and the file is modified between layers in a way
// that does affect the features it describes (adds, updates, or removes features), which is currently only a
// concern for the Java source type. However, this event is unlikely, which is why it is not considered at this time.
func getLanguageData(db database.Datastore, layerName, lineage string, uncertifiedRHEL bool) ([]database.FeatureVersion, error) {
	layersToComponents, err := db.GetLayerLanguageComponents(layerName, lineage, &database.DatastoreOptions{
		UncertifiedRHEL: uncertifiedRHEL,
	})
	if err != nil {
		return nil, err
	}

	ignoredLanguageComponents := getIgnoredLanguageComponents(layersToComponents)

	languageFeatureMap := make(map[string]languageFeatureValue)
	var removedLanguageComponentLocations []string
	var features []database.FeatureVersion

	// Loop from the highest layer to lowest.
	for i := len(layersToComponents) - 1; i >= 0; i-- {
		layerToComponents := layersToComponents[i]

		// Ignore components which were removed in higher layers.
		components := layerToComponents.Components[:0]
		for _, c := range layerToComponents.Components {
			if c.FromPackageManager {
				continue
			}
			if lfv, ok := ignoredLanguageComponents[c.Location]; ok && lfv.name == c.Name && lfv.version == c.Version {
				continue
			}
			include := true
			for _, removedLocation := range removedLanguageComponentLocations {
				if strings.HasPrefix(c.Location, removedLocation) {
					include = false
					break
				}
			}

			if include {
				components = append(components, c)
			}
		}

		newFeatures := cpe.CheckForVulnerabilities(layerToComponents.Layer, components)
		for _, fv := range newFeatures {
			location := fv.Feature.Location
			featureValue := languageFeatureValue{
				name:    fv.Feature.Name,
				version: fv.Version,
				layer:   layerToComponents.Layer,
			}
			if existing, ok := languageFeatureMap[location]; ok {
				if featureValue.name != existing.name || featureValue.version != existing.version {
					// The contents at this location have changed between layers.
					// Use the higher layer's.
					continue
				}
			}

			features = append(features, fv)
			languageFeatureMap[location] = featureValue
		}

		removedLanguageComponentLocations = append(removedLanguageComponentLocations, layerToComponents.Removed...)
	}

	// We want to output the features in layer-order, so we must reverse the feature slice.
	// At the same time, we want to be sure to remove any repeat features that were not filtered previously
	// (this would be due us detecting a feature was introduced into the image at a lower level than originally thought).
	filtered := make([]database.FeatureVersion, 0, len(features))
	for i := len(features) - 1; i >= 0; i-- {
		feature := features[i]

		featureValue := languageFeatureMap[feature.Feature.Location]
		if feature.AddedBy.Name == featureValue.layer {
			filtered = append(filtered, feature)
		}
	}

	return filtered, nil
}

// getLanguageComponents returns the language components present in the image whose top layer is indicated by the given layerName.
func getLanguageComponents(db database.Datastore, layerName, lineage string, uncertifiedRHEL bool) []*component.Component {
	layersToComponents, err := db.GetLayerLanguageComponents(layerName, lineage, &database.DatastoreOptions{
		UncertifiedRHEL: uncertifiedRHEL,
	})
	if err != nil {
		log.Errorf("error getting language data: %v", err)
		return nil
	}

	var components []*component.Component
	var removedLanguageComponentLocations []string
	ignoredLanguageComponents := getIgnoredLanguageComponents(layersToComponents)
	// Loop from the highest layer to lowest.
	for i := len(layersToComponents) - 1; i >= 0; i-- {
		layerToComponents := layersToComponents[i]

		// Ignore components which were removed in higher layers.
		layerComponents := layerToComponents.Components[:0]
		for _, c := range layerToComponents.Components {
			if c.FromPackageManager {
				continue
			}
			if lfv, ok := ignoredLanguageComponents[c.Location]; ok && lfv.name == c.Name && lfv.version == c.Version {
				continue
			}
			include := true
			for _, removedLocation := range removedLanguageComponentLocations {
				if strings.HasPrefix(c.Location, removedLocation) {
					include = false
					break
				}
			}

			if include {
				c.AddedBy = layerToComponents.Layer
				layerComponents = append(layerComponents, c)
			}
		}

		removedLanguageComponentLocations = append(removedLanguageComponentLocations, layerToComponents.Removed...)

		components = append(components, layerComponents...)
	}

	// Keep the components in reverse-layer order, as this will be useful when performing
	// vulnerability matching.

	return components
}

func dedupeVersionMatcher(v1, v2 string) bool {
	if v1 == v2 {
		return true
	}
	return strings.HasPrefix(v2, v1)
}

func dedupeFeatureNameMatcher(feature Feature, osFeature Feature) bool {
	if feature.Name == osFeature.Name {
		return true
	}

	if feature.VersionFormat == component.PythonSourceType.String() {
		for _, ext := range possiblePythonPrefixesOrSuffixes {
			if feature.Name == strings.TrimPrefix(osFeature.Name, fmt.Sprintf("%s-", ext)) {
				return true
			}
			if feature.Name == strings.TrimSuffix(osFeature.Name, fmt.Sprintf("-%s", ext)) {
				return true
			}
		}
	}
	return false
}

func shouldDedupeLanguageFeature(feature Feature, osFeatures []Feature) bool {
	// Can probably sort this and it'll be faster
	for _, osFeature := range osFeatures {
		if dedupeFeatureNameMatcher(feature, osFeature) && dedupeVersionMatcher(feature.Version, osFeature.Version) {
			return true
		}
	}
	return false
}

// addLanguageVulns adds language-based features into the given layer.
// Assumes layer is not nil.
func addLanguageVulns(db database.Datastore, layer *Layer, lineage string, uncertifiedRHEL bool) {
	// Add Language Features
	languageFeatureVersions, err := getLanguageData(db, layer.Name, lineage, uncertifiedRHEL)
	if err != nil {
		log.Errorf("error getting language data: %v", err)
		return
	}

	var languageFeatures []Feature
	for _, dbFeatureVersion := range languageFeatureVersions {
		feature := featureFromDatabaseModel(dbFeatureVersion, uncertifiedRHEL, nil)
		if !shouldDedupeLanguageFeature(*feature, layer.Features) {
			updateFeatureWithVulns(feature, dbFeatureVersion.AffectedBy, language.ParserName)
			languageFeatures = append(languageFeatures, *feature)
		}
	}
	layer.Features = append(layer.Features, languageFeatures...)
}

func getLanguageFeatures(osFeatures []Feature, components []*v1.LanguageComponent, uncertifiedRHEL bool) ([]Feature, error) {
	layerToComponents := getLayerToComponents(components)

	languageFeatureMap := make(map[string]languageFeatureValue)
	var featureVersions []database.FeatureVersion

	for _, layer := range layerToComponents {
		fvs := cpe.CheckForVulnerabilities(layer.Layer, layer.Components)
		for _, fv := range fvs {
			location := fv.Feature.Location
			featureValue := languageFeatureValue{
				name:    fv.Feature.Name,
				version: fv.Version,
				layer:   layer.Layer,
			}
			if existing, ok := languageFeatureMap[location]; ok {
				if featureValue.name != existing.name || featureValue.version != existing.version {
					// The contents at this location have changed between layers.
					// Use the higher layer's.
					continue
				}
			}

			featureVersions = append(featureVersions, fv)
			languageFeatureMap[location] = featureValue
		}
	}

	// We want to output the features in layer-order, so we must reverse the feature slice.
	// At the same time, we want to be sure to remove any repeat features that were not filtered previously
	// (this would be due us detecting a feature was introduced into the image at a lower level than originally thought).
	features := make([]Feature, 0, len(featureVersions))
	for i := len(featureVersions) - 1; i >= 0; i-- {
		fv := featureVersions[i]

		featureValue := languageFeatureMap[fv.Feature.Location]
		if fv.AddedBy.Name == featureValue.layer {
			feature := featureFromDatabaseModel(fv, uncertifiedRHEL, nil)
			if !shouldDedupeLanguageFeature(*feature, osFeatures) {
				updateFeatureWithVulns(feature, fv.AffectedBy, language.ParserName)
				features = append(features, *feature)
			}
		}
	}

	return features, nil
}

// getLayerToComponents gets a slice of LayerToComponents from the given components.
// It is assumed the given components are sorted in order from the highest layer to the lowest layer.
func getLayerToComponents(components []*v1.LanguageComponent) []*component.LayerToComponents {
	var layers []*component.LayerToComponents
	var prevLayer string
	for _, languageComponent := range components {
		c := &component.Component{
			Name:     languageComponent.GetName(),
			Version:  languageComponent.GetVersion(),
			Location: languageComponent.GetLocation(),
			AddedBy:  languageComponent.GetAddedBy(),
		}

		var err error
		switch languageComponent.GetType() {
		case v1.SourceType_JAVA:
			c.SourceType = component.JavaSourceType
			c.JavaPkgMetadata, err = getJavaPkgMetadata(languageComponent.GetJava())
			if err != nil {
				log.Warnf("Unable to parse Java metadata: %v. skipping...", err)
				continue
			}
		case v1.SourceType_PYTHON:
			c.SourceType = component.PythonSourceType
			c.PythonPkgMetadata, err = getPythonPkgMetadata(languageComponent.GetPython())
			if err != nil {
				log.Warnf("Unable to parse Java metadata: %v. skipping...", err)
				continue
			}
		case v1.SourceType_NPM:
			c.SourceType = component.NPMSourceType
		case v1.SourceType_GEM:
			c.SourceType = component.GemSourceType
		case v1.SourceType_DOTNETCORERUNTIME:
			c.SourceType = component.DotNetCoreRuntimeSourceType
		}

		if prevLayer == "" || prevLayer != languageComponent.GetAddedBy() {
			prevLayer = languageComponent.GetAddedBy()
			layers = append(layers, &component.LayerToComponents{
				Layer: languageComponent.GetAddedBy(),
			})
		}

		bottomLayer := layers[len(layers)-1]
		bottomLayer.Components = append(bottomLayer.Components, c)
	}

	return layers
}

func getJavaPkgMetadata(java *v1.JavaComponent) (*component.JavaPkgMetadata, error) {
	if java == nil {
		return nil, errors.New("Java component is empty")
	}
	return &component.JavaPkgMetadata{
		ImplementationVersion: java.GetImplementationVersion(),
		MavenVersion:          java.GetMavenVersion(),
		Origins:               java.GetOrigins(),
		SpecificationVersion:  java.GetSpecificationVersion(),
		BundleName:            java.GetBundleName(),
	}, nil
}

func getPythonPkgMetadata(python *v1.PythonComponent) (*component.PythonPkgMetadata, error) {
	if python == nil {
		return nil, errors.New("Python component is empty")
	}
	return &component.PythonPkgMetadata{
		Homepage:    python.GetHomepage(),
		AuthorEmail: python.GetAuthorEmail(),
		DownloadURL: python.GetDownloadUrl(),
		Summary:     python.GetSummary(),
		Description: python.GetDescription(),
	}, nil
}
