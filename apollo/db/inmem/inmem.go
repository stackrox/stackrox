package inmem

import (
	"sort"
	"sync"

	registryTypes "bitbucket.org/stack-rox/apollo/apollo/registries/types"
	scannerTypes "bitbucket.org/stack-rox/apollo/apollo/scanners/types"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

// InMemoryStore is an in memory representation of the database
type InMemoryStore struct {
	images     map[string]*v1.Image
	imageMutex sync.Mutex

	imageRules      map[string]*v1.ImageRule
	imageRulesMutex sync.Mutex

	alerts     map[string]*v1.Alert
	alertMutex sync.Mutex

	registries    map[string]registryTypes.ImageRegistry
	registryMutex sync.Mutex

	scanners  map[string]scannerTypes.ImageScanner
	scanMutex sync.Mutex
}

// New creates a new InMemoryStore
func New() *InMemoryStore {
	return &InMemoryStore{
		images:     make(map[string]*v1.Image),
		imageRules: make(map[string]*v1.ImageRule),
		alerts:     make(map[string]*v1.Alert),
		registries: make(map[string]registryTypes.ImageRegistry),
		scanners:   make(map[string]scannerTypes.ImageScanner),
	}
}

// AddImage adds an image to the database
func (i *InMemoryStore) AddImage(image *v1.Image) {
	i.imageMutex.Lock()
	defer i.imageMutex.Unlock()
	i.images[image.Sha] = image
}

// RemoveImage removes a specific image specified by it's SHA
func (i *InMemoryStore) RemoveImage(sha string) {
	i.imageMutex.Lock()
	defer i.imageMutex.Unlock()
	delete(i.images, sha)
}

// GetImages returns all images
func (i *InMemoryStore) GetImages() []*v1.Image {
	i.imageMutex.Lock()
	defer i.imageMutex.Unlock()
	images := make([]*v1.Image, 0, len(i.images))
	for _, image := range i.images {
		images = append(images, image)
	}
	sort.SliceStable(images, func(i, j int) bool { return images[i].Remote < images[j].Remote })
	return images
}

// AddImageRule adds the image rule to the database
func (i *InMemoryStore) AddImageRule(rule *v1.ImageRule) {
	i.imageRulesMutex.Lock()
	defer i.imageRulesMutex.Unlock()
	i.imageRules[rule.Name] = rule
}

// RemoveImageRule removes the image rule
func (i *InMemoryStore) RemoveImageRule(name string) {
	i.imageRulesMutex.Lock()
	defer i.imageRulesMutex.Unlock()
	delete(i.imageRules, name)
}

// UpdateImageRule replaces the image rule stored with the new one
func (i *InMemoryStore) UpdateImageRule(rule *v1.ImageRule) {
	i.imageRulesMutex.Lock()
	defer i.imageRulesMutex.Unlock()
	i.imageRules[rule.Name] = rule
}

// GetImageRules returns all image rules
func (i *InMemoryStore) GetImageRules() []*v1.ImageRule {
	i.imageRulesMutex.Lock()
	defer i.imageRulesMutex.Unlock()
	rules := make([]*v1.ImageRule, 0, len(i.imageRules))
	for _, v := range i.imageRules {
		rules = append(rules, v)
	}
	sort.SliceStable(rules, func(i, j int) bool { return rules[i].Name < rules[j].Name })
	return rules
}

// GetImageRule retrieves an image rule by it's name
func (i *InMemoryStore) GetImageRule(name string) *v1.ImageRule {
	i.imageRulesMutex.Lock()
	defer i.imageRulesMutex.Unlock()
	return i.imageRules[name]
}

// GetAlert retrieves an alert by it's id
func (i *InMemoryStore) GetAlert(id string) *v1.Alert {
	i.alertMutex.Lock()
	defer i.alertMutex.Unlock()
	return i.alerts[id]
}

// GetAlerts retrieves all alerts
func (i *InMemoryStore) GetAlerts() []*v1.Alert {
	i.alertMutex.Lock()
	defer i.alertMutex.Unlock()
	alerts := make([]*v1.Alert, 0, len(i.alerts))
	for _, alert := range i.alerts {
		alerts = append(alerts, alert)
	}
	sort.SliceStable(alerts, func(i, j int) bool { return alerts[i].Id < alerts[j].Id })
	return alerts
}

// AddAlert stores a new alert
func (i *InMemoryStore) AddAlert(alert *v1.Alert) {
	i.alertMutex.Lock()
	defer i.alertMutex.Unlock()
	i.alerts[alert.Id] = alert
}

// RemoveAlert removes an alert
func (i *InMemoryStore) RemoveAlert(id string) {
	i.alertMutex.Lock()
	defer i.alertMutex.Unlock()
	delete(i.alerts, id)
}

// AddRegistry adds a registry
func (i *InMemoryStore) AddRegistry(name string, registry registryTypes.ImageRegistry) {
	i.registryMutex.Lock()
	defer i.registryMutex.Unlock()
	i.registries[name] = registry
}

// RemoveRegistry removes a registry
func (i *InMemoryStore) RemoveRegistry(name string) {
	i.registryMutex.Lock()
	defer i.registryMutex.Unlock()
	delete(i.registries, name)
}

// GetRegistries retrieves all registries from the DB
func (i *InMemoryStore) GetRegistries() map[string]registryTypes.ImageRegistry {
	i.registryMutex.Lock()
	defer i.registryMutex.Unlock()
	return i.registries
}

// AddScanner adds a scanner
func (i *InMemoryStore) AddScanner(name string, scanner scannerTypes.ImageScanner) {
	i.scanMutex.Lock()
	defer i.scanMutex.Unlock()
	i.scanners[name] = scanner
}

// RemoveScanner removes a scanner
func (i *InMemoryStore) RemoveScanner(name string) {
	i.scanMutex.Lock()
	defer i.scanMutex.Unlock()
	delete(i.scanners, name)
}

// GetScanners retrieves all scanners from the db
func (i *InMemoryStore) GetScanners() map[string]scannerTypes.ImageScanner {
	i.scanMutex.Lock()
	defer i.scanMutex.Unlock()
	return i.scanners
}
