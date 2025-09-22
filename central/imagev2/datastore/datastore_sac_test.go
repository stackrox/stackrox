//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	imageCVEEV2DataStore "github.com/stackrox/rox/central/cve/image/v2/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	graphDBTestUtils "github.com/stackrox/rox/central/graphdb/testutils"
	namespaceDataStore "github.com/stackrox/rox/central/namespace/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/sac/testutils"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestImageV2DataStoreSAC(t *testing.T) {
	if !features.FlattenImageData.Enabled() {
		t.Skip("FlattenImageData disabled.  Test is not appropriate.")
	}
	suite.Run(t, new(imageV2DatastoreSACSuite))
}

type imageV2DatastoreSACSuite struct {
	suite.Suite

	// Elements for postgres mode
	pgtestbase *pgtest.TestPostgres

	datastore DataStore

	imageVulnDatastore  imageCVEEV2DataStore.DataStore
	deploymentDatastore deploymentDataStore.DataStore
	namespaceDatastore  namespaceDataStore.DataStore

	optionsMap searchPkg.OptionsMap

	testContexts map[string]context.Context
	testImageIDs []string

	extraImage *storage.ImageV2
}

func (s *imageV2DatastoreSACSuite) SetupSuite() {
	var err error
	s.pgtestbase = pgtest.ForT(s.T())
	s.Require().NotNil(s.pgtestbase)
	s.datastore = GetTestPostgresDataStore(s.T(), s.pgtestbase.DB)
	s.imageVulnDatastore = imageCVEEV2DataStore.GetTestPostgresDataStore(s.T(), s.pgtestbase.DB)
	s.deploymentDatastore, err = deploymentDataStore.GetTestPostgresDataStore(s.T(), s.pgtestbase.DB)
	s.Require().NoError(err)
	s.namespaceDatastore, err = namespaceDataStore.GetTestPostgresDataStore(s.T(), s.pgtestbase.DB)
	s.Require().NoError(err)
	s.optionsMap = schema.ImagesV2Schema.OptionsMap

	s.testContexts = testutils.GetNamespaceScopedTestContexts(context.Background(), s.T(),
		resources.Image)

	s.extraImage = fixtures.GetImageV2()
}

func (s *imageV2DatastoreSACSuite) TearDownSuite() {
	s.pgtestbase.Close()
}

func (s *imageV2DatastoreSACSuite) SetupTest() {
	s.testImageIDs = make([]string, 0)
}

func (s *imageV2DatastoreSACSuite) TearDownTest() {
	for _, id := range s.testImageIDs {
		s.deleteImage(id)
	}
}

func (s *imageV2DatastoreSACSuite) deleteImage(id string) {
	s.Require().NoError(s.datastore.DeleteImages(s.testContexts[testutils.UnrestrictedReadWriteCtx], id))
}

func (s *imageV2DatastoreSACSuite) deleteDeployment(clusterid, id string) {
	s.Require().NoError(s.deploymentDatastore.RemoveDeployment(sac.WithAllAccess(context.Background()), clusterid, id))
}

func (s *imageV2DatastoreSACSuite) deleteNamespace(id string) {
	s.Require().NoError(s.namespaceDatastore.RemoveNamespace(sac.WithAllAccess(context.Background()), id))
}

func (s *imageV2DatastoreSACSuite) verifyRawImagesEqual(image1, image2 *storage.ImageV2) {
	s.Equal(image1.GetId(), image2.GetId())
	s.Equal(image1.GetScanStats().GetComponentCount(), image2.GetScanStats().GetComponentCount())
	s.Equal(image1.GetScanStats().GetCveCount(), image2.GetScanStats().GetCveCount())
}

func (s *imageV2DatastoreSACSuite) TestUpsertImage() {
	cases := testutils.GenericGlobalSACUpsertTestCases(s.T(), testutils.VerbUpsert)

	for name, testCase := range cases {
		s.Run(name, func() {
			ctx := s.testContexts[testCase.ScopeKey]
			image := fixtures.GetImageV2SherlockHolmes1()
			err := s.datastore.UpsertImage(ctx, image)
			defer s.deleteImage(image.GetId())
			if testCase.ExpectError {
				s.Error(err)
				s.ErrorIs(err, testCase.ExpectedError)
			} else {
				s.NoError(err)
				checkCtx := s.testContexts[testutils.UnrestrictedReadCtx]
				readImage, found, checkErr := s.datastore.GetImage(checkCtx, image.GetId())
				s.NoError(checkErr)
				s.True(found)
				s.Equal(*image.GetName(), *readImage.GetName())
			}
		})
	}
}

func (s *imageV2DatastoreSACSuite) TestUpdateVulnerabilityState() {
	cases := testutils.GenericGlobalSACUpsertTestCases(s.T(), "update vulnerability state")

	for name, testCase := range cases {
		s.Run(name, func() {
			ctx := s.testContexts[testCase.ScopeKey]
			writeCtx := s.testContexts[testutils.UnrestrictedReadWriteCtx]
			checkCtx := s.testContexts[testutils.UnrestrictedReadCtx]
			image := fixtures.GetImageV2SherlockHolmes1()
			cve1 := fixtures.GetEmbeddedImageCVE1234x0001()
			cve2 := fixtures.GetEmbeddedImageCVE4567x0002()
			cve3 := fixtures.GetEmbeddedImageCVE1234x0003()
			cve4 := fixtures.GetEmbeddedImageCVE3456x0004()
			cve5 := fixtures.GetEmbeddedImageCVE3456x0005()
			cve6 := fixtures.GetEmbeddedImageCVE2345x0006()
			foundCVEs := []*storage.EmbeddedVulnerability{cve1, cve2, cve3, cve4, cve5}
			missingCVEs := []*storage.EmbeddedVulnerability{cve6}
			insertErr := s.datastore.UpsertImage(writeCtx, image)
			defer s.deleteImage(image.GetId())
			s.Require().NoError(insertErr)
			for _, cve := range foundCVEs {
				query := searchPkg.NewQueryBuilder().AddExactMatches(searchPkg.CVE, cve.GetCve()).
					AddExactMatches(searchPkg.ImageID, image.GetId()).ProtoQuery()
				vulns, err := s.imageVulnDatastore.SearchRawImageCVEs(checkCtx, query)
				s.NoError(err)
				s.True(len(vulns) > 0)
				for _, vuln := range vulns {
					s.Equal(storage.VulnerabilityState_OBSERVED, vuln.GetState())
				}
			}
			for _, cve := range missingCVEs {
				query := searchPkg.NewQueryBuilder().AddExactMatches(searchPkg.CVE, cve.GetCve()).
					AddExactMatches(searchPkg.ImageID, image.GetId()).ProtoQuery()
				vulns, err := s.imageVulnDatastore.SearchRawImageCVEs(checkCtx, query)
				s.NoError(err)
				s.True(len(vulns) == 0)
			}
			targetCve := cve3.GetCve()
			newState := storage.VulnerabilityState_DEFERRED
			updateErr := s.datastore.UpdateVulnerabilityState(ctx, targetCve, []string{image.GetId()}, newState)
			if testCase.ExpectError {
				s.Error(updateErr)
				s.ErrorIs(updateErr, testCase.ExpectedError)
			} else {
				s.NoError(updateErr)
				for _, cve := range foundCVEs {
					expectedState := storage.VulnerabilityState_OBSERVED
					if cve.GetCve() == cve3.GetCve() {
						expectedState = storage.VulnerabilityState_DEFERRED
					}
					query := searchPkg.NewQueryBuilder().AddExactMatches(searchPkg.CVE, cve.GetCve()).
						AddExactMatches(searchPkg.ImageID, image.GetId()).ProtoQuery()
					vulns, err := s.imageVulnDatastore.SearchRawImageCVEs(checkCtx, query)
					s.NoError(err)
					s.True(len(vulns) > 0)
					for _, vuln := range vulns {
						s.Equal(expectedState, vuln.GetState())
					}
				}
				for _, cve := range missingCVEs {
					query := searchPkg.NewQueryBuilder().AddExactMatches(searchPkg.CVE, cve.GetCve()).
						AddExactMatches(searchPkg.ImageID, image.GetId()).ProtoQuery()
					vulns, err := s.imageVulnDatastore.SearchRawImageCVEs(checkCtx, query)
					s.NoError(err)
					s.True(len(vulns) == 0)
				}
			}
		})
	}
}

func (s *imageV2DatastoreSACSuite) TestDeleteImages() {
	cases := testutils.GenericGlobalSACDeleteTestCases(s.T())

	for name, testCase := range cases {
		s.Run(name, func() {
			ctx := s.testContexts[testCase.ScopeKey]
			writeCtx := s.testContexts[testutils.UnrestrictedReadWriteCtx]
			checkCtx := s.testContexts[testutils.UnrestrictedReadCtx]
			image := fixtures.GetImageV2SherlockHolmes1()
			defer s.deleteImage(image.GetId())
			upsertErr := s.datastore.UpsertImage(writeCtx, image)
			s.Require().NoError(upsertErr)
			_, found, check1Err := s.datastore.GetImage(checkCtx, image.GetId())
			s.Require().NoError(check1Err)
			s.Require().True(found)
			deleteErr := s.datastore.DeleteImages(ctx, image.GetId())
			if testCase.ExpectError {
				s.Error(deleteErr)
				s.ErrorIs(deleteErr, testCase.ExpectedError)
				_, postRemovalFound, check2Err := s.datastore.GetImage(checkCtx, image.GetId())
				s.NoError(check2Err)
				s.True(postRemovalFound)
			} else {
				s.NoError(deleteErr)
				_, postRemovalFound, check2Err := s.datastore.GetImage(checkCtx, image.GetId())
				s.NoError(check2Err)
				s.False(postRemovalFound)
			}
		})
	}
}

func (s *imageV2DatastoreSACSuite) setupReadTest() ([]*storage.ImageV2, func(), error) {
	var setupErr error
	namespacesToDelete := make([]string, 0, 1)
	deploymentsToDelete := make([]*storage.Deployment, 0, 1)
	imagesToDelete := make([]string, 0, 1)
	images := make([]*storage.ImageV2, 0, 1)
	cleanup := func() {
		for _, img := range imagesToDelete {
			s.deleteImage(img)
		}
		for _, deployment := range deploymentsToDelete {
			s.deleteDeployment(deployment.GetClusterId(), deployment.GetId())
		}
		for _, ns := range namespacesToDelete {
			s.deleteNamespace(ns)
		}
	}
	setupCtx := sac.WithAllAccess(context.Background())
	namespace := fixtures.GetNamespace(testconsts.Cluster2, testconsts.Cluster2, testconsts.NamespaceB)
	namespacesToDelete = append(namespacesToDelete, namespace.GetId())
	setupErr = s.namespaceDatastore.AddNamespace(setupCtx, namespace)
	if setupErr != nil {
		return nil, cleanup, setupErr
	}
	deployment := fixtures.GetDeploymentSherlockHolmes1(uuid.NewV4().String(), namespace)
	deploymentsToDelete = append(deploymentsToDelete, deployment)
	setupErr = s.deploymentDatastore.UpsertDeployment(setupCtx, deployment)
	if setupErr != nil {
		return nil, cleanup, setupErr
	}
	deployment2 := fixtures.GetDeploymentDoctorJekyll2(uuid.NewV4().String(), namespace)
	deploymentsToDelete = append(deploymentsToDelete, deployment2)
	setupErr = s.deploymentDatastore.UpsertDeployment(setupCtx, deployment2)
	if setupErr != nil {
		return nil, cleanup, setupErr
	}
	image := fixtures.GetImageV2SherlockHolmes1()
	imagesToDelete = append(imagesToDelete, image.GetId())
	images = append(images, image)
	setupErr = s.datastore.UpsertImage(setupCtx, image)
	if setupErr != nil {
		return nil, cleanup, setupErr
	}
	image2 := fixtures.GetImageV2DoctorJekyll2()
	imagesToDelete = append(imagesToDelete, image2.GetId())
	images = append(images, image2)
	setupErr = s.datastore.UpsertImage(setupCtx, image2)
	if setupErr != nil {
		return nil, cleanup, setupErr
	}
	return images, cleanup, nil
}

func (s *imageV2DatastoreSACSuite) TestExists() {
	images, cleanup, setupErr := s.setupReadTest()
	defer cleanup()
	s.Require().NoError(setupErr)
	s.Require().NotZero(len(images))
	image := images[0]

	s.runReadTest("TestExists", "", func(testCase testutils.SACCrudTestCase) {
		ctx := s.testContexts[testCase.ScopeKey]
		exists, err := s.datastore.Exists(ctx, image.GetId())
		s.NoError(err)
		if testCase.ExpectedFound {
			s.True(exists)
		} else {
			s.False(exists)
		}
	})
}

func (s *imageV2DatastoreSACSuite) TestGetImage() {
	images, cleanup, setupErr := s.setupReadTest()
	defer cleanup()
	s.Require().NoError(setupErr)
	s.Require().NotZero(len(images))
	image := images[0]

	s.runReadTest("TestGetImage", "", func(testCase testutils.SACCrudTestCase) {
		ctx := s.testContexts[testCase.ScopeKey]
		readImage, found, err := s.datastore.GetImage(ctx, image.GetId())
		s.Require().NoError(err)
		if testCase.ExpectedFound {
			s.True(found)
			s.verifyRawImagesEqual(image, readImage)
		} else {
			s.False(found)
			s.Nil(readImage)
		}
	})
}

func (s *imageV2DatastoreSACSuite) TestGetImageMetadata() {
	images, cleanup, setupErr := s.setupReadTest()
	defer cleanup()
	s.Require().NoError(setupErr)
	s.Require().NotZero(len(images))
	image := images[0]

	s.runReadTest("TestGetImageMetadata", "", func(testCase testutils.SACCrudTestCase) {
		ctx := s.testContexts[testCase.ScopeKey]
		readImageMeta, found, err := s.datastore.GetImageMetadata(ctx, image.GetId())
		s.Require().NoError(err)
		if testCase.ExpectedFound {
			s.True(found)
			s.Equal(image.GetId(), readImageMeta.GetId())
			s.Equal(image.GetScanStats().GetComponentCount(), readImageMeta.GetScanStats().GetComponentCount())
			s.Equal(image.GetScanStats().GetCveCount(), readImageMeta.GetScanStats().GetCveCount())
		} else {
			s.False(found)
			s.Nil(readImageMeta)
		}
	})

	s.Require().True(len(images) > 1)
	image2 := images[1]
	// Test GetManyImageMetadata in postgres mode (only supported mode).
	s.runReadTest("TestGetManyImageMetadata", "Many_", func(testCase testutils.SACCrudTestCase) {
		ctx := s.testContexts[testCase.ScopeKey]
		readMeta, err := s.datastore.GetManyImageMetadata(ctx, []string{image.GetId(), image2.GetId()})
		s.Require().NoError(err)
		if testCase.ExpectedFound {
			s.Require().Len(readMeta, 2)
			readImageMeta1 := readMeta[0]
			readImageMeta2 := readMeta[1]
			if readImageMeta1.GetId() == image.GetId() {
				s.Equal(image.GetId(), readImageMeta1.GetId())
				s.Equal(image.GetScanStats().GetComponentCount(), readImageMeta1.GetScanStats().GetComponentCount())
				s.Equal(image.GetScanStats().GetCveCount(), readImageMeta1.GetScanStats().GetCveCount())
				s.Equal(image2.GetId(), readImageMeta2.GetId())
				s.Equal(image2.GetScanStats().GetComponentCount(), readImageMeta2.GetScanStats().GetComponentCount())
				s.Equal(image2.GetScanStats().GetCveCount(), readImageMeta2.GetScanStats().GetCveCount())
			} else {
				s.Equal(image2.GetId(), readImageMeta1.GetId())
				s.Equal(image2.GetScanStats().GetComponentCount(), readImageMeta1.GetScanStats().GetComponentCount())
				s.Equal(image2.GetScanStats().GetCveCount(), readImageMeta1.GetScanStats().GetCveCount())
				s.Equal(image.GetId(), readImageMeta2.GetId())
				s.Equal(image.GetScanStats().GetComponentCount(), readImageMeta2.GetScanStats().GetComponentCount())
				s.Equal(image.GetScanStats().GetCveCount(), readImageMeta2.GetScanStats().GetCveCount())
			}
		} else {
			s.Len(readMeta, 0)
		}
	})
}

func (s *imageV2DatastoreSACSuite) TestGetImagesBatch() {
	images, cleanup, setupErr := s.setupReadTest()
	defer cleanup()
	s.Require().NoError(setupErr)
	s.Require().True(len(images) > 1)
	image1 := images[0]
	image2 := images[1]

	s.runReadTest("TestGetImagesBatch", "", func(testCase testutils.SACCrudTestCase) {
		ctx := s.testContexts[testCase.ScopeKey]
		readMeta, err := s.datastore.GetImagesBatch(ctx, []string{image1.GetId(), image2.GetId()})
		s.Require().NoError(err)
		if testCase.ExpectedFound {
			s.Require().Len(readMeta, 2)
			readImageMeta1 := readMeta[0]
			readImageMeta2 := readMeta[1]
			if readImageMeta1.GetId() == image1.GetId() {
				s.Equal(image1.GetId(), readImageMeta1.GetId())
				s.Equal(image1.GetScanStats().GetComponentCount(), readImageMeta1.GetScanStats().GetComponentCount())
				s.Equal(image1.GetScanStats().GetCveCount(), readImageMeta1.GetScanStats().GetCveCount())
				s.Equal(image2.GetId(), readImageMeta2.GetId())
				s.Equal(image2.GetScanStats().GetComponentCount(), readImageMeta2.GetScanStats().GetComponentCount())
				s.Equal(image2.GetScanStats().GetCveCount(), readImageMeta2.GetScanStats().GetCveCount())
			} else {
				s.Equal(image2.GetId(), readImageMeta1.GetId())
				s.Equal(image2.GetScanStats().GetComponentCount(), readImageMeta1.GetScanStats().GetComponentCount())
				s.Equal(image2.GetScanStats().GetCveCount(), readImageMeta1.GetScanStats().GetCveCount())
				s.Equal(image1.GetId(), readImageMeta2.GetId())
				s.Equal(image1.GetScanStats().GetComponentCount(), readImageMeta2.GetScanStats().GetComponentCount())
				s.Equal(image1.GetScanStats().GetCveCount(), readImageMeta2.GetScanStats().GetCveCount())
			}
		} else {
			s.Len(readMeta, 0)
		}
	})
}

func (s *imageV2DatastoreSACSuite) TestWalkByQuery() {
	images, cleanup, setupErr := s.setupReadTest()
	defer cleanup()
	s.Require().NoError(setupErr)
	s.Require().True(len(images) > 1)
	image1 := images[0]
	image2 := images[1]

	s.runReadTest("TestWalkByQuery", "", func(testCase testutils.SACCrudTestCase) {
		ctx := s.testContexts[testCase.ScopeKey]
		var foundAtLeastOne bool
		err := s.datastore.WalkByQuery(ctx, nil, func(image *storage.ImageV2) error {
			foundAtLeastOne = true
			matchedImage := image1
			if image.GetId() == image2.GetId() {
				matchedImage = image2
			}
			s.Equal(matchedImage.GetId(), image.GetId())
			s.Equal(matchedImage.GetScanStats().GetComponentCount(), image.GetScanStats().GetComponentCount())
			s.Equal(matchedImage.GetScanStats().GetCveCount(), image.GetScanStats().GetCveCount())
			return nil
		})
		s.Require().NoError(err)
		s.Equal(testCase.ExpectedFound, foundAtLeastOne)
	})
}

func (s *imageV2DatastoreSACSuite) runReadTest(testName string, prefix string, testFunc func(c testutils.SACCrudTestCase)) {
	imageGraphBefore := graphDBTestUtils.GetImageGraph(
		sac.WithAllAccess(context.Background()),
		s.T(),
		s.pgtestbase.DB,
	)

	cases := testutils.GenericNamespaceSACGetTestCases(s.T())

	failed := false
	for name, testCase := range cases {
		caseSucceeded := s.Run(prefix+name, func() {
			// When triggered in parallel, most tests fail.
			// TearDownTest is executed before the sub-tests.
			// See https://github.com/stretchr/testify/issues/934
			// s.T().Parallel()
			testFunc(testCase)
		})
		if !caseSucceeded {
			failed = true
		}
	}
	if failed {
		log.Infof("%s failed, dumping DB content.", testName)
		imageGraphBefore.Log()
	}

}

func (s *imageV2DatastoreSACSuite) getSearchTestCases() map[string]map[string]bool {
	// The map structure is the mapping ScopeKey -> ImageID -> Visible
	cases := map[string]map[string]bool{
		testutils.UnrestrictedReadCtx: {
			s.extraImage.GetId():                         true,
			fixtures.GetImageV2SherlockHolmes1().GetId(): true,
			fixtures.GetImageV2DoctorJekyll2().GetId():   true,
		},
		testutils.UnrestrictedReadWriteCtx: {
			s.extraImage.GetId():                         true,
			fixtures.GetImageV2SherlockHolmes1().GetId(): true,
			fixtures.GetImageV2DoctorJekyll2().GetId():   true,
		},
		testutils.Cluster1ReadWriteCtx: {
			s.extraImage.GetId():                         false,
			fixtures.GetImageV2SherlockHolmes1().GetId(): true,
			fixtures.GetImageV2DoctorJekyll2().GetId():   false,
		},
		testutils.Cluster1NamespaceAReadWriteCtx: {
			s.extraImage.GetId():                         false,
			fixtures.GetImageV2SherlockHolmes1().GetId(): true,
			fixtures.GetImageV2DoctorJekyll2().GetId():   false,
		},
		testutils.Cluster1NamespaceBReadWriteCtx: {
			s.extraImage.GetId():                         false,
			fixtures.GetImageV2SherlockHolmes1().GetId(): false,
			fixtures.GetImageV2DoctorJekyll2().GetId():   false,
		},
		testutils.Cluster1NamespacesABReadWriteCtx: {
			s.extraImage.GetId():                         false,
			fixtures.GetImageV2SherlockHolmes1().GetId(): true,
			fixtures.GetImageV2DoctorJekyll2().GetId():   false,
		},
		testutils.Cluster1NamespacesBCReadWriteCtx: {
			s.extraImage.GetId():                         false,
			fixtures.GetImageV2SherlockHolmes1().GetId(): false,
			fixtures.GetImageV2DoctorJekyll2().GetId():   false,
		},
		testutils.Cluster2ReadWriteCtx: {
			s.extraImage.GetId():                         false,
			fixtures.GetImageV2SherlockHolmes1().GetId(): true,
			fixtures.GetImageV2DoctorJekyll2().GetId():   true,
		},
		testutils.Cluster2NamespaceAReadWriteCtx: {
			s.extraImage.GetId():                         false,
			fixtures.GetImageV2SherlockHolmes1().GetId(): false,
			fixtures.GetImageV2DoctorJekyll2().GetId():   false,
		},
		testutils.Cluster2NamespaceBReadWriteCtx: {
			s.extraImage.GetId():                         false,
			fixtures.GetImageV2SherlockHolmes1().GetId(): true,
			fixtures.GetImageV2DoctorJekyll2().GetId():   true,
		},
		testutils.Cluster2NamespacesACReadWriteCtx: {
			s.extraImage.GetId():                         false,
			fixtures.GetImageV2SherlockHolmes1().GetId(): false,
			fixtures.GetImageV2DoctorJekyll2().GetId():   false,
		},
		testutils.Cluster2NamespacesBCReadWriteCtx: {
			s.extraImage.GetId():                         false,
			fixtures.GetImageV2SherlockHolmes1().GetId(): true,
			fixtures.GetImageV2DoctorJekyll2().GetId():   true,
		},
		testutils.Cluster3ReadWriteCtx: {
			s.extraImage.GetId():                         false,
			fixtures.GetImageV2SherlockHolmes1().GetId(): false,
			fixtures.GetImageV2DoctorJekyll2().GetId():   false,
		},
		testutils.Cluster3NamespaceAReadWriteCtx: {
			s.extraImage.GetId():                         false,
			fixtures.GetImageV2SherlockHolmes1().GetId(): false,
			fixtures.GetImageV2DoctorJekyll2().GetId():   false,
		},
		testutils.Cluster3NamespaceBReadWriteCtx: {
			s.extraImage.GetId():                         false,
			fixtures.GetImageV2SherlockHolmes1().GetId(): false,
			fixtures.GetImageV2DoctorJekyll2().GetId():   false,
		},
		testutils.MixedClusterAndNamespaceReadCtx: {
			s.extraImage.GetId():                         false,
			fixtures.GetImageV2SherlockHolmes1().GetId(): true,
			fixtures.GetImageV2DoctorJekyll2().GetId():   true,
		},
	}
	return cases
}

func (s *imageV2DatastoreSACSuite) setupSearchTest() (func(), error) {
	var setupErr error

	namespacesToDelete := make([]string, 0, 1)
	deploymentsToDelete := make([]*storage.Deployment, 0, 1)
	imagesToDelete := make([]string, 0, 1)

	cleanup := func() {
		for _, img := range imagesToDelete {
			s.deleteImage(img)
		}
		for _, deployment := range deploymentsToDelete {
			s.deleteDeployment(deployment.GetClusterId(), deployment.GetId())
		}
		for _, ns := range namespacesToDelete {
			s.deleteNamespace(ns)
		}
	}

	image1 := fixtures.GetImageV2SherlockHolmes1()
	imagesToDelete = append(imagesToDelete, image1.GetId())
	image2 := fixtures.GetImageV2DoctorJekyll2()
	imagesToDelete = append(imagesToDelete, image2.GetId())
	imagesToDelete = append(imagesToDelete, s.extraImage.GetId())

	namespace1A := fixtures.GetNamespace(testconsts.Cluster1, testconsts.Cluster1, testconsts.NamespaceA)
	namespacesToDelete = append(namespacesToDelete, namespace1A.GetId())
	namespace2B := fixtures.GetNamespace(testconsts.Cluster2, testconsts.Cluster2, testconsts.NamespaceB)
	namespacesToDelete = append(namespacesToDelete, namespace2B.GetId())

	deployment1A1 := fixtures.GetDeploymentSherlockHolmes1(uuid.NewV4().String(), namespace1A)
	deploymentsToDelete = append(deploymentsToDelete, deployment1A1)
	deployment2B1 := fixtures.GetDeploymentSherlockHolmes1(uuid.NewV4().String(), namespace2B)
	deploymentsToDelete = append(deploymentsToDelete, deployment2B1)
	deployment2B2 := fixtures.GetDeploymentDoctorJekyll2(uuid.NewV4().String(), namespace2B)
	deploymentsToDelete = append(deploymentsToDelete, deployment2B2)

	setupCtx := sac.WithAllAccess(context.Background())

	setupErr = s.namespaceDatastore.AddNamespace(setupCtx, namespace1A)
	if setupErr != nil {
		return cleanup, setupErr
	}
	setupErr = s.namespaceDatastore.AddNamespace(setupCtx, namespace2B)
	if setupErr != nil {
		return cleanup, setupErr
	}

	setupErr = s.datastore.UpsertImage(setupCtx, s.extraImage)
	if setupErr != nil {
		return cleanup, setupErr
	}
	setupErr = s.datastore.UpsertImage(setupCtx, image1)
	if setupErr != nil {
		return cleanup, setupErr
	}
	setupErr = s.datastore.UpsertImage(setupCtx, image2)
	if setupErr != nil {
		return cleanup, setupErr
	}

	setupErr = s.deploymentDatastore.UpsertDeployment(setupCtx, deployment1A1)
	if setupErr != nil {
		return cleanup, setupErr
	}
	setupErr = s.deploymentDatastore.UpsertDeployment(setupCtx, deployment2B1)
	if setupErr != nil {
		return cleanup, setupErr
	}
	setupErr = s.deploymentDatastore.UpsertDeployment(setupCtx, deployment2B2)
	if setupErr != nil {
		return cleanup, setupErr
	}

	return cleanup, nil
}

func (s *imageV2DatastoreSACSuite) TestCount() {
	cleanup, setupErr := s.setupSearchTest()
	defer cleanup()
	s.Require().NoError(setupErr)

	s.runSearchTest("TestCount", func(key string, testCase map[string]bool) {
		ctx := s.testContexts[key]
		expectedCount := 0
		for _, visible := range testCase {
			if visible {
				expectedCount++
			}
		}
		count, err := s.datastore.Count(ctx, searchPkg.EmptyQuery())
		s.NoError(err)
		s.Equal(expectedCount, count)
	})
}

func (s *imageV2DatastoreSACSuite) TestSearch() {
	cleanup, setupErr := s.setupSearchTest()
	defer cleanup()
	s.Require().NoError(setupErr)

	s.runSearchTest("TestSearch", func(key string, testCase map[string]bool) {
		ctx := s.testContexts[key]
		expectedIDs := make([]string, 0, len(testCase))
		for imageID, visible := range testCase {
			if visible {
				expectedIDs = append(expectedIDs, imageID)
			}
		}
		results, err := s.datastore.Search(ctx, searchPkg.EmptyQuery())
		s.NoError(err)
		resultIDHeap := make(map[string]struct{}, 0)
		for _, r := range results {
			resultIDHeap[r.ID] = struct{}{}
		}
		resultIDs := make([]string, 0, len(resultIDHeap))
		for k := range resultIDHeap {
			resultIDs = append(resultIDs, k)
		}
		s.ElementsMatch(expectedIDs, resultIDs)
	})
}

func (s *imageV2DatastoreSACSuite) TestSearchImages() {
	cleanup, setupErr := s.setupSearchTest()
	defer cleanup()
	s.Require().NoError(setupErr)

	s.runSearchTest("TestSearchImages", func(key string, testCase map[string]bool) {
		ctx := s.testContexts[key]
		expectedIDs := make([]string, 0, len(testCase))
		for imageID, visible := range testCase {
			if visible {
				expectedIDs = append(expectedIDs, imageID)
			}
		}
		results, err := s.datastore.SearchImages(ctx, searchPkg.EmptyQuery())
		s.NoError(err)
		resultIDHeap := make(map[string]struct{}, 0)
		for _, r := range results {
			resultIDHeap[r.GetId()] = struct{}{}
		}
		resultIDs := make([]string, 0, len(resultIDHeap))
		for k := range resultIDHeap {
			resultIDs = append(resultIDs, k)
		}
		s.ElementsMatch(expectedIDs, resultIDs)
	})
}

func (s *imageV2DatastoreSACSuite) TestSearchRawImages() {
	cleanup, setupErr := s.setupSearchTest()
	defer cleanup()
	s.Require().NoError(setupErr)
	refImages := map[string]*storage.ImageV2{
		s.extraImage.GetId():                         s.extraImage,
		fixtures.GetImageV2SherlockHolmes1().GetId(): fixtures.GetImageV2SherlockHolmes1(),
		fixtures.GetImageV2DoctorJekyll2().GetId():   fixtures.GetImageV2DoctorJekyll2(),
	}

	s.runSearchTest("TestSearchRawImages", func(key string, testCase map[string]bool) {
		ctx := s.testContexts[key]
		expectedIDs := make([]string, 0, len(testCase))
		for imageID, visible := range testCase {
			if visible {
				expectedIDs = append(expectedIDs, imageID)
			}
		}
		results, err := s.datastore.SearchRawImages(ctx, searchPkg.EmptyQuery())
		s.NoError(err)
		resultImages := make(map[string]*storage.ImageV2, 0)
		for _, r := range results {
			resultImages[r.GetId()] = r
		}
		resultIDs := make([]string, 0, len(resultImages))
		for k := range resultImages {
			resultIDs = append(resultIDs, k)
		}
		s.ElementsMatch(expectedIDs, resultIDs)
		for _, imageID := range expectedIDs {
			s.verifyRawImagesEqual(refImages[imageID], resultImages[imageID])
		}
	})
}

func (s *imageV2DatastoreSACSuite) runSearchTest(testName string, testFunc func(key string, c map[string]bool)) {
	imageGraphBefore := graphDBTestUtils.GetImageGraph(
		sac.WithAllAccess(context.Background()),
		s.T(),
		s.pgtestbase.DB,
	)

	cases := s.getSearchTestCases()
	failed := false
	for key, testCase := range cases {
		caseSucceeded := s.Run(key, func() {
			// When triggered in parallel, most tests fail.
			// TearDownTest is executed before the sub-tests.
			// See https://github.com/stretchr/testify/issues/934
			// s.T().Parallel()
			testFunc(key, testCase)
		})
		if !caseSucceeded {
			failed = true
		}
	}
	if failed {
		log.Infof("%s failed, dumping DB content.", testName)
		imageGraphBefore.Log()
	}
}
