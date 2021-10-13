package postgres

import (
	"database/sql"
	"fmt"
	"testing"

	_ "github.com/lib/pq"
)

func TestStore(t *testing.T) {
	source := "host=localhost port=5432 user=postgres password=3CF2H6RFAf8wp1WhrXuQUAhlo sslmode=disable statement_timeout=60000"
	db, err := sql.Open("postgres", source)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		panic(err)
	}

	store := New(db)

	risk, exists, err := store.Get("id1")
	fmt.Println(risk, exists, err)
	return
	//
	//err = store.Upsert(&storage.Risk{
	//	Id:                   "id1",
	//	Subject:              &storage.RiskSubject{
	//		Id:                   "deployment",
	//		Namespace:            "stackrox",
	//		ClusterId:            "clusteriod",
	//		Type:                 storage.RiskSubjectType_IMAGE,
	//	},
	//	Score:                12,
	//	Results:              []*storage.Risk_Result {
	//		{
	//			Name: "factor1",
	//			Factors: []*storage.Risk_Result_Factor {
	//				{
	//					Message: "message1",
	//				},
	//				{
	//					Message: "message2",
	//				},
	//			},
	//			Score: 22,
	//		},
	//		{
	//			Name: "factor2",
	//			Factors: []*storage.Risk_Result_Factor {
	//				{
	//					Message: "message1",
	//				},
	//				{
	//					Message: "message2",
	//				},
	//			},
	//			Score: 22,
	//		},
	//		{
	//			Name: "factor3",
	//			Factors: []*storage.Risk_Result_Factor {
	//				{
	//					Message: "message10",
	//				},
	//				{
	//					Message: "message20",
	//				},
	//			},
	//			Score: 15,
	//		},
	//	},
	//})
	//if err != nil {
	//	panic(err)
	//}
	//
	//exists, err := store.Exists("id1")
	//fmt.Println(exists, err)
	//
	//count, err := store.Count()
	//fmt.Println(count, err)
	//
	//risk, exists, err := store.Get("id1")
	//fmt.Println(risk, exists, err)

}
