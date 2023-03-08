// //go:build sql_integration

package reprocessor

import (
	"context"
	"testing"
	"time"

	componentCVEEdgeDackbox "github.com/stackrox/rox/central/componentcveedge/dackbox"
	componentCVEEdgeIndex "github.com/stackrox/rox/central/componentcveedge/index"
	cveDackbox "github.com/stackrox/rox/central/cve/dackbox"
	cveIndex "github.com/stackrox/rox/central/cve/index"
	deploymentDackbox "github.com/stackrox/rox/central/deployment/dackbox"
	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	deploymentIndex "github.com/stackrox/rox/central/deployment/index"
	"github.com/stackrox/rox/central/globalindex"
	indexDackbox "github.com/stackrox/rox/central/image/dackbox"
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	imagePG "github.com/stackrox/rox/central/image/datastore/store/postgres"
	imageIndex "github.com/stackrox/rox/central/image/index"
	imageComponentDackbox "github.com/stackrox/rox/central/imagecomponent/dackbox"
	imageComponentIndex "github.com/stackrox/rox/central/imagecomponent/index"
	imageComponentEdgeDackbox "github.com/stackrox/rox/central/imagecomponentedge/dackbox"
	imageComponentEdgeIndex "github.com/stackrox/rox/central/imagecomponentedge/index"
	processlisteningonportDatastore "github.com/stackrox/rox/central/processlisteningonport/datastore"
	postgresStore "github.com/stackrox/rox/central/processlisteningonport/store/postgres"
	processIndicatorDataStore "github.com/stackrox/rox/central/processindicator/datastore"
        processIndicatorSearch "github.com/stackrox/rox/central/processindicator/search"
        processIndicatorStorage "github.com/stackrox/rox/central/processindicator/store/postgres"
	"github.com/stackrox/rox/central/ranking"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dackbox"
	dackboxConcurrency "github.com/stackrox/rox/pkg/dackbox/concurrency"
	"github.com/stackrox/rox/pkg/dackbox/indexer"
	"github.com/stackrox/rox/pkg/dackbox/utils/queue"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/process/filter"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/protoconv"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/assert"
)

func TestGetActiveImageIDs(t *testing.T) {
	t.Parallel()

	testCtx := sac.WithAllAccess(context.Background())

	var (
		pool          *postgres.DB
		imageDS       imageDatastore.DataStore
		deploymentsDS deploymentDatastore.DataStore
		indexingQ     queue.WaitableQueue
		err           error
	)

	if env.PostgresDatastoreEnabled.BooleanSetting() {
		testingDB := pgtest.ForT(t)
		pool = testingDB.DB
		defer pool.Close()

		//testingDB.Teardown(t)

		imageDS = imageDatastore.NewWithPostgres(imagePG.New(pool, false, dackboxConcurrency.NewKeyFence()), imagePG.NewIndexer(pool), nil, ranking.ImageRanker(), ranking.ComponentRanker())
		deploymentsDS, err = deploymentDatastore.New(nil, dackboxConcurrency.NewKeyFence(), pool, nil, nil, nil, nil, nil, nil,
			nil, filter.NewFilter(5, []int{5}), ranking.NewRanker(), ranking.NewRanker(), ranking.NewRanker())
		require.NoError(t, err)
	} else {
		rocksDB := rocksdbtest.RocksDBForT(t)

		indexingQ = queue.NewWaitableQueue()
		dacky, err := dackbox.NewRocksDBDackBox(rocksDB, indexingQ, []byte("graph"), []byte("dirty"), []byte("valid"))
		require.NoError(t, err)

		bleveIndex, err := globalindex.MemOnlyIndex()
		require.NoError(t, err)

		reg := indexer.NewWrapperRegistry()
		lazy := indexer.NewLazy(indexingQ, reg, bleveIndex, dacky.AckIndexed)
		lazy.Start()

		reg.RegisterWrapper(deploymentDackbox.Bucket, deploymentIndex.Wrapper{})
		reg.RegisterWrapper(indexDackbox.Bucket, imageIndex.Wrapper{})
		reg.RegisterWrapper(cveDackbox.Bucket, cveIndex.Wrapper{})
		reg.RegisterWrapper(componentCVEEdgeDackbox.Bucket, componentCVEEdgeIndex.Wrapper{})
		reg.RegisterWrapper(imageComponentDackbox.Bucket, imageComponentIndex.Wrapper{})
		reg.RegisterWrapper(imageComponentEdgeDackbox.Bucket, imageComponentEdgeIndex.Wrapper{})

		imageDS = imageDatastore.New(dacky, dackboxConcurrency.NewKeyFence(), bleveIndex, bleveIndex, false, nil, ranking.NewRanker(), ranking.NewRanker())

		deploymentsDS, err = deploymentDatastore.New(dacky, dackboxConcurrency.NewKeyFence(), nil, bleveIndex, bleveIndex, nil, nil, nil, nil,
			nil, filter.NewFilter(5, []int{5}), ranking.NewRanker(), ranking.NewRanker(), ranking.NewRanker())
		require.NoError(t, err)
	}

	loop := NewLoop(nil, nil, nil, deploymentsDS, imageDS, nil, nil, nil, nil, nil, queue.NewWaitableQueue()).(*loopImpl)

	ids, err := loop.getActiveImageIDs()
	require.NoError(t, err)
	require.Equal(t, 0, len(ids))

	deployment := fixtures.GetDeployment()
	require.NoError(t, deploymentsDS.UpsertDeployment(testCtx, deployment))

	images := fixtures.DeploymentImages()
	imageIDs := make([]string, 0, len(images))
	for _, image := range images {
		require.NoError(t, imageDS.UpsertImage(testCtx, image))
		imageIDs = append(imageIDs, image.GetId())
	}

	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		newSig := concurrency.NewSignal()
		indexingQ.PushSignal(&newSig)
		newSig.Wait()
	}

	ids, err = loop.getActiveImageIDs()
	require.NoError(t, err)
	require.ElementsMatch(t, imageIDs, ids)
}

func TestProcessListeningOnPortReprocess(t *testing.T) {

	testCtx := sac.WithAllAccess(context.Background())
	testNamespace := "test_namespace"

	var (
		pool          *postgres.DB
		plops		processlisteningonportDatastore.DataStore
	)

	if env.PostgresDatastoreEnabled.BooleanSetting() {
		testingDB := pgtest.ForT(t)
		pool = testingDB.DB
		defer pool.Close()
	}

	store := postgresStore.NewFullStore(pool)

        indicatorStorage := processIndicatorStorage.New(pool)
        indicatorIndexer := processIndicatorStorage.NewIndexer(pool)
        indicatorSearcher := processIndicatorSearch.New(indicatorStorage, indicatorIndexer)

	indicatorDataStore, _ := processIndicatorDataStore.New(
                indicatorStorage, store, indicatorIndexer, indicatorSearcher, nil)

	plops = processlisteningonportDatastore.New(store, indicatorDataStore)

	loop := NewLoop(nil, nil, nil, nil, nil, nil, nil, nil, nil, plops, queue.NewWaitableQueue()).(*loopImpl)


        plopObjects := []*storage.ProcessListeningOnPortFromSensor{
                {
                        Port:           1234,
                        Protocol:       storage.L4Protocol_L4_PROTOCOL_TCP,
                        CloseTimestamp: nil,
                        Process: &storage.ProcessIndicatorUniqueKey{
                                PodId:               fixtureconsts.PodUID1,
                                ContainerName:       "test_container1",
                                ProcessName:         "test_process1",
                                ProcessArgs:         "test_arguments1",
                                ProcessExecFilePath: "test_path1",
                        },
                },
        }

        // Verify that the table is empty before the test
        plopsFromDB := loop.plops.GetPlopsFromDB(testCtx)
        assert.Equal(t, len(plopsFromDB), 0)

	loop.plops.AddProcessListeningOnPort(testCtx, plopObjects...)

	indicators := []*storage.ProcessIndicator{
	       {
	               Id:            fixtureconsts.ProcessIndicatorID1,
	               DeploymentId:  fixtureconsts.Deployment1,
	               PodId:         fixtureconsts.PodUID1,
	               ClusterId:     fixtureconsts.Cluster1,
	               ContainerName: "test_container1",
	               Namespace:     testNamespace,

	               Signal: &storage.ProcessSignal{
	                       Name:         "test_process1",
	                       Args:         "test_arguments1",
	                       ExecFilePath: "test_path1",
	               },
	       },
       }

       indicatorDataStore.AddProcessIndicators(testCtx, indicators...)

        plopsFromDB = loop.plops.GetPlopsFromDB(testCtx)
        assert.Equal(t, len(plopsFromDB), 1)

	expectedPlopStorage := []*storage.ProcessListeningOnPortStorage{
		{
			Id:                 plopsFromDB[0].GetId(),
			Port:               plopObjects[0].GetPort(),
			Protocol:           plopObjects[0].GetProtocol(),
			CloseTimestamp:     nil,
			ProcessIndicatorId: "",
			Closed:             false,
			Process:            plopObjects[0].Process,
		},
        }


        assert.Equal(t, expectedPlopStorage[0], plopsFromDB[0])

	loop.runProcessListeningOnPortReprocessing()

        plopsFromDB = loop.plops.GetPlopsFromDB(testCtx)
        assert.Equal(t, len(plopsFromDB), 1)

	expectedPlopStorage = []*storage.ProcessListeningOnPortStorage{
		{
			Id:                 plopsFromDB[0].GetId(),
			Port:               plopObjects[0].GetPort(),
			Protocol:           plopObjects[0].GetProtocol(),
			CloseTimestamp:     nil,
			ProcessIndicatorId: indicators[0].Id,
			Closed:             false,
			Process:            nil,
		},
        }

        assert.Equal(t, expectedPlopStorage[0], plopsFromDB[0])

	newPlops, _ := loop.plops.GetProcessListeningOnPort(testCtx, fixtureconsts.Deployment1)

	assert.Equal(t, 1, len(newPlops))

	expectedNewPlops := []*storage.ProcessListeningOnPort{
		{
			ContainerName: "test_container1",
			PodId:         fixtureconsts.PodUID1,
			DeploymentId:  fixtureconsts.Deployment1,
			ClusterId:     fixtureconsts.Cluster1,
			Namespace:     testNamespace,
			Endpoint: &storage.ProcessListeningOnPort_Endpoint{
			        Port:     1234,
			        Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			},
			Signal: &storage.ProcessSignal{
			        Name:         "test_process1",
			        Args:         "test_arguments1",
			        ExecFilePath: "test_path1",
			},
		},
        }

	assert.Equal(t, expectedNewPlops, newPlops)
}

func TestProcessListeningOnPortReprocessCloseBeforeRetrying(t *testing.T) {

	testCtx := sac.WithAllAccess(context.Background())
	testNamespace := "test_namespace"

	var (
		pool          *postgres.DB
		plops		processlisteningonportDatastore.DataStore
	)

	if env.PostgresDatastoreEnabled.BooleanSetting() {
		testingDB := pgtest.ForT(t)
		pool = testingDB.DB
		defer pool.Close()
	}

	store := postgresStore.NewFullStore(pool)

        indicatorStorage := processIndicatorStorage.New(pool)
        indicatorIndexer := processIndicatorStorage.NewIndexer(pool)
        indicatorSearcher := processIndicatorSearch.New(indicatorStorage, indicatorIndexer)

	indicatorDataStore, _ := processIndicatorDataStore.New(
                indicatorStorage, store, indicatorIndexer, indicatorSearcher, nil)

	plops = processlisteningonportDatastore.New(store, indicatorDataStore)

	loop := NewLoop(nil, nil, nil, nil, nil, nil, nil, nil, nil, plops, queue.NewWaitableQueue()).(*loopImpl)


        plopObjects := []*storage.ProcessListeningOnPortFromSensor{
                {
                        Port:           1234,
                        Protocol:       storage.L4Protocol_L4_PROTOCOL_TCP,
                        CloseTimestamp: nil,
                        Process: &storage.ProcessIndicatorUniqueKey{
                                PodId:               fixtureconsts.PodUID1,
                                ContainerName:       "test_container1",
                                ProcessName:         "test_process1",
                                ProcessArgs:         "test_arguments1",
                                ProcessExecFilePath: "test_path1",
                        },
                },
        }

        // Verify that the table is empty before the test
        plopsFromDB := loop.plops.GetPlopsFromDB(testCtx)
        assert.Equal(t, len(plopsFromDB), 0)

	loop.plops.AddProcessListeningOnPort(testCtx, plopObjects...)

	indicators := []*storage.ProcessIndicator{
	       {
	               Id:            fixtureconsts.ProcessIndicatorID1,
	               DeploymentId:  fixtureconsts.Deployment1,
	               PodId:         fixtureconsts.PodUID1,
	               ClusterId:     fixtureconsts.Cluster1,
	               ContainerName: "test_container1",
	               Namespace:     testNamespace,

	               Signal: &storage.ProcessSignal{
	                       Name:         "test_process1",
	                       Args:         "test_arguments1",
	                       ExecFilePath: "test_path1",
	               },
	       },
       }

       indicatorDataStore.AddProcessIndicators(testCtx, indicators...)

        plopsFromDB = loop.plops.GetPlopsFromDB(testCtx)
        assert.Equal(t, len(plopsFromDB), 1)

	expectedPlopStorage := []*storage.ProcessListeningOnPortStorage{
		{
			Id:                 plopsFromDB[0].GetId(),
			Port:               plopObjects[0].GetPort(),
			Protocol:           plopObjects[0].GetProtocol(),
			CloseTimestamp:     nil,
			ProcessIndicatorId: "",
			Closed:             false,
			Process:            plopObjects[0].Process,
		},
        }


        assert.Equal(t, expectedPlopStorage[0], plopsFromDB[0])

        closedPlopObjects := []*storage.ProcessListeningOnPortFromSensor{
                {
                        Port:           1234,
                        Protocol:       storage.L4Protocol_L4_PROTOCOL_TCP,
                        CloseTimestamp: protoconv.ConvertTimeToTimestamp(time.Now()),
                        Process: &storage.ProcessIndicatorUniqueKey{
                                PodId:               fixtureconsts.PodUID1,
                                ContainerName:       "test_container1",
                                ProcessName:         "test_process1",
                                ProcessArgs:         "test_arguments1",
                                ProcessExecFilePath: "test_path1",
                        },
                },
        }

	loop.plops.AddProcessListeningOnPort(testCtx, closedPlopObjects...)

	loop.runProcessListeningOnPortReprocessing()

        plopsFromDB = loop.plops.GetPlopsFromDB(testCtx)
        assert.Equal(t, len(plopsFromDB), 1)

	expectedPlopStorage = []*storage.ProcessListeningOnPortStorage{
		{
			Id:                 plopsFromDB[0].GetId(),
			Port:               plopObjects[0].GetPort(),
			Protocol:           plopObjects[0].GetProtocol(),
			CloseTimestamp:     closedPlopObjects[0].GetCloseTimestamp(),
			ProcessIndicatorId: indicators[0].Id,
			Closed:             true,
			Process:            nil,
		},
        }

        assert.Equal(t, expectedPlopStorage[0], plopsFromDB[0])

	newPlops, _ := loop.plops.GetProcessListeningOnPort(testCtx, fixtureconsts.Deployment1)

	assert.Equal(t, len(newPlops), 0)
}

// Currently this fails. 
// The test does the following things
// 1. Adds an open plop with no matching indicator
// 2. Adds the indicator for the plop 
// 3. Adds a batch where the plop is closed and then opened
// 4. Retries the plops that were not matched to processes
//
// Currently the result is that the plop is recorded as being closed which is incorrect
// The reason for this is that when the batch with the open and close plop is added
// it does not know about the open plop that was not matched to a process and it does
// not know about the order of the plops in the batch so it thinks the plop was opened and then closed
// Then when it goes to do the retry it ignores the first unmatched plop, because the plop is already
// in the table.
func TestProcessListeningOnPortReprocessBatchBeforeRetrying(t *testing.T) {


	testCtx := sac.WithAllAccess(context.Background())
	testNamespace := "test_namespace"

	var (
		pool          *postgres.DB
		plops		processlisteningonportDatastore.DataStore
	)

	if env.PostgresDatastoreEnabled.BooleanSetting() {
		testingDB := pgtest.ForT(t)
		pool = testingDB.DB
		defer pool.Close()
	}

	store := postgresStore.NewFullStore(pool)

        indicatorStorage := processIndicatorStorage.New(pool)
        indicatorIndexer := processIndicatorStorage.NewIndexer(pool)
        indicatorSearcher := processIndicatorSearch.New(indicatorStorage, indicatorIndexer)

	indicatorDataStore, _ := processIndicatorDataStore.New(
                indicatorStorage, store, indicatorIndexer, indicatorSearcher, nil)

	plops = processlisteningonportDatastore.New(store, indicatorDataStore)

	loop := NewLoop(nil, nil, nil, nil, nil, nil, nil, nil, nil, plops, queue.NewWaitableQueue()).(*loopImpl)


        // Verify that the table is empty before the test
        plopsFromDB := loop.plops.GetPlopsFromDB(testCtx)
        assert.Equal(t, 0, len(plopsFromDB))

	openPlopObject := &storage.ProcessListeningOnPortFromSensor{
	        Port:           1234,
	        Protocol:       storage.L4Protocol_L4_PROTOCOL_TCP,
	        CloseTimestamp: nil,
	        Process: &storage.ProcessIndicatorUniqueKey{
	                PodId:               fixtureconsts.PodUID1,
	                ContainerName:       "test_container1",
	                ProcessName:         "test_process1",
	                ProcessArgs:         "test_arguments1",
	                ProcessExecFilePath: "test_path1",
	        },
	}

        plopObjects := []*storage.ProcessListeningOnPortFromSensor{openPlopObject}

	loop.plops.AddProcessListeningOnPort(testCtx, plopObjects...)

	indicators := []*storage.ProcessIndicator{
	       {
	               Id:            fixtureconsts.ProcessIndicatorID1,
	               DeploymentId:  fixtureconsts.Deployment1,
	               PodId:         fixtureconsts.PodUID1,
	               ClusterId:     fixtureconsts.Cluster1,
	               ContainerName: "test_container1",
	               Namespace:     testNamespace,

	               Signal: &storage.ProcessSignal{
	                       Name:         "test_process1",
	                       Args:         "test_arguments1",
	                       ExecFilePath: "test_path1",
	               },
	       },
       }

       indicatorDataStore.AddProcessIndicators(testCtx, indicators...)

        plopsFromDB = loop.plops.GetPlopsFromDB(testCtx)
        assert.Equal(t, 1, len(plopsFromDB))

	expectedPlopStorage := []*storage.ProcessListeningOnPortStorage{
		{
			Id:                 plopsFromDB[0].GetId(),
			Port:               plopObjects[0].GetPort(),
			Protocol:           plopObjects[0].GetProtocol(),
			CloseTimestamp:     nil,
			ProcessIndicatorId: "",
			Closed:             false,
			Process:            plopObjects[0].Process,
		},
        }


        assert.Equal(t, expectedPlopStorage[0], plopsFromDB[0])

	closedPlopObject := &storage.ProcessListeningOnPortFromSensor{
	        Port:           1234,
	        Protocol:       storage.L4Protocol_L4_PROTOCOL_TCP,
                CloseTimestamp: protoconv.ConvertTimeToTimestamp(time.Now()),
	        Process: &storage.ProcessIndicatorUniqueKey{
	                PodId:               fixtureconsts.PodUID1,
	                ContainerName:       "test_container1",
	                ProcessName:         "test_process1",
	                ProcessArgs:         "test_arguments1",
	                ProcessExecFilePath: "test_path1",
	        },
	}

        batchPlopObjects := []*storage.ProcessListeningOnPortFromSensor{closedPlopObject, openPlopObject}

	loop.plops.AddProcessListeningOnPort(testCtx, batchPlopObjects...)

	loop.runProcessListeningOnPortReprocessing()

        plopsFromDB = loop.plops.GetPlopsFromDB(testCtx)
        assert.Equal(t, 1, len(plopsFromDB))

	expectedPlopStorage = []*storage.ProcessListeningOnPortStorage{
		{
			Id:                 plopsFromDB[0].GetId(),
			Port:               plopObjects[0].GetPort(),
			Protocol:           plopObjects[0].GetProtocol(),
			CloseTimestamp:     nil,
			ProcessIndicatorId: indicators[0].Id,
			Closed:             false,
			Process:            nil,
		},
        }

        assert.Equal(t, expectedPlopStorage[0], plopsFromDB[0])

	newPlops, _ := loop.plops.GetProcessListeningOnPort(testCtx, fixtureconsts.Deployment1)

	assert.Equal(t, 1, len(newPlops))

	expectedNewPlops := []*storage.ProcessListeningOnPort{
		{
			ContainerName: "test_container1",
			PodId:         fixtureconsts.PodUID1,
			DeploymentId:  fixtureconsts.Deployment1,
			ClusterId:     fixtureconsts.Cluster1,
			Namespace:     testNamespace,
			Endpoint: &storage.ProcessListeningOnPort_Endpoint{
			        Port:     1234,
			        Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			},
			Signal: &storage.ProcessSignal{
			        Name:         "test_process1",
			        Args:         "test_arguments1",
			        ExecFilePath: "test_path1",
			},
		},
        }

	assert.Equal(t, expectedNewPlops, newPlops)

}
