package imageprocessor

import (
	"fmt"
	"regexp"
	"sync"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/apollo/registries"
	registryTypes "bitbucket.org/stack-rox/apollo/apollo/registries/types"
	"bitbucket.org/stack-rox/apollo/apollo/scanners"
	scannerTypes "bitbucket.org/stack-rox/apollo/apollo/scanners/types"
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

	registryMutex sync.Mutex
	registries    map[string]registryTypes.ImageRegistry

	scannerMutex sync.Mutex
	scanners     map[string]scannerTypes.ImageScanner
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
	if _, ok := registryTypes.DockerfileInstructionSet[line.Instruction]; !ok {
		return nil, fmt.Errorf("%v is not a valid dockerfile instruction", line.Instruction)
	}
	value, err := compileStringRegex(line.Value)
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, fmt.Errorf("value must be defined for a dockerfile instruction")
	}
	return &lineRuleFieldRegex{
		Instruction: line.Instruction,
		Value:       value,
	}, nil
}

// UpdateRegistry updates image processors map of active registries
func (i *ImageProcessor) UpdateRegistry(registry registryTypes.ImageRegistry) {
	i.registryMutex.Lock()
	defer i.registryMutex.Unlock()
	i.registries[registry.ProtoRegistry().Name] = registry
}

// UpdateScanner updates image processors map of active scanners
func (i *ImageProcessor) UpdateScanner(scanner scannerTypes.ImageScanner) {
	i.scannerMutex.Lock()
	defer i.scannerMutex.Unlock()
	i.scanners[scanner.ProtoScanner().Name] = scanner
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

func (i *ImageProcessor) initializeRegistries() error {
	registryMap := make(map[string]registryTypes.ImageRegistry)
	protoRegistries, err := i.database.GetRegistries(&v1.GetRegistriesRequest{})
	if err != nil {
		return err
	}
	for _, protoRegistry := range protoRegistries {
		registry, err := registries.CreateRegistry(protoRegistry)
		if err != nil {
			return fmt.Errorf("error generating a registry from persisted registry data: %+v", err)
		}
		registryMap[protoRegistry.Name] = registry
	}
	i.registries = registryMap
	return nil
}

func (i *ImageProcessor) initializeScanners() error {
	scannerMap := make(map[string]scannerTypes.ImageScanner)
	protoScanners, err := i.database.GetScanners(&v1.GetScannersRequest{})
	if err != nil {
		return err
	}
	for _, protoScanner := range protoScanners {
		scanner, err := scanners.CreateScanner(protoScanner)
		if err != nil {
			return fmt.Errorf("error generating a registry from persisted registry data: %+v", err)
		}
		scannerMap[protoScanner.Name] = scanner
	}
	i.scanners = scannerMap
	return nil
}

func (i *ImageProcessor) initializeImagePolicies() error {
	imagePolicies, err := i.database.GetImagePolicies(&v1.GetImagePoliciesRequest{})
	if err != nil {
		return err
	}
	for _, imagePolicy := range imagePolicies {
		if err := i.addRegexImagePolicy(imagePolicy); err != nil {
			return err
		}
	}
	return nil
}

// New creates a new image processor and initializes the registries and scanners from the DB if they exist
func New(database db.Storage) (*ImageProcessor, error) {
	processor := &ImageProcessor{
		database:      database,
		regexPolicies: make(map[string]*regexImagePolicy),
	}
	if err := processor.initializeImagePolicies(); err != nil {
		return nil, err
	}
	if err := processor.initializeRegistries(); err != nil {
		return nil, err
	}
	if err := processor.initializeScanners(); err != nil {
		return nil, err
	}
	return processor, nil
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
		if policy.Original.GetDisabled() {
			continue
		}

		if alert := policy.matchPolicyToImage(deployment.GetImage()); alert != nil {
			alert.Deployment = deployment
			alerts = append(alerts, alert)
		}
	}
	return alerts, nil
}

func (i *ImageProcessor) enrichImage(image *v1.Image) error {
	i.registryMutex.Lock()
	for _, registry := range i.registries {
		metadata, err := registry.Metadata(image)
		if err != nil {
			log.Error(err) // This will be removed, but useful for debugging at this point
			continue
		}
		image.Metadata = metadata
		break
	}
	i.registryMutex.Unlock()

	i.scannerMutex.Lock()
	for _, scanner := range i.scanners {
		scan, err := scanner.GetLastScan(image)
		if err != nil {
			log.Error(err)
			continue
		}
		image.Scan = scan
		break
	}
	i.scannerMutex.Unlock()

	// Store image in the database
	return i.database.AddImage(image)
}
