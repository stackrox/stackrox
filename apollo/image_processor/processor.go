package imageprocessor

import (
	"fmt"
	"regexp"
	"sync"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	registryTypes "bitbucket.org/stack-rox/apollo/apollo/registries/types"
	scannerTypes "bitbucket.org/stack-rox/apollo/apollo/scanners/types"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
)

var (
	log = logging.New("imageprocessor/processor")
)

// ImageProcessor enriches and processes images to determine rule violations
type ImageProcessor struct {
	registries []registryTypes.ImageRegistry
	scanners   []scannerTypes.ImageScanner

	database db.Storage

	ruleMutex  sync.Mutex
	regexRules map[string]*regexImageRule
}

type regexImageRule struct {
	Registry *regexp.Regexp
	Tag      *regexp.Regexp
	Image    *regexp.Regexp
	Severity v1.Severity
	Name     string
}

func (i *ImageProcessor) addRegexImageRule(rule *v1.ImageRule) error {
	registry, err := regexp.Compile(rule.Registry)
	if err != nil {
		return fmt.Errorf("registry regex: %+v", err)
	}
	tag, err := regexp.Compile(rule.Tag)
	if err != nil {
		return fmt.Errorf("tag regex: %+v", err)
	}
	image, err := regexp.Compile(rule.Image)
	if err != nil {
		return fmt.Errorf("image regex: %+v", err)
	}
	i.regexRules[rule.Name] = &regexImageRule{
		Registry: registry,
		Tag:      tag,
		Image:    image,
		Name:     rule.Name,
	}
	return nil
}

// New creates a new image processor
func New(database db.Storage) (*ImageProcessor, error) {
	return &ImageProcessor{
		database: database,

		regexRules: make(map[string]*regexImageRule),
	}, nil
}

// UpdateRule updates the current rule in a threadsafe manner
func (i *ImageProcessor) UpdateRule(rule *v1.ImageRule) error {
	i.ruleMutex.Lock()
	defer i.ruleMutex.Unlock()
	return i.addRegexImageRule(rule)
}

// RemoveRule removes the rule specified by name in a threadsafe manner
func (i *ImageProcessor) RemoveRule(name string) {
	i.ruleMutex.Lock()
	defer i.ruleMutex.Unlock()
	delete(i.regexRules, name)
}

func matchRuleToImage(rule *regexImageRule, image *v1.Image) ([]*v1.Violation, bool) {
	var violations []*v1.Violation

	if rule.Image.MatchString(image.Repo) {
		violations = append(violations, &v1.Violation{
			Severity: rule.Severity,
			Message:  fmt.Sprintf("Rule %v matched image %v via image", rule.Image.String(), image.String()),
		})
	}
	if rule.Registry.MatchString(image.Registry) {
		violations = append(violations, &v1.Violation{
			Severity: rule.Severity,
			Message:  fmt.Sprintf("Rule %v matched image %v via registry", rule.Registry.String(), image.String()),
		})
	}
	if rule.Tag.MatchString(image.Tag) {
		violations = append(violations, &v1.Violation{
			Severity: rule.Severity,
			Message:  fmt.Sprintf("Rule %v matched image %v via tag", rule.Tag.String(), image.String()),
		})
	}
	return violations, len(violations) == 0
}

// Process takes in a new image and determines if an alert should be fired
func (i *ImageProcessor) Process(image *v1.Image) (*v1.Alert, error) {
	if err := i.enrichImage(image); err != nil {
		return nil, err
	}

	return i.checkImage(image)
}

func (i *ImageProcessor) checkImage(image *v1.Image) (*v1.Alert, error) {
	i.ruleMutex.Lock()
	defer i.ruleMutex.Unlock()

	var violations []*v1.Violation
	// TODO(cgorman) implement this violation logic coherently
	if len(violations) != 0 {
		alert := &v1.Alert{
			Id:         "ID", // UUID
			Violations: violations,
		}
		return alert, nil
	}
	return nil, nil
}

func (i *ImageProcessor) enrichImage(image *v1.Image) error {

	for _, registry := range i.registries {
		metadata, err := registry.Metadata(image)
		if err != nil {
			log.Error(err)
			continue
		}
		image.Metadata = metadata
		break
	}

	for _, scanner := range i.scanners {
		scan, err := scanner.GetScan(image.Sha)
		if err != nil {
			log.Error(err)
			continue
		}
		image.Scan = scan
		break
	}

	// Store image in the database
	i.database.AddImage(image)
	return nil
}
