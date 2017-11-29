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

// ImageProcessor enriches and processes images to determine image policy violations.
type ImageProcessor struct {
	database interface {
		db.ImagePolicyStorage
		db.ImageStorage
		db.RegistryStorage
		db.ScannerStorage
	}

	policyMutex   sync.Mutex
	regexPolicies map[string]*regexImagePolicy
}

type lineRuleFieldRegex struct {
	Instruction string
	Value       *regexp.Regexp
}

type imageNamePolicyRegex struct {
	Registry  *regexp.Regexp
	Namespace *regexp.Regexp
	Repo      *regexp.Regexp
	Tag       *regexp.Regexp
}

type regexImagePolicy struct {
	Original *v1.ImagePolicy

	ImageNamePolicy *imageNamePolicyRegex

	ImageAgeDays int64
	LineRule     *lineRuleFieldRegex

	CVSS        *v1.NumericalPolicy
	CVE         *regexp.Regexp
	Component   *regexp.Regexp
	ScanAgeDays int64
}

func compileImageNamePolicyRegex(policy *v1.ImageNamePolicy) (*imageNamePolicyRegex, error) {
	if policy == nil {
		return nil, nil
	}
	registry, err := compileStringRegex(policy.GetRegistry())
	if err != nil {
		return nil, err
	}
	namespace, err := compileStringRegex(policy.GetNamespace())
	if err != nil {
		return nil, err
	}
	repo, err := compileStringRegex(policy.GetRepo())
	if err != nil {
		return nil, err
	}
	tag, err := compileStringRegex(policy.GetTag())
	if err != nil {
		return nil, err
	}
	return &imageNamePolicyRegex{
		Registry:  registry,
		Namespace: namespace,
		Repo:      repo,
		Tag:       tag,
	}, nil
}

func compileStringRegex(policy string) (*regexp.Regexp, error) {
	if policy == "" {
		return nil, nil
	}
	return regexp.Compile(policy)
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

func (i *ImageProcessor) addRegexImagePolicy(policy *v1.ImagePolicy) error {
	imageNameRegex, err := compileImageNamePolicyRegex(policy.GetImageName())
	if err != nil {
		return err
	}
	lineRule, err := compileLineRuleFieldRegex(policy.GetLineRule())
	if err != nil {
		return err
	}
	component, err := compileStringRegex(policy.GetComponent())
	if err != nil {
		return err
	}
	cve, err := compileStringRegex(policy.GetCve())
	if err != nil {
		return err
	}
	i.regexPolicies[policy.GetName()] = &regexImagePolicy{
		Original: policy,

		ImageNamePolicy: imageNameRegex,

		ImageAgeDays: policy.GetImageAgeDays(),
		LineRule:     lineRule,

		CVSS:        policy.GetCvss(),
		CVE:         cve,
		Component:   component,
		ScanAgeDays: policy.GetScanAgeDays(),
	}
	return nil
}

// New creates a new image processor
func New(database db.Storage) (*ImageProcessor, error) {
	return &ImageProcessor{
		database: database,

		regexPolicies: make(map[string]*regexImagePolicy),
	}, nil
}

// UpdatePolicy updates the current policy in a threadsafe manner.
func (i *ImageProcessor) UpdatePolicy(policy *v1.ImagePolicy) error {
	i.policyMutex.Lock()
	defer i.policyMutex.Unlock()
	return i.addRegexImagePolicy(policy)
}

// RemovePolicy removes the policy specified by name in a threadsafe manner.
func (i *ImageProcessor) RemovePolicy(name string) {
	i.policyMutex.Lock()
	defer i.policyMutex.Unlock()
	delete(i.regexPolicies, name)
}

// Process takes in a new image and determines if an alert should be fired
func (i *ImageProcessor) Process(deployment *v1.Deployment) ([]*v1.Alert, error) {
	// TODO(cgorman) Better notification system or scheme around notifying if there is an error
	if err := i.enrichImage(deployment.Image); err != nil {
		return nil, err
	}
	return i.checkImage(deployment)
}

func (i *ImageProcessor) checkImage(deployment *v1.Deployment) ([]*v1.Alert, error) {
	i.policyMutex.Lock()
	defer i.policyMutex.Unlock()

	var alerts []*v1.Alert
	for _, policy := range i.regexPolicies {
		if alert := policy.matchPolicyToImage(deployment.GetImage()); alert != nil {
			alert.Deployment = deployment
			alerts = append(alerts, alert)
		}
	}
	return alerts, nil
}

func (i *ImageProcessor) enrichImage(image *v1.Image) error {
	i.policyMutex.Lock()
	defer i.policyMutex.Unlock()
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
