module github.com/stackrox/rox

go 1.16

require (
	cloud.google.com/go v0.70.0
	cloud.google.com/go/storage v1.12.0
	github.com/BurntSushi/toml v0.3.1
	github.com/Masterminds/semver v1.5.0
	github.com/Masterminds/sprig/v3 v3.2.2
	github.com/NYTimes/gziphandler v1.1.1
	github.com/PagerDuty/go-pagerduty v1.4.2
	github.com/RoaringBitmap/roaring v0.9.4
	github.com/VividCortex/ewma v1.2.0
	github.com/andygrunwald/go-jira v1.14.0
	github.com/antihax/optional v1.0.0
	github.com/aws/aws-sdk-go v1.40.36
	github.com/blevesearch/bleve v0.8.0
	github.com/cenkalti/backoff/v3 v3.2.2
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/ckaznocha/protoc-gen-lint v0.2.4
	github.com/cloudflare/cfssl v0.0.0-20190510060611-9c027c93ba9e
	github.com/containers/image/v5 v5.11.1
	github.com/coreos/etcd v3.3.17+incompatible
	github.com/coreos/go-oidc/v3 v3.0.0
	github.com/coreos/go-systemd/v22 v22.3.2
	github.com/dave/jennifer v1.4.1
	github.com/deckarep/golang-set v1.7.1
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker v1.4.2-0.20200203170920-46ec8731fbce
	github.com/docker/go-connections v0.4.0
	github.com/docker/go-units v0.4.0
	github.com/emicklei/proto v1.9.1 // indirect
	github.com/facebookincubator/nvdtools v0.1.4
	github.com/fsnotify/fsnotify v1.5.1 // indirect
	github.com/fullsailor/pkcs7 v0.0.0
	github.com/ghodss/yaml v1.0.0
	github.com/go-logr/logr v0.4.0
	github.com/godbus/dbus/v5 v5.0.4
	github.com/gofrs/uuid v4.0.0+incompatible
	github.com/gogo/protobuf v1.3.2
	github.com/golang-jwt/jwt v3.2.2+incompatible // indirect
	github.com/golang/mock v1.4.4
	github.com/golang/protobuf v1.5.2
	github.com/golangci/golangci-lint v1.33.0
	github.com/google/certificate-transparency-go v1.0.21
	github.com/google/go-cmp v0.5.6
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/googleapis/gnostic v0.5.1
	github.com/gookit/color v1.4.2
	github.com/gorilla/schema v1.2.0
	github.com/graph-gophers/graphql-go v1.1.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/hako/durafmt v0.0.0-20210608085754-5c1018a4e16b
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1
	github.com/hashicorp/go-version v1.3.0
	github.com/hashicorp/golang-lru v0.5.4
	github.com/heroku/docker-registry-client v0.0.0
	github.com/huandu/xstrings v1.3.2 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/itchyny/gojq v0.12.5
	github.com/joelanford/helm-operator v0.0.7
	github.com/jonboulle/clockwork v0.2.2 // indirect
	github.com/jstemmer/go-junit-report v0.9.1
	github.com/lib/pq v1.9.0
	github.com/machinebox/graphql v0.2.2
	github.com/magiconair/properties v1.8.5
	github.com/mailru/easyjson v0.7.7
	github.com/mattermost/xml-roundtrip-validator v0.1.0 // indirect
	github.com/mattn/goveralls v0.0.2
	github.com/mauricelam/genny v0.0.0-20190320071652-0800202903e5
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1
	github.com/mitchellh/hashstructure v1.1.0
	github.com/nilslice/protolock v0.0.0
	github.com/nxadm/tail v1.4.8
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.16.0
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.0.2-0.20190823105129-775207bd45b6
	github.com/openshift/api v3.9.1-0.20191201231411-9f834e337466+incompatible
	github.com/openshift/client-go v3.9.0+incompatible
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/operator-framework/operator-sdk v0.19.4
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.11.0
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.30.0
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/russellhaering/gosaml2 v0.6.0
	github.com/russellhaering/goxmldsig v1.1.0
	github.com/sergi/go-diff v1.2.0
	github.com/spf13/cast v1.4.1 // indirect
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	github.com/stackrox/anchore-client v0.0.0-20190929180200-981e05834836
	github.com/stackrox/default-authz-plugin v0.0.0-20210608105219-00ad9c9f3855
	github.com/stackrox/external-network-pusher v0.0.0-20210419192707-074af92bbfa7
	github.com/stackrox/k8s-istio-cve-pusher v0.0.0-20210422200002-d89f671ac4f5
	github.com/stackrox/scanner v0.0.0-20210817231857-4acbccb682b8
	github.com/stretchr/objx v0.3.0 // indirect
	github.com/stretchr/testify v1.7.0
	github.com/tecbot/gorocksdb v0.0.0-20190705090504-162552197222
	github.com/tkuchiki/go-timezone v0.1.3
	github.com/vbauerster/mpb/v4 v4.12.2
	go.etcd.io/bbolt v1.3.6
	go.uber.org/atomic v1.9.0
	go.uber.org/multierr v1.7.0 // indirect
	go.uber.org/zap v1.15.1-0.20200717220000-53a387079b46
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5
	golang.org/x/mod v0.5.0 // indirect
	golang.org/x/net v0.0.0-20210902165921-8d991716f632
	golang.org/x/oauth2 v0.0.0-20210514164344-f6687ab2804c
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/sys v0.0.0-20210903071746-97244b99971b
	golang.org/x/term v0.0.0-20210615171337-6886f2dfbf5b // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/time v0.0.0-20210723032227-1f47c861a9ac
	golang.org/x/tools v0.1.5
	golang.stackrox.io/grpc-http1 v0.2.3
	google.golang.org/api v0.33.0
	google.golang.org/genproto v0.0.0-20210831024726-fe130286e0e2
	google.golang.org/grpc v1.40.0
	google.golang.org/grpc/examples v0.0.0-20210902184326-c93e472777b9
	gopkg.in/robfig/cron.v2 v2.0.0-20150107220207-be2e0b0deed5
	gopkg.in/square/go-jose.v2 v2.6.0
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
	gotest.tools v2.2.0+incompatible
	helm.sh/helm/v3 v3.5.4
	honnef.co/go/tools v0.0.1-2020.1.6
	k8s.io/api v0.20.4
	k8s.io/apimachinery v0.20.4
	k8s.io/apiserver v0.20.4
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/kube-openapi v0.0.0-20201113171705-d219536bb9fd
	k8s.io/kubectl v0.20.4
	k8s.io/kubelet v0.20.4
	k8s.io/utils v0.0.0-20210820185131-d34e5cb4466e
	sigs.k8s.io/controller-runtime v0.8.3
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/blevesearch/bleve => github.com/stackrox/bleve v0.0.0-20200807170555-6c4fa9f5e726
	github.com/couchbase/ghistogram => github.com/couchbase/ghistogram v0.0.1-0.20170308220240-d910dd063dd6
	github.com/couchbase/vellum => github.com/couchbase/vellum v0.0.0-20190829182332-ef2e028c01fd
	github.com/dgraph-io/badger => github.com/stackrox/badger v1.6.1-0.20200807170638-4177b4beb2ed

	// github.com/deislabs/oras doesn't specify a real version here but uses a replace, which is replicated by
	// helm, which we now have to replicate, too...
	github.com/docker/distribution => github.com/docker/distribution v0.0.0-20191216044856-a8371794149d
	github.com/docker/docker => github.com/moby/moby v17.12.0-ce-rc1.0.20200618181300-9dc6525e6118+incompatible

	github.com/facebookincubator/nvdtools => github.com/stackrox/nvdtools v0.0.0-20210326191554-5daeb6395b56
	github.com/fullsailor/pkcs7 => github.com/misberner/pkcs7 v0.0.0-20190417093538-a48bf0f78dea
	github.com/go-resty/resty => gopkg.in/resty.v1 v1.11.0
	github.com/gogo/protobuf => github.com/connorgorman/protobuf v1.2.2-0.20210115205927-b892c1b298f7

	// Something pulls in an older version with uppercase OpenAPIv2 package version
	github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.5.3
	github.com/heroku/docker-registry-client => github.com/stackrox/docker-registry-client v0.0.0-20210302165330-43446b0a41b5
	github.com/joelanford/helm-operator => github.com/stackrox/helm-operator v0.0.8-0.20210706005254-7857f66cf95e
	github.com/mattn/goveralls => github.com/viswajithiii/goveralls v0.0.3-0.20190917224517-4dd02c532775

	// github.com/mikefarah/yaml/v2 is a clone of github.com/go-yaml/yaml/v2.
	// Both github.com/go-yaml/yaml/v2 and github.com/go-yaml/yaml/v3 do not provide go.sum
	// so dependabot is not able to check dependecies.
	// See https://github.com/go-yaml/yaml/issues/772
	// Therefore we point all to our fork of `go-yaml` - github.com/stackrox/yaml/v2|v3
	// where we provide the actual `go.sum`.
	github.com/mikefarah/yaml/v2 => gopkg.in/yaml.v2 v2.4.0

	github.com/nilslice/protolock => github.com/viswajithiii/protolock v0.10.1-0.20190117180626-43bb8a9ba4e8

	github.com/nxadm/tail => github.com/stackrox/tail v1.4.9-0.20210831224919-407035634f5d
	github.com/opencontainers/runc => github.com/opencontainers/runc v1.0.0-rc9
	github.com/openshift/client-go => github.com/openshift/client-go v0.0.0-20200623090625-83993cebb5ae
	github.com/tecbot/gorocksdb => github.com/DataDog/gorocksdb v0.0.0-20200107201226-9722c3a2e063
	go.uber.org/zap => github.com/stackrox/zap v1.15.1-0.20200720133746-810fd602fd0f
	golang.org/x/oauth2 => github.com/misberner/oauth2 v0.0.0-20200208204620-d153c71f6b8d

	gopkg.in/yaml.v2 => github.com/stackrox/yaml/v2 v2.4.1
	gopkg.in/yaml.v3 => github.com/stackrox/yaml/v3 v3.0.0

	honnef.co/go/tools => honnef.co/go/tools v0.0.1-2020.1.5
	k8s.io/api => k8s.io/api v0.20.4

	// Circular github.com/stackrox/rox sets this to an incompatible version
	k8s.io/client-go => k8s.io/client-go v0.20.4
	vbom.ml/util => github.com/fvbommel/util v0.0.0-20200828041400-c69461e88a36
)
