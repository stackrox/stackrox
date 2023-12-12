package protoconvert

//go:generate protoconvert-wrapper --from test.TestClone --from-path github.com/stackrox/rox/generated/test --to test2.TestClone --to-path github.com/stackrox/rox/generated/test2 --bidirectional --file testclone.go
//go:generate protoconvert-wrapper --from storage.ImageIntegration --to v1.ImageIntegration --bidirectional --file imageintegration.go
