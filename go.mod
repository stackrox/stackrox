module github.com/stackrox/rox

go 1.15

require (
	cloud.google.com/go v0.62.0
	cloud.google.com/go/storage v1.10.0
	github.com/BurntSushi/toml v0.3.1
	github.com/Masterminds/semver v1.5.0
	github.com/Masterminds/sprig/v3 v3.1.0
	github.com/NYTimes/gziphandler v1.1.1
	github.com/PagerDuty/go-pagerduty v0.0.0-20191002190746-f60f4fc45222
	github.com/RoaringBitmap/roaring v0.4.21
	github.com/VividCortex/ewma v1.1.1
	github.com/andygrunwald/go-jira v1.10.0
	github.com/antihax/optional v0.0.0-20180407024304-ca021399b1a6
	github.com/aws/aws-sdk-go v1.29.17
	github.com/blevesearch/bleve v0.8.0
	github.com/cenkalti/backoff/v3 v3.2.2
	github.com/cloudflare/cfssl v0.0.0-20190510060611-9c027c93ba9e
	github.com/containers/image/v5 v5.5.2
	github.com/coreos/go-oidc v2.1.0+incompatible
	github.com/coreos/go-systemd/v22 v22.0.0
	github.com/dave/jennifer v1.3.0
	github.com/deckarep/golang-set v1.7.1
	github.com/dgraph-io/badger v0.0.0-20190131175406-28ef9bfd2438
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker v1.4.2-0.20200203170920-46ec8731fbce
	github.com/docker/go-connections v0.4.0
	github.com/docker/go-units v0.4.0
	github.com/facebookincubator/nvdtools v0.1.4-0.20191024132624-1cb041402875
	github.com/fullsailor/pkcs7 v0.0.0
	github.com/ghodss/yaml v1.0.0
	github.com/gobuffalo/packd v0.3.0
	github.com/gobuffalo/packr v1.30.1
	github.com/godbus/dbus/v5 v5.0.3
	github.com/gogo/protobuf v1.3.1
	github.com/golang/mock v1.4.3
	github.com/golang/protobuf v1.4.2
	github.com/golangci/golangci-lint v1.27.1-0.20200616100528-38d298c2c859
	github.com/google/certificate-transparency-go v1.0.21
	github.com/google/go-cmp v0.5.1
	github.com/googleapis/gnostic v0.5.1
	github.com/gorilla/schema v1.1.0
	github.com/graph-gophers/graphql-go v0.0.0-20190513003547-158e7b876106
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.0
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/grpc-ecosystem/grpc-gateway v1.11.4-0.20191004150533-c677e419aa5c
	github.com/hako/durafmt v0.0.0-20180520121703-7b7ae1e72ead
	github.com/hashicorp/go-multierror v1.0.0
	github.com/hashicorp/go-version v1.2.0
	github.com/hashicorp/golang-lru v0.5.3
	github.com/heroku/docker-registry-client v0.0.0
	github.com/jstemmer/go-junit-report v0.9.1
	github.com/machinebox/graphql v0.2.2
	github.com/magiconair/properties v1.8.1
	github.com/mailru/easyjson v0.7.6
	github.com/mattn/goveralls v0.0.2
	github.com/mauricelam/genny v0.0.0-20190320071652-0800202903e5
	github.com/mitchellh/go-wordwrap v1.0.0
	github.com/mitchellh/hashstructure v1.0.0
	github.com/nilslice/protolock v0.0.0
	github.com/opencontainers/go-digest v1.0.0
	github.com/openshift/api v3.9.1-0.20191201231411-9f834e337466+incompatible
	github.com/openshift/client-go v3.9.0+incompatible
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.7.1
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.10.0
	github.com/russellhaering/gosaml2 v0.3.1
	github.com/russellhaering/goxmldsig v0.0.0-20180430223755-7acd5e4a6ef7
	github.com/satori/go.uuid v1.2.0
	github.com/spf13/cobra v1.0.0
	github.com/spf13/pflag v1.0.5
	github.com/stackrox/anchore-client v0.0.0-20190929180200-981e05834836
	github.com/stackrox/default-authz-plugin v0.0.0-20190708153800-070801f52e6e
	github.com/stackrox/k8s-istio-cve-pusher v0.0.0-20191029220117-2a73008e51a9
	github.com/stackrox/scanner v0.0.0-20200930193229-9be8257b3580
	github.com/stretchr/testify v1.6.1
	github.com/tecbot/gorocksdb v0.0.0-20190705090504-162552197222
	github.com/tkuchiki/go-timezone v0.1.3
	github.com/vbauerster/mpb/v4 v4.11.1
	go.etcd.io/bbolt v1.3.5
	go.uber.org/atomic v1.6.0
	go.uber.org/zap v1.15.1-0.20200717220000-53a387079b46
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9
	golang.org/x/net v0.0.0-20200707034311-ab3426394381
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/sync v0.0.0-20200625203802-6e8e738ad208
	golang.org/x/sys v0.0.0-20200803210538-64077c9b5642
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0
	golang.org/x/tools v0.0.0-20200804011535-6c149bb5ef0d
	golang.stackrox.io/grpc-http1 v0.2.0
	google.golang.org/api v0.30.0
	google.golang.org/genproto v0.0.0-20200804131852-c06518451d9c
	google.golang.org/grpc v1.31.0
	gopkg.in/robfig/cron.v2 v2.0.0-20150107220207-be2e0b0deed5
	gopkg.in/square/go-jose.v2 v2.3.1
	gopkg.in/yaml.v2 v2.3.0
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776
	gotest.tools v2.2.0+incompatible
	helm.sh/helm/v3 v3.3.0
	honnef.co/go/tools v0.0.1-2020.1.5
	k8s.io/api v0.19.0
	k8s.io/apimachinery v0.19.0
	k8s.io/apiserver v0.19.0
	k8s.io/client-go v11.0.0+incompatible
	k8s.io/kubectl v0.19.0
	k8s.io/utils v0.0.0-20200821003339-5e75c0163111
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/PagerDuty/go-pagerduty => github.com/stackrox/go-pagerduty v0.0.0-20191021101800-15cb77365cca
	github.com/blevesearch/bleve => github.com/stackrox/bleve v0.0.0-20200807170555-6c4fa9f5e726
	github.com/couchbase/ghistogram => github.com/couchbase/ghistogram v0.0.1-0.20170308220240-d910dd063dd6
	github.com/couchbase/vellum => github.com/couchbase/vellum v0.0.0-20190829182332-ef2e028c01fd
	github.com/dgraph-io/badger => github.com/stackrox/badger v1.6.1-0.20200807170638-4177b4beb2ed
	github.com/facebookincubator/nvdtools => github.com/stackrox/nvdtools v0.0.0-20200903060121-ccc2b5ea9f6f
	github.com/fullsailor/pkcs7 => github.com/misberner/pkcs7 v0.0.0-20190417093538-a48bf0f78dea
	github.com/go-resty/resty => gopkg.in/resty.v1 v1.11.0
	github.com/gogo/protobuf => github.com/connorgorman/protobuf v1.2.2-0.20200827223713-3c42fc2eb426

	// Something pulls in an older version with uppercase OpenAPIv2 package version
	github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.5.1
	github.com/heroku/docker-registry-client => github.com/stackrox/docker-registry-client v0.0.0-20200930173048-36c5a823baf5
	github.com/mattn/goveralls => github.com/viswajithiii/goveralls v0.0.3-0.20190917224517-4dd02c532775
	github.com/nilslice/protolock => github.com/viswajithiii/protolock v0.10.1-0.20190117180626-43bb8a9ba4e8
	github.com/opencontainers/runc => github.com/opencontainers/runc v1.0.0-rc9
	github.com/openshift/client-go => github.com/openshift/client-go v0.0.0-20200623090625-83993cebb5ae
	github.com/tecbot/gorocksdb => github.com/DataDog/gorocksdb v0.0.0-20200107201226-9722c3a2e063
	go.uber.org/zap => github.com/stackrox/zap v1.15.1-0.20200720133746-810fd602fd0f
	golang.org/x/oauth2 => github.com/misberner/oauth2 v0.0.0-20200208204620-d153c71f6b8d

	// Helm needs k8s 0.19.0 deps in order to not screw everything up
	helm.sh/helm/v3 => github.com/misberner/helm/v3 v3.3.1-0.20200828132258-c3bfeb777bfb
	honnef.co/go/tools => honnef.co/go/tools v0.0.1-2020.1.5

	// Circular github.com/stackrox/rox sets this to an incompatible version
	k8s.io/client-go => k8s.io/client-go v0.19.0
	vbom.ml/util => github.com/fvbommel/util v0.0.0-20200828041400-c69461e88a36
)
