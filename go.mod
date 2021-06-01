module github.com/stackrox/rox

go 1.16

require (
	cloud.google.com/go v0.70.0
	cloud.google.com/go/storage v1.12.0
	github.com/BurntSushi/toml v0.3.1
	github.com/Masterminds/semver v1.5.0
	github.com/Masterminds/sprig/v3 v3.2.2
	github.com/NYTimes/gziphandler v1.1.1
	github.com/PagerDuty/go-pagerduty v0.0.0-20191002190746-f60f4fc45222
	github.com/RoaringBitmap/roaring v0.6.0
	github.com/VividCortex/ewma v1.2.0
	github.com/andygrunwald/go-jira v1.10.0
	github.com/antihax/optional v0.0.0-20180407024304-ca021399b1a6
	github.com/aws/aws-sdk-go v1.38.29
	github.com/blevesearch/bleve v0.8.0
	github.com/bugsnag/bugsnag-go v1.5.3 // indirect
	github.com/bugsnag/panicwrap v1.2.0 // indirect
	github.com/cenkalti/backoff/v3 v3.2.2
	github.com/ckaznocha/protoc-gen-lint v0.2.1
	github.com/cloudflare/cfssl v0.0.0-20190510060611-9c027c93ba9e
	github.com/containers/image/v5 v5.11.1
	github.com/coreos/etcd v3.3.17+incompatible
	github.com/coreos/go-oidc/v3 v3.0.0
	github.com/coreos/go-systemd/v22 v22.1.0
	github.com/dave/jennifer v1.4.1
	github.com/deckarep/golang-set v1.7.1
	github.com/dgraph-io/badger v0.0.0-20190131175406-28ef9bfd2438
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker v1.4.2-0.20200203170920-46ec8731fbce
	github.com/docker/go-connections v0.4.0
	github.com/docker/go-units v0.4.0
	github.com/facebookincubator/nvdtools v0.1.4
	github.com/fullsailor/pkcs7 v0.0.0
	github.com/garyburd/redigo v1.6.0 // indirect
	github.com/ghodss/yaml v1.0.0
	github.com/go-openapi/spec v0.19.5 // indirect
	github.com/godbus/dbus/v5 v5.0.3
	github.com/gofrs/uuid v4.0.0+incompatible
	github.com/gogo/protobuf v1.3.2
	github.com/golang/mock v1.4.4
	github.com/golang/protobuf v1.4.3
	github.com/golangci/golangci-lint v1.33.0
	github.com/google/certificate-transparency-go v1.0.21
	github.com/google/go-cmp v0.5.4
	github.com/googleapis/gnostic v0.5.1
	github.com/gookit/color v1.4.2
	github.com/gorilla/handlers v1.4.2 // indirect
	github.com/gorilla/schema v1.2.0
	github.com/graph-gophers/graphql-go v0.0.0-20190513003547-158e7b876106
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.0
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/grpc-ecosystem/grpc-gateway v1.11.4-0.20191004150533-c677e419aa5c
	github.com/hako/durafmt v0.0.0-20180520121703-7b7ae1e72ead
	github.com/hashicorp/go-multierror v1.1.1
	github.com/hashicorp/go-version v1.3.0
	github.com/hashicorp/golang-lru v0.5.4
	github.com/heroku/docker-registry-client v0.0.0
	github.com/itchyny/gojq v0.12.1
	github.com/joelanford/helm-operator v0.0.7
	github.com/jstemmer/go-junit-report v0.9.1
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0 // indirect
	github.com/machinebox/graphql v0.2.2
	github.com/magiconair/properties v1.8.1
	github.com/mailru/easyjson v0.7.6
	github.com/mattn/goveralls v0.0.2
	github.com/mauricelam/genny v0.0.0-20190320071652-0800202903e5
	github.com/mitchellh/go-wordwrap v1.0.0
	github.com/mitchellh/hashstructure v1.0.0
	github.com/nilslice/protolock v0.0.0
	github.com/nxadm/tail v1.4.8
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.0.2-0.20190823105129-775207bd45b6
	github.com/openshift/api v3.9.1-0.20191201231411-9f834e337466+incompatible
	github.com/openshift/client-go v3.9.0+incompatible
	github.com/operator-framework/operator-lib v0.4.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.9.0
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.15.0
	github.com/russellhaering/gosaml2 v0.6.0
	github.com/russellhaering/goxmldsig v1.1.0
	github.com/sergi/go-diff v1.1.0
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	github.com/stackrox/anchore-client v0.0.0-20190929180200-981e05834836
	github.com/stackrox/default-authz-plugin v0.0.0-20190708153800-070801f52e6e
	github.com/stackrox/external-network-pusher v0.0.0-20201201000949-ec60e0486e7a
	github.com/stackrox/k8s-istio-cve-pusher v0.0.0-20191029220117-2a73008e51a9
	github.com/stackrox/scanner v0.0.0-20210513233124-463cefc804fa
	github.com/stretchr/testify v1.7.0
	github.com/tecbot/gorocksdb v0.0.0-20190705090504-162552197222
	github.com/tkuchiki/go-timezone v0.1.3
	github.com/vbauerster/mpb/v4 v4.12.2
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/yvasiyarov/go-metrics v0.0.0-20150112132944-c25f46c4b940 // indirect
	github.com/yvasiyarov/gorelic v0.0.7 // indirect
	github.com/yvasiyarov/newrelic_platform_go v0.0.0-20160601141957-9c099fbc30e9 // indirect
	go.etcd.io/bbolt v1.3.5
	go.uber.org/atomic v1.7.0
	go.uber.org/zap v1.15.1-0.20200717220000-53a387079b46
	golang.org/x/crypto v0.0.0-20210421170649-83a5a9bb288b
	golang.org/x/lint v0.0.0-20201208152925-83fdc39ff7b5 // indirect
	golang.org/x/net v0.0.0-20210525063256-abc453219eb5
	golang.org/x/oauth2 v0.0.0-20200902213428-5d25da1a8d43
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/sys v0.0.0-20210426230700-d19ff857e887
	golang.org/x/time v0.0.0-20210220033141-f8bda1e9f3ba
	golang.org/x/tools v0.1.0
	golang.stackrox.io/grpc-http1 v0.2.3
	gomodules.xyz/jsonpatch/v3 v3.0.1
	google.golang.org/api v0.33.0
	google.golang.org/genproto v0.0.0-20201110150050-8816d57aaa9a
	google.golang.org/grpc v1.33.2
	google.golang.org/grpc/examples v0.0.0-20200731180010-8bec2f5d898f
	gopkg.in/robfig/cron.v2 v2.0.0-20150107220207-be2e0b0deed5
	gopkg.in/square/go-jose.v2 v2.5.1
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
	gotest.tools v2.2.0+incompatible
	helm.sh/helm/v3 v3.5.4
	honnef.co/go/tools v0.0.1-2020.1.6
	k8s.io/api v0.20.4
	k8s.io/apiextensions-apiserver v0.20.4
	k8s.io/apimachinery v0.20.4
	k8s.io/apiserver v0.20.4
	k8s.io/cli-runtime v0.20.4
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/kube-openapi v0.0.0-20201113171705-d219536bb9fd
	k8s.io/kubectl v0.20.4
	k8s.io/kubelet v0.20.4
	k8s.io/utils v0.0.0-20210111153108-fddb29f9d009
	sigs.k8s.io/controller-runtime v0.8.3
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/PagerDuty/go-pagerduty => github.com/stackrox/go-pagerduty v0.0.0-20191021101800-15cb77365cca
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
	github.com/joelanford/helm-operator => github.com/stackrox/helm-operator v0.0.8-0.20210525092525-88508a237f9f
	github.com/mattn/goveralls => github.com/viswajithiii/goveralls v0.0.3-0.20190917224517-4dd02c532775
	github.com/nilslice/protolock => github.com/viswajithiii/protolock v0.10.1-0.20190117180626-43bb8a9ba4e8
	github.com/opencontainers/runc => github.com/opencontainers/runc v1.0.0-rc9
	github.com/openshift/client-go => github.com/openshift/client-go v0.0.0-20200623090625-83993cebb5ae
	github.com/tecbot/gorocksdb => github.com/DataDog/gorocksdb v0.0.0-20200107201226-9722c3a2e063
	go.uber.org/zap => github.com/stackrox/zap v1.15.1-0.20200720133746-810fd602fd0f
	golang.org/x/oauth2 => github.com/misberner/oauth2 v0.0.0-20200208204620-d153c71f6b8d

	honnef.co/go/tools => honnef.co/go/tools v0.0.1-2020.1.5
	k8s.io/api => k8s.io/api v0.20.4

	// Circular github.com/stackrox/rox sets this to an incompatible version
	k8s.io/client-go => k8s.io/client-go v0.20.4
	vbom.ml/util => github.com/fvbommel/util v0.0.0-20200828041400-c69461e88a36
)
