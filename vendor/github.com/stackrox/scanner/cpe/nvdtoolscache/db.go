package nvdtoolscache

import (
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/facebookincubator/nvdtools/cvefeed"
	"github.com/facebookincubator/nvdtools/cvefeed/nvd"
	"github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/scanner/cpe/attributes/common"
	"github.com/stackrox/scanner/pkg/cpeutils"
	"github.com/stackrox/scanner/pkg/vulndump"
	"github.com/stackrox/scanner/pkg/vulnloader/nvdloader"
	"github.com/stackrox/scanner/pkg/wellknowndirnames"
	"go.etcd.io/bbolt"
)

var (
	// BoltPath is a temporary path for the boltDB and is expected to be backed by
	// an empty dir. Exported for localdev and tests to be able to set it.
	// TODO: Make this injectable instead.
	BoltPath = filepath.Join(wellknowndirnames.WriteableDir, "temp.db")

	cveToProductBucket = []byte("stackrox-cve-to-product")
)

func newWithDB(db *bbolt.DB) Cache {
	return &cacheImpl{DB: db, dir: vulndump.NVDDirName}
}

func initializeDB(db *bbolt.DB) error {
	return db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(cveToProductBucket)
		return err
	})
}

// New returns a new NVD vulnerability cache.
func New() (Cache, error) {
	opts := bbolt.Options{
		NoFreelistSync: true,
		FreelistType:   bbolt.FreelistMapType,
		NoSync:         true,
	}
	db, err := bbolt.Open(BoltPath, 0600, &opts)
	if err != nil {
		return nil, err
	}
	if err := initializeDB(db); err != nil {
		return nil, err
	}
	return newWithDB(db), nil
}

type cacheImpl struct {
	*bbolt.DB

	dir             string
	updateLock      sync.Mutex
	lastUpdatedTime time.Time
}

func (c *cacheImpl) Dir() string {
	return c.dir
}

func (c *cacheImpl) addProductToCVE(vuln cvefeed.Vuln, cve *schema.NVDCVEFeedJSON10DefCVEItem) error {
	bytes, err := nvdloader.MarshalNVDFeedCVEItem(cve)
	if err != nil {
		return err
	}
	// Track the products that are associated with this CVE.
	productAlreadyWritten := set.NewStringSet()
	// Track the products that are no longer associated with this CVE.
	productsToDelete := set.NewStringSet()
	return c.Update(func(tx *bbolt.Tx) error {
		// Get the CVE to product mapping.
		cveBucket := tx.Bucket(cveToProductBucket)
		productBytes := cveBucket.Get([]byte(cve.CVE.CVEDataMeta.ID))
		if productBytes != nil {
			products, err := nvdloader.UnmarshalStringSlice(productBytes)
			if err != nil {
				return err
			}
			productsToDelete.AddAll(products...)
		}

		// Update the associated product buckets with the CVE.
		for _, a := range vuln.Config() {
			if !productAlreadyWritten.Add(a.Product) {
				continue
			}
			productsToDelete.Remove(a.Product)

			product := []byte(a.Product)
			bucket, err := tx.CreateBucketIfNotExists(product)
			if err != nil {
				return err
			}
			if err := bucket.Put([]byte(cve.CVE.CVEDataMeta.ID), bytes); err != nil {
				return err
			}
		}

		// Update the CVE bucket with the latest products.
		productBytes, err = nvdloader.MarshalStringSlice(productAlreadyWritten.AsSlice())
		if err != nil {
			return err
		}
		if err := cveBucket.Put([]byte(cve.CVE.CVEDataMeta.ID), productBytes); err != nil {
			return err
		}

		for product := range productsToDelete {
			bucket := tx.Bucket([]byte(product))
			if bucket == nil {
				return errors.Errorf("Bucket %s does not exist", product)
			}
			if err := bucket.Delete([]byte(cve.CVE.CVEDataMeta.ID)); err != nil {
				return err
			}
		}
		return nil
	})
}

func (c *cacheImpl) GetVulnsForProducts(products []string) ([]cvefeed.Vuln, error) {
	vulnSet := set.NewStringSet()
	var vulns []cvefeed.Vuln
	err := c.View(func(tx *bbolt.Tx) error {
		for _, product := range products {
			bucket := tx.Bucket([]byte(product))
			if bucket == nil {
				continue
			}
			err := bucket.ForEach(func(k, v []byte) error {
				if !vulnSet.Add(string(k)) {
					return nil
				}
				vuln, err := nvdloader.UnmarshalNVDFeedCVEItem(v)
				if err != nil {
					return errors.Wrapf(err, "unmarshaling vuln %s", string(k))
				}
				vulns = append(vulns, nvd.ToVuln(vuln))
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	return vulns, err
}

func (c *cacheImpl) GetVulnsForComponent(vendor, product, version string) ([]*NVDCVEItemWithFixedIn, error) {
	var cveItems []*schema.NVDCVEFeedJSON10DefCVEItem
	err := c.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(escapeDash(escapePeriod(product))))
		if bucket == nil {
			return errors.Errorf("unable to fetch bucket for %s", product)
		}
		return bucket.ForEach(func(k, v []byte) error {
			vuln, err := nvdloader.UnmarshalNVDFeedCVEItem(v)
			if err != nil {
				return errors.Wrapf(err, "unmarshaling vuln %s", string(k))
			}
			cveItems = append(cveItems, vuln)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	// TODO: Consider using pre-existing functions.
	vendorSet := set.NewStringSet(vendor, escapeDash(vendor), escapePeriod(vendor), escapeDash(escapePeriod(vendor)))
	productSet := set.NewStringSet(product, escapeDash(product), escapePeriod(product), escapeDash(escapePeriod(product)))
	versionSet := set.NewStringSet(version, escapeDash(version), escapePeriod(version), escapeDash(escapePeriod(version)))
	attrs := common.GenerateAttributesFromSets(vendorSet, productSet, versionSet, "")

	var vulnsWithFixed []*NVDCVEItemWithFixedIn
	for _, cveItem := range cveItems {
		nvdVuln := nvd.ToVuln(cveItem)
		if matchesWithFixed := nvdVuln.MatchWithFixedIn(attrs, false); len(matchesWithFixed) > 0 {
			vulnsWithFixed = append(vulnsWithFixed, &NVDCVEItemWithFixedIn{
				NVDCVEFeedJSON10DefCVEItem: cveItem,
				FixedIn:                    cpeutils.GetMostSpecificCPE(matchesWithFixed).FixedIn,
			})
		}
	}

	return vulnsWithFixed, nil
}

func escapePeriod(str string) string {
	return strings.ReplaceAll(str, ".", `\.`)
}

func escapeDash(s string) string {
	return strings.ReplaceAll(s, "-", `\-`)
}

func (c *cacheImpl) GetLastUpdate() time.Time {
	c.updateLock.Lock()
	defer c.updateLock.Unlock()

	return c.lastUpdatedTime
}

func (c *cacheImpl) SetLastUpdate(t time.Time) {
	c.updateLock.Lock()
	defer c.updateLock.Unlock()

	c.lastUpdatedTime = t
}

func (c *cacheImpl) sync() error {
	return c.DB.Sync()
}
