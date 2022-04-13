module github.com/stackrox/rox

go 1.16

// CAVEAT: This introduces a circular dependency. If you change this line, you MUST change the "exclude"
// directive at the bottom of the file as well.
require github.com/stackrox/scanner v0.0.0-20220408151911-460993206ee4

require (
	cloud.google.com/go/compute v1.5.0
	cloud.google.com/go/containeranalysis v0.1.1
	cloud.google.com/go/storage v1.19.0
	github.com/BurntSushi/toml v1.0.0
	github.com/Masterminds/semver v1.5.0
	github.com/Masterminds/sprig/v3 v3.2.2
	github.com/NYTimes/gziphandler v1.1.1
	github.com/PagerDuty/go-pagerduty v1.4.2
	github.com/RoaringBitmap/roaring v0.9.4
	github.com/VividCortex/ewma v1.2.0
	github.com/andygrunwald/go-jira v1.15.1
	github.com/antihax/optional v1.0.0
	github.com/aws/aws-sdk-go v1.42.43
	github.com/blevesearch/bleve v1.0.14
	github.com/blevesearch/blevex v1.0.0 // indirect
	github.com/blevesearch/go-porterstemmer v1.0.3 // indirect
	github.com/blevesearch/segment v0.9.0 // indirect
	github.com/cenkalti/backoff/v3 v3.2.2
	github.com/ckaznocha/protoc-gen-lint v0.2.4
	github.com/cloudflare/cfssl v0.0.0-20190510060611-9c027c93ba9e
	github.com/containers/image/v5 v5.20.0
	github.com/coreos/go-oidc/v3 v3.1.0
	github.com/coreos/go-systemd/v22 v22.3.2
	github.com/couchbase/moss v0.1.0 // indirect
	github.com/couchbase/vellum v1.0.2 // indirect
	github.com/dave/jennifer v1.5.0
	github.com/deckarep/golang-set v1.8.0
	github.com/dexidp/dex v0.0.0-20210917061239-f0186ff2651e
	github.com/docker/distribution v2.8.1+incompatible
	github.com/docker/docker v20.10.13+incompatible
	github.com/docker/go-connections v0.4.0
	github.com/docker/go-units v0.4.0
	github.com/facebookincubator/flog v0.0.0-20190930132826-d2511d0ce33c // indirect
	github.com/facebookincubator/nvdtools v0.1.4
	github.com/fatih/color v1.13.0
	github.com/fullsailor/pkcs7 v0.0.0
	github.com/ghodss/yaml v1.0.0
	github.com/go-logr/logr v0.4.0
	github.com/godbus/dbus/v5 v5.0.4
	github.com/gofrs/uuid v4.2.0+incompatible
	github.com/gogo/protobuf v1.3.2
	github.com/golang-jwt/jwt/v4 v4.4.1
	github.com/golang/mock v1.6.0
	github.com/golang/protobuf v1.5.2
	github.com/google/certificate-transparency-go v1.1.2
	github.com/google/go-cmp v0.5.7
	github.com/google/go-containerregistry v0.8.1-0.20220125170349-50dfc2733d10
	github.com/googleapis/gnostic v0.5.5
	github.com/gorilla/schema v1.2.0
	github.com/graph-gophers/graphql-go v1.3.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/hako/durafmt v0.0.0-20210608085754-5c1018a4e16b
	github.com/hashicorp/go-multierror v1.1.1
	github.com/hashicorp/go-version v1.4.0
	github.com/hashicorp/golang-lru v0.5.4
	github.com/heroku/docker-registry-client v0.0.0
	github.com/jackc/pgtype v1.10.0
	github.com/jackc/pgx/v4 v4.15.0
	github.com/joelanford/helm-operator v0.0.7
	github.com/joshdk/go-junit v0.0.0-20210226021600-6145f504ca0d
	github.com/kisielk/sqlstruct v0.0.0-20210630145711-dae28ed37023 // indirect
	github.com/machinebox/graphql v0.2.2
	github.com/magiconair/properties v1.8.5
	github.com/mailru/easyjson v0.7.7
	github.com/matryer/is v1.4.0 // indirect
	github.com/mauricelam/genny v0.0.0-20190320071652-0800202903e5
	github.com/mitchellh/go-wordwrap v1.0.1
	github.com/mitchellh/hashstructure v1.1.0
	github.com/moby/sys/mount v0.3.0 // indirect
	github.com/nxadm/tail v1.4.8
	github.com/olekukonko/tablewriter v0.0.5
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.0.3-0.20211202193544-a5463b7f9c84
	github.com/openshift/api v3.9.1-0.20191201231411-9f834e337466+incompatible
	github.com/openshift/client-go v0.0.0-20200623090625-83993cebb5ae
	github.com/operator-framework/operator-sdk v0.19.4
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.12.1
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.32.1
	github.com/russellhaering/gosaml2 v0.7.0
	github.com/russellhaering/goxmldsig v1.2.0
	github.com/sergi/go-diff v1.2.0
	github.com/sigstore/cosign v1.5.1
	github.com/sigstore/sigstore v1.1.1-0.20220130134424-bae9b66b8442
	github.com/spf13/cobra v1.4.0
	github.com/spf13/pflag v1.0.5
	github.com/stackrox/anchore-client v0.0.0-20190929180200-981e05834836
	github.com/stackrox/default-authz-plugin v0.0.0-20210608105219-00ad9c9f3855
	github.com/stackrox/external-network-pusher v0.0.0-20210419192707-074af92bbfa7
	github.com/stackrox/helmtest v0.0.0-20220118100812-1ad97c4de347
	github.com/stackrox/k8s-istio-cve-pusher v0.0.0-20210422200002-d89f671ac4f5
	github.com/steveyen/gtreap v0.1.0 // indirect
	github.com/stretchr/testify v1.7.1
	github.com/tecbot/gorocksdb v0.0.0-20191217155057-f0fad39f321c
	github.com/tidwall/gjson v1.14.0
	github.com/tkuchiki/go-timezone v0.2.2
	github.com/vbauerster/mpb/v4 v4.12.2
	go.etcd.io/bbolt v1.3.6
	go.uber.org/atomic v1.9.0
	go.uber.org/zap v1.20.0
	golang.org/x/crypto v0.0.0-20220112180741-5e0467b6c7ce
	golang.org/x/net v0.0.0-20220225172249-27dd8689420f
	golang.org/x/oauth2 v0.0.0-20220309155454-6242fa91716a
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/sys v0.0.0-20220310020820-b874c991c1a5
	golang.org/x/time v0.0.0-20211116232009-f0f3c7e86c11
	golang.org/x/tools v0.1.8
	golang.stackrox.io/grpc-http1 v0.2.4
	google.golang.org/api v0.73.0
	google.golang.org/genproto v0.0.0-20220310185008-1973136f34c6
	google.golang.org/grpc v1.45.0
	google.golang.org/grpc/examples v0.0.0-20210902184326-c93e472777b9
	gopkg.in/robfig/cron.v2 v2.0.0-20150107220207-be2e0b0deed5
	gopkg.in/square/go-jose.v2 v2.6.0
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
	gotest.tools v2.2.0+incompatible
	helm.sh/helm/v3 v3.7.1
	k8s.io/api v0.22.8
	k8s.io/apimachinery v0.22.8
	k8s.io/apiserver v0.22.5
	k8s.io/client-go v0.22.8
	k8s.io/kubectl v0.22.8
	k8s.io/kubelet v0.22.7
	k8s.io/utils v0.0.0-20211208161948-7d6a63dca704
	sigs.k8s.io/controller-runtime v0.10.3
	sigs.k8s.io/yaml v1.3.0
)

replace (
	github.com/blevesearch/bleve => github.com/stackrox/bleve v0.0.0-20200807170555-6c4fa9f5e726

	github.com/facebookincubator/nvdtools => github.com/stackrox/nvdtools v0.0.0-20210326191554-5daeb6395b56
	github.com/fullsailor/pkcs7 => github.com/misberner/pkcs7 v0.0.0-20190417093538-a48bf0f78dea
	github.com/gogo/protobuf => github.com/connorgorman/protobuf v1.2.2-0.20210115205927-b892c1b298f7

	// Both dependencies are requiring k8s.io/klog/v2@v2.40.1, which introduces a breaking change within
	// go-logr-logr dependency. For the time being, use older versions of the package.
	github.com/google/go-containerregistry/pkg/authn/k8schain v0.0.0-20220125170349-50dfc2733d10 => github.com/google/go-containerregistry/pkg/authn/k8schain v0.0.0-20211216152112-d1271fea6383
	github.com/google/go-containerregistry/pkg/authn/kubernetes v0.0.0-20220125170349-50dfc2733d10 => github.com/google/go-containerregistry/pkg/authn/k8schain v0.0.0-20211222182933-7c19fa370dbd
	github.com/heroku/docker-registry-client => github.com/stackrox/docker-registry-client v0.0.0-20220204234128-07f109db0819
	github.com/joelanford/helm-operator => github.com/stackrox/helm-operator v0.0.8-0.20211217081542-57dfe5d681e3
	github.com/mattn/goveralls => github.com/viswajithiii/goveralls v0.0.3-0.20190917224517-4dd02c532775

	// github.com/mikefarah/yaml/v2 is a clone of github.com/go-yaml/yaml/v2.
	// Both github.com/go-yaml/yaml/v2 and github.com/go-yaml/yaml/v3 do not provide go.sum
	// so dependabot is not able to check dependecies.
	// See https://github.com/go-yaml/yaml/issues/772
	// Therefore we point all to our fork of `go-yaml` - github.com/stackrox/yaml/v2|v3
	// where we provide the actual `go.sum`.
	github.com/mikefarah/yaml/v2 => gopkg.in/yaml.v2 v2.4.0

	github.com/nxadm/tail => github.com/stackrox/tail v1.4.9-0.20210831224919-407035634f5d
	github.com/opencontainers/runc => github.com/opencontainers/runc v1.0.0-rc9
	github.com/tecbot/gorocksdb => github.com/DataDog/gorocksdb v0.0.0-20200107201226-9722c3a2e063
	go.uber.org/zap => github.com/stackrox/zap v1.15.1-0.20200720133746-810fd602fd0f

	// Our fork has a change exposing a method to do generic POST requests
	// against the OAuth server in order to realize the refresh token flow.
	// The problem is that:
	//   (a) the oauth2 library doesn’t support token refresh out of the box;
	//   (b) authenticating with an OAuth server is super complicated because
	//       there is a mix of header auth and body auth in existence, which
	//       the library solves with autosensing + caching, and what we don't
	//       want to reimplement in our code.
	golang.org/x/oauth2 => github.com/misberner/oauth2 v0.0.0-20210904010302-0b4d90ae6a84

	gopkg.in/yaml.v2 => github.com/stackrox/yaml/v2 v2.4.1
	gopkg.in/yaml.v3 => github.com/stackrox/yaml/v3 v3.0.0

	// knative.dev/pkg upgraded from k8s.io/klog@v1.0.0 to k8s.io/klog/v2@v2.40.1, which introduces a breaking
	// change within the go-logr/logr dependency. For the time being, use older versions of the package.
	knative.dev/pkg v0.0.0-20220121092305-3ba5d72e310a => knative.dev/pkg v0.0.0-20220111134415-80b253f23023
)

// Circular rox -> scanner -> rox dependency would pull in a long list of past rox versions, cluttering go.sum
// and the module cache.
// If you upgrade the scanner version, you MUST change this line as well to refer to the rox version included
// from the given scanner version.
exclude github.com/stackrox/rox v0.0.0-20210914215712-9ac265932e28

exclude k8s.io/client-go v12.0.0+incompatible

exclude github.com/openshift/client-go v3.9.0+incompatible
