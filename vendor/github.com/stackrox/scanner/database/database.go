// Copyright 2017 clair authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package database defines the Clair's models and a common interface for
// database implementations.
package database

import (
	"errors"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/scanner/pkg/component"
)

var (
	// ErrBackendException is an error that occurs when the database backend does
	// not work properly (ie. unreachable).
	ErrBackendException = errors.New("database: an error occured when querying the backend")

	// ErrInconsistent is an error that occurs when a database consistency check
	// fails (i.e. when an entity which is supposed to be unique is detected
	// twice)
	ErrInconsistent = errors.New("database: inconsistent database")
)

// RegistrableComponentConfig is a configuration block that can be used to
// determine which registrable component should be initialized and pass custom
// configuration to it.
// Any updates to this should be tested in cmd/clair/config_test.go.
type RegistrableComponentConfig struct {
	Type    string
	Options map[string]interface{}
}

var drivers = make(map[string]Driver)

// Driver is a function that opens a Datastore specified by its database driver type and specific
// configuration.
type Driver func(cfg RegistrableComponentConfig, passwordRequired bool) (Datastore, error)

// Register makes a Constructor available by the provided name.
//
// If this function is called twice with the same name or if the Constructor is
// nil, it panics.
func Register(name string, driver Driver) {
	if driver == nil {
		panic("database: could not register nil Driver")
	}
	if _, dup := drivers[name]; dup {
		panic("database: could not register duplicate Driver: " + name)
	}
	drivers[name] = driver
}

// OpenWithRetries opens the database with the given number of retries.
func OpenWithRetries(cfg RegistrableComponentConfig, passwordRequired bool, maxTries int, sleepBetweenTries time.Duration) (Datastore, error) {
	driver, ok := drivers[cfg.Type]
	if !ok {
		return nil, fmt.Errorf("database: unknown Driver %q (forgotten configuration or import?)", cfg.Type)
	}
	var db Datastore
	err := retry.WithRetry(func() error {
		var err error
		db, err = driver(cfg, passwordRequired)
		return err
	}, retry.Tries(maxTries), retry.OnFailedAttempts(func(err error) {
		log.WithError(err).Error("Failed to open database.")
	}), retry.BetweenAttempts(func(previousAttemptNumber int) {
		log.WithField("Attempt", previousAttemptNumber+1).Warn("Retrying connection to DB")
		time.Sleep(sleepBetweenTries)
	}),
	)
	if err != nil {
		return nil, err
	}
	return db, nil
}

// Open opens a Datastore specified by a configuration.
func Open(cfg RegistrableComponentConfig, passwordRequired bool) (Datastore, error) {
	return OpenWithRetries(cfg, passwordRequired, 1, 0)
}

// Datastore represents the required operations on a persistent data store for
// a Clair deployment.
type Datastore interface {
	// InsertLayer stores a Layer in the database.
	//
	// A Layer is uniquely identified by its Name.
	// The Name and EngineVersion fields are mandatory.
	// If a Parent is specified, it is expected that it has been retrieved using
	// FindLayer.
	// If a Layer that already exists is inserted and the EngineVersion of the
	// given Layer is higher than the stored one, the stored Layer should be
	// updated.
	// The function has to be idempotent, inserting a layer that already exists
	// shouldn't return an error.
	InsertLayer(Layer, string, *DatastoreOptions) error

	// FindLayer retrieves a Layer from the database.
	FindLayer(name string, lineage string, opts *DatastoreOptions) (Layer, error)

	// InsertRHELv2Layer inserts a RHELv2 layer into the database.
	InsertRHELv2Layer(*RHELv2Layer) error

	// InsertVulnerabilities stores the given Vulnerabilities in the database,
	// updating them if necessary.
	//
	// A vulnerability is uniquely identified by its Namespace and its Name.
	// The FixedIn field may only contain a partial list of Features that are
	// affected by the Vulnerability, along with the version in which the
	// vulnerability is fixed. It is the responsibility of the implementation to
	// update the list properly.
	// A version equals to versionfmt.MinVersion means that the given Feature is
	// not being affected by the Vulnerability at all and thus, should be removed
	// from the list.
	// It is important that Features should be unique in the FixedIn list. For
	// example, it doesn't make sense to have two `openssl` Feature listed as a
	// Vulnerability can only be fixed in one Version. This is true because
	// Vulnerabilities and Features are namespaced (i.e. specific to one
	// operating system).
	// Each vulnerability insertion or update has to create a Notification that
	// will contain the old and the updated Vulnerability, unless
	// createNotification equals to true.
	InsertVulnerabilities(vulnerabilities []Vulnerability) error

	// InsertRHELv2Vulnerabilities stores the given RHELv2 vulnerabilities into
	// the database.
	InsertRHELv2Vulnerabilities(vulnerabilities []*RHELv2Vulnerability) error

	// GetRHELv2Layers retrieves the corresponding layers for the image
	// represented by the given layer.
	// The returned slice is sorted in order from base layer to top.
	GetRHELv2Layers(layer string) ([]*RHELv2Layer, error)

	// GetRHELv2Vulnerabilities retrieves RHELv2 vulnerabilities based on the given records.
	// The returned value maps package ID to the related vulnerabilities.
	GetRHELv2Vulnerabilities(records []*RHELv2Record) (map[int][]*RHELv2Vulnerability, error)

	// InsertKeyValue stores or updates a simple key/value pair in the database.
	InsertKeyValue(key, value string) error

	// GetKeyValue retrieves a value from the database from the given key.
	//
	// It returns an empty string if there is no such key.
	GetKeyValue(key string) (string, error)

	// Ping returns the health status of the database.
	Ping() bool

	// Close closes the database and frees any allocated resource.
	Close()

	// Lock creates or renew a Lock in the database with the given name, owner
	// and duration.
	//
	// After the specified duration, the Lock expires by itself if it hasn't been
	// unlocked, and thus, let other users create a Lock with the same name.
	// However, the owner can renew its Lock by setting renew to true.
	// Lock should not block, it should instead returns whether the Lock has been
	// successfully acquired/renewed. If it's the case, the expiration time of
	// that Lock is returned as well.
	Lock(name string, owner string, duration time.Duration, renew bool) (bool, time.Time)

	// Unlock releases an existing Lock.
	Unlock(name, owner string)

	// FindLock returns the owner of a Lock specified by the name, and its
	// expiration time if it exists.
	FindLock(name string) (string, time.Time, error)

	// Image
	GetLayerBySHA(sha string, opts *DatastoreOptions) (string, string, bool, error)
	GetLayerByName(name string, opts *DatastoreOptions) (string, string, bool, error)
	AddImage(layer, lineage, digest, name string, opts *DatastoreOptions) error
	InsertLayerComponents(l, lineage string, c []*component.Component, r []string, opts *DatastoreOptions) error

	GetLayerLanguageComponents(layer, lineage string, opts *DatastoreOptions) ([]*component.LayerToComponents, error)

	GetVulnerabilitiesForFeatureVersion(featureVersion FeatureVersion) ([]Vulnerability, error)
	LoadVulnerabilities(featureVersions []FeatureVersion) error

	FeatureExists(namespace, feature string) (bool, error)
}
