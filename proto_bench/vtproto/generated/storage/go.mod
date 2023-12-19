module github.com/stackrox/stackrox/proto_bench/vtproto/generated/storage

go 1.20

replace (
	github.com/gogo/protobuf => github.com/connorgorman/protobuf v1.2.2-0.20210115205927-b892c1b298f7
	github.com/stackrox/rox => github.com/stackrox/stackrox v0.0.0-20231011153947-54855479c1ba

	go.uber.org/zap => github.com/stackrox/zap v1.15.1-0.20230918235618-2bd149903d0e
)

require (
	github.com/stackrox/rox v0.0.0-00010101000000-000000000000
	google.golang.org/protobuf v1.31.0
)

require (
	github.com/cloudflare/cfssl v1.6.4 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/gofrs/uuid v4.4.0+incompatible // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/certificate-transparency-go v1.1.6 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.6 // indirect
	github.com/jmoiron/sqlx v1.3.5 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/testify v1.8.4 // indirect
	github.com/weppos/publicsuffix-go v0.20.1-0.20221031080346-e4081aa8a6de // indirect
	github.com/zmap/zcrypto v0.0.0-20220402174210-599ec18ecbac // indirect
	github.com/zmap/zlint/v3 v3.4.0 // indirect
	go.etcd.io/bbolt v1.3.7 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/mock v0.3.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.25.0 // indirect
	golang.org/x/crypto v0.14.0 // indirect
	golang.org/x/exp v0.0.0-20230510235704-dd950f8aeaea // indirect
	golang.org/x/net v0.17.0 // indirect
	golang.org/x/sys v0.13.0 // indirect
	golang.org/x/text v0.13.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/apimachinery v0.28.2 // indirect
	k8s.io/klog/v2 v2.100.1 // indirect
	k8s.io/utils v0.0.0-20230505201702-9f6742963106 // indirect
)
