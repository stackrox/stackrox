package datastore

import (
	"database/sql"
	"fmt"
	"testing"

	_ "github.com/lib/pq"
	alertPGIndex "github.com/stackrox/rox/central/alert/datastore/internal/index/postgres"
	alertPGStore "github.com/stackrox/rox/central/alert/datastore/internal/store/postgres"
	deploymentPGIndex "github.com/stackrox/rox/central/deployment/index/postgres"
	deploymentPGStore "github.com/stackrox/rox/central/deployment/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/search"
)

func getDeployment() *storage.Deployment {
	dep := fixtures.GetDeployment()
	dep.Ports = []*storage.PortConfig {
		{
			ExposureInfos:        []*storage.PortConfig_ExposureInfo {
				{
					ServiceName: "pc1",
					ServiceClusterIp: "ip1",
				},
				{
					ServiceName: "pc2",
					ServiceClusterIp: "ip2",
				},
			},
		},
		{
			ExposureInfos:        []*storage.PortConfig_ExposureInfo {
				{
					ServiceName: "pc1",
					ServiceClusterIp: "ip2",
				},
				{
					ServiceName: "pc2",
					ServiceClusterIp: "ip1",
				},
				{
					ServiceName: "pc1",
					ServiceClusterIp: "ip3",
				},
			},
		},
	}
	return dep
}

func TestT(t *testing.T) {
	source := "host=localhost port=5432 user=postgres sslmode=disable statement_timeout=60000"
	db, err := sql.Open("postgres", source)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	depStore := deploymentPGStore.New(db)
	fmt.Println(depStore)
	depIndex := deploymentPGIndex.NewIndexer(db)
	fmt.Println(depIndex)

	alertStore := alertPGStore.New(db)
	fmt.Println(alertStore)
	alertIndex := alertPGIndex.NewIndexer(db)
	fmt.Println(alertIndex)

	//if err := alertStore.Upsert(fixtures.GetAlert()); err != nil {
	//	panic(err)
	//}
	qb := search.NewQueryBuilder().
		AddStrings(search.ResourceType, "deployment")
		//AddStrings(
		//	search.ViolationState,
		//	storage.ViolationState_ACTIVE.String(),
		//	storage.ViolationState_ATTEMPTED.String()).
		//AddStrings(search.Cluster, "remote").
	results, err := alertIndex.Search(qb.ProtoQuery(), nil)
	if err != nil {
		panic(err)
	}
	fmt.Println("alert results", len(results))


	//
	//for i := 0; i < 10000; i++ {
	//	dep := getDeployment()
	//	dep.Id = uuid.NewV4().String()
	//	for j, c := range dep.GetContainers() {
	//		c.Name = fmt.Sprintf("%d-%d", i, j)
	//	}
	//	if err := store.Upsert(dep); err != nil {
	//		panic(err)
	//	}
	//}

	// deployment name
	//     float memory_mb_limit   = 4 [(gogoproto.moretags) = "search:\"Memory Limit (MB),store\""];
	/*
	      string key      = 1 [(gogoproto.moretags) = "search:\"Environment Key,store\" sql:\"pk\""];
	           string value    = 2 [(gogoproto.moretags) = "search:\"Environment Value,store\""];
	       repeated string add_capabilities    = 4 [(gogoproto.moretags) = "search:\"Add Capabilities,store\""];
	   	        string service_name       = 2 [(gogoproto.moretags) = "search:\"Exposing Service,store\""];
	*/
	query := search.NewQueryBuilder().
		AddStrings(search.DeploymentName, "NGINX").
		AddStrings(search.EnvironmentValue, "envvalue").
		AddStrings(search.AddCapabilities, "r/SYS.*", "NET_RAW").
		AddStrings(search.ExposingService, "pc1").
		AddStrings(search.ExposureLevel, storage.PortConfig_UNSET.String()).
		ProtoQuery()

	results, err = depIndex.Search(query)
	if err != nil {
		panic(err)
	}
	for _, res := range results {
		fmt.Println(res)
	}




	//
	//_, err = db.Exec(createTable)
	//if err != nil{
	//	panic(err)
	//}
	//
	//dep := fixtures.GetDeployment()
	//
	//dep.Containers = append(dep.Containers, dep.Containers[0])
	//dep.Containers[1].Name = "container2"
	//dep.Containers[1].Ports = []*storage.PortConfig {
	//	{
	//		ExposureInfos:        []*storage.PortConfig_ExposureInfo {
	//			{
	//				ServiceName: "pc1",
	//				ServiceClusterIp: "ip1",
	//			},
	//			{
	//				ServiceName: "pc2",
	//				ServiceClusterIp: "ip2",
	//			},
	//		},
	//	},
	//	{
	//		ExposureInfos:        []*storage.PortConfig_ExposureInfo {
	//			{
	//				ServiceName: "pc1",
	//				ServiceClusterIp: "ip2",
	//			},
	//			{
	//				ServiceName: "pc2",
	//				ServiceClusterIp: "ip1",
	//			},
	//			{
	//				ServiceName: "pc1",
	//				ServiceClusterIp: "ip3",
	//			},
	//		},
	//	},
	//}
	//
	//data, err := (&jsonpb.Marshaler{Indent: "  "}).MarshalToString(dep)
	//if err != nil {
	//	panic(err)
	//}
	//
	//fmt.Println(string(data))
	//
	//stmt, err := db.Prepare("INSERT INTO Deployment(id, value) VALUES($1, $2)")
	//if err != nil {
	//	panic(err)
	//}
	//
	//t := time.Now()
	//_, err = stmt.Exec(dep.GetId(), []byte(data))
	//fmt.Printf("Took %d ms to insert", time.Since(t).Milliseconds())
	//if err != nil {
	//	panic(err)
	//}
	//
	//select
	//	schools->>'school_id' school_id,
	//	addresses->>'addr_id' addr_id,
	//	addresses->>'house_description' house_description,
	//	addresses->>'house_no' house_no
	//	from title_register_data,
	//	jsonb_array_elements(address_data->'schools') schools,
	//	jsonb_array_elements(schools->'addresses') addresses
	//	where addresses->>'house_description' = addresses->>'house_no';
	//

}
