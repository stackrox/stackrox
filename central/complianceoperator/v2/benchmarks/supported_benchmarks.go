package benchmarks

//var benchmarkToComplianceOperatorMapping = map[string]*storage.ComplianceOperatorBenchmark{
//	"cis-ocp-1.4.0": {
//		Name: "CIS Red Hat OpenShift Container Platform Benchmark v1.4.0",
//		//TODO: Add version and more metadata to a benchmark
//	},
//}
//
//// GetByComplianceOperatorLabel returns a benchmark by its Compliance Operator Label.
//func GetByComplianceOperatorLabel(name string) *storage.ComplianceOperatorBenchmark {
//	return benchmarkToComplianceOperatorMapping[name]
//}

// initializeBenchmarks upserts all supported benchmark profiles. A compliance operator profile can implement a benchmark.
//// On sync with compliance operator a link between a benchmark and a profile can be created automatically.
//// TODO(question): How to import static data to Central?
//func initializeBenchmarks(ctx context.Context) error {
//	db := globaldb.GetPostgres()
//	store := postgres.New(db)
//	for _, benchmark := range benchmarkToComplianceOperatorMapping {
//		if err := store.Upsert(ctx, benchmark); err != nil {
//			return err
//		}
//	}
//	return nil
//}
