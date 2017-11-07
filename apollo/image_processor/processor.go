package imageprocessor

import (
	"regexp"
	"sync"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
)

var (
	log = logging.New("imageprocessor/processor")
)

// ImageProcessor enriches and processes images to determine rule violations
type ImageProcessor struct {
	database db.Storage

	ruleMutex  sync.Mutex
	regexRules map[string]*regexImageRule
}

type lineRuleFieldRegex struct {
	Instruction string
	Value       *regexp.Regexp
}

type imageNameRuleRegex struct {
	Registry  *regexp.Regexp
	Namespace *regexp.Regexp
	Repo      *regexp.Regexp
	Tag       *regexp.Regexp
}

type regexImageRule struct {
	Name     string
	Severity v1.Severity

	ImageNameRule *imageNameRuleRegex

	ImageAgeDays int64
	LineRule     *lineRuleFieldRegex

	CVSS        *v1.NumericalRule
	CVE         *regexp.Regexp
	Component   *regexp.Regexp
	ScanAgeDays int64
}

func compileImageNameRuleRegex(rule *v1.ImageNameRule) (*imageNameRuleRegex, error) {
	if rule == nil {
		return nil, nil
	}
	registry, err := compileStringRegex(rule.Registry)
	if err != nil {
		return nil, err
	}
	namespace, err := compileStringRegex(rule.Namespace)
	if err != nil {
		return nil, err
	}
	repo, err := compileStringRegex(rule.Repo)
	if err != nil {
		return nil, err
	}
	tag, err := compileStringRegex(rule.Tag)
	if err != nil {
		return nil, err
	}
	return &imageNameRuleRegex{
		Registry:  registry,
		Namespace: namespace,
		Repo:      repo,
		Tag:       tag,
	}, nil
}

func compileStringRegex(rule string) (*regexp.Regexp, error) {
	if rule == "" {
		return nil, nil
	}
	return regexp.Compile(rule)
}

func compileLineRuleFieldRegex(line *v1.DockerfileLineRuleField) (*lineRuleFieldRegex, error) {
	if line == nil {
		return nil, nil
	}
	value, err := regexp.Compile(line.Value)
	if err != nil {
		return nil, err
	}
	return &lineRuleFieldRegex{
		Instruction: line.Instruction,
		Value:       value,
	}, nil
}

func (i *ImageProcessor) addRegexImageRule(rule *v1.ImageRule) error {
	imageNameRegex, err := compileImageNameRuleRegex(rule.ImageName)
	if err != nil {
		return err
	}
	lineRule, err := compileLineRuleFieldRegex(rule.LineRule)
	if err != nil {
		return err
	}
	component, err := compileStringRegex(rule.Component)
	if err != nil {
		return err
	}
	cve, err := compileStringRegex(rule.Cve)
	if err != nil {
		return err
	}
	i.regexRules[rule.Name] = &regexImageRule{
		Name:     rule.Name,
		Severity: rule.Severity,

		ImageNameRule: imageNameRegex,

		ImageAgeDays: rule.ImageAgeDays,
		LineRule:     lineRule,

		CVSS:        rule.Cvss,
		CVE:         cve,
		Component:   component,
		ScanAgeDays: rule.ScanAgeDays,
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

// Process takes in a new image and determines if an alert should be fired
func (i *ImageProcessor) Process(image *v1.Image) ([]*v1.Alert, error) {
	// TODO(cgorman) Better notification system or scheme around notifying if there is an error
	if err := i.enrichImage(image); err != nil {
		return nil, err
	}
	return i.checkImage(image)
}

func (i *ImageProcessor) checkImage(image *v1.Image) ([]*v1.Alert, error) {
	i.ruleMutex.Lock()
	defer i.ruleMutex.Unlock()

	var alerts []*v1.Alert
	for _, rule := range i.regexRules {
		if alert := rule.matchRuleToImage(image); alert != nil {
			alerts = append(alerts, alert)
		}
	}
	return alerts, nil
}

func (i *ImageProcessor) enrichImage(image *v1.Image) error {
	i.ruleMutex.Lock()
	defer i.ruleMutex.Unlock()
	for _, registry := range i.database.GetRegistries() {
		metadata, err := registry.Metadata(image)
		if err != nil {
			log.Error(err) // This will be removed, but useful for debugging at this point
			continue
		}
		image.Metadata = metadata
		break
	}

	for _, scanner := range i.database.GetScanners() {
		scan, err := scanner.GetLastScan(image)
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
