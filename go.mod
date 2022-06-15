module github.com/stackrox/rox

go 1.16

// CAVEAT: This introduces a circular dependency. If you change this line, you MUST change the "exclude"
// directive at the bottom of the file as well.
require github.com/stackrox/scanner v0.0.0-20220426001230-9ab6777c9581

require (
	cloud.google.com/go/compute v1.6.1
	cloud.google.com/go/containeranalysis v0.3.0
	cloud.google.com/go/storage v1.22.1
	github.com/BurntSushi/toml v1.0.0
	github.com/Masterminds/semver v1.5.0
	github.com/Masterminds/sprig/v3 v3.2.2
	github.com/NYTimes/gziphandler v1.1.1
	github.com/PagerDuty/go-pagerduty v1.5.1
	github.com/RoaringBitmap/roaring v1.1.0
	github.com/VividCortex/ewma v1.2.0
	github.com/andygrunwald/go-jira v1.15.1
	github.com/aws/aws-sdk-go v1.44.29
	github.com/blevesearch/bleve v1.0.14
	github.com/cenkalti/backoff/v3 v3.2.2
	github.com/ckaznocha/protoc-gen-lint v0.2.4
	github.com/cloudflare/cfssl v1.6.1
	github.com/containers/image/v5 v5.20.0
	github.com/coreos/go-oidc/v3 v3.2.0
	github.com/coreos/go-systemd/v22 v22.3.2
	github.com/couchbase/moss v0.1.0 // indirect
	github.com/dave/jennifer v1.5.0
	github.com/deckarep/golang-set v1.8.0
	github.com/dexidp/dex v0.0.0-20220607113954-3836196af2e7
	github.com/docker/distribution v2.8.1+incompatible
	github.com/docker/docker v20.10.17+incompatible
	github.com/docker/go-connections v0.4.0
	github.com/docker/go-units v0.4.0
	github.com/facebookincubator/nvdtools v0.1.4
	github.com/fatih/color v1.13.0
	github.com/fullsailor/pkcs7 v0.0.0
	github.com/ghodss/yaml v1.0.0
	github.com/go-logr/logr v1.2.2
	github.com/godbus/dbus/v5 v5.1.0
	github.com/gofrs/uuid v4.2.0+incompatible
	github.com/gogo/protobuf v1.3.2
	github.com/golang-jwt/jwt/v4 v4.4.1
	github.com/golang/mock v1.6.0
	github.com/golang/protobuf v1.5.2
	github.com/google/certificate-transparency-go v1.1.3
	github.com/google/go-cmp v0.5.8
	github.com/google/go-containerregistry v0.9.0
	github.com/googleapis/gnostic v0.5.5
	github.com/gorilla/schema v1.2.0
	github.com/graph-gophers/graphql-go v1.3.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/hashicorp/go-multierror v1.1.1
	github.com/hashicorp/go-version v1.5.0
	github.com/hashicorp/golang-lru v0.5.4
	github.com/heroku/docker-registry-client v0.0.0
	github.com/jackc/pgtype v1.11.0
	github.com/jackc/pgx/v4 v4.16.1
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/joshdk/go-junit v0.0.0-20210226021600-6145f504ca0d
	github.com/kisielk/sqlstruct v0.0.0-20210630145711-dae28ed37023 // indirect
	github.com/lib/pq v1.10.6
	github.com/machinebox/graphql v0.2.2
	github.com/magiconair/properties v1.8.6
	github.com/mailru/easyjson v0.7.7
	github.com/mauricelam/genny v0.0.0-20190320071652-0800202903e5
	github.com/mitchellh/go-wordwrap v1.0.1
	github.com/mitchellh/hashstructure v1.1.0
	github.com/moby/sys/mount v0.3.0 // indirect
	github.com/nxadm/tail v1.4.8
	github.com/olekukonko/tablewriter v0.0.5
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.0.3-0.20220114050600-8b9d41f48198
	github.com/openshift/api v3.9.1-0.20191201231411-9f834e337466+incompatible
	github.com/openshift/client-go v0.0.0-20200623090625-83993cebb5ae
	github.com/operator-framework/helm-operator-plugins v0.0.7
	github.com/operator-framework/operator-sdk v0.19.4
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.12.2
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.34.0
	github.com/russellhaering/gosaml2 v0.7.0
	github.com/russellhaering/goxmldsig v1.2.0
	github.com/sergi/go-diff v1.2.0
	github.com/sigstore/cosign v1.8.1-0.20220530190726-3a43ddc93914
	github.com/sigstore/sigstore v1.2.1-0.20220528141235-6d98e7d59dee
	github.com/spf13/cobra v1.4.0
	github.com/spf13/pflag v1.0.5
	github.com/stackrox/default-authz-plugin v0.0.0-20210608105219-00ad9c9f3855
	github.com/stackrox/external-network-pusher v0.0.0-20210419192707-074af92bbfa7
	github.com/stackrox/helmtest v0.0.0-20220118100812-1ad97c4de347
	github.com/stackrox/k8s-istio-cve-pusher v0.0.0-20210422200002-d89f671ac4f5
	github.com/stretchr/testify v1.7.2
	github.com/tecbot/gorocksdb v0.0.0-20191217155057-f0fad39f321c
	github.com/tidwall/gjson v1.14.1
	github.com/tkuchiki/go-timezone v0.2.2
	github.com/vbauerster/mpb/v4 v4.12.2
	go.etcd.io/bbolt v1.3.6
	go.uber.org/atomic v1.9.0
	go.uber.org/zap v1.21.0
	golang.org/x/crypto v0.0.0-20220411220226-7b82a4e95df4
	golang.org/x/net v0.0.0-20220607020251-c690dde0001d
	golang.org/x/oauth2 v0.0.0-20220524215830-622c5d57e401
	golang.org/x/sync v0.0.0-20220601150217-0de741cfad7f
	golang.org/x/sys v0.0.0-20220520151302-bc2c85ada10a
	golang.org/x/time v0.0.0-20220411224347-583f2d630306
	golang.org/x/tools v0.1.10
	golang.stackrox.io/grpc-http1 v0.2.4
	google.golang.org/api v0.83.0
	google.golang.org/genproto v0.0.0-20220602131408-e326c6e8e9c8
	google.golang.org/grpc v1.47.0
	google.golang.org/grpc/examples v0.0.0-20210902184326-c93e472777b9
	gopkg.in/robfig/cron.v2 v2.0.0-20150107220207-be2e0b0deed5
	gopkg.in/square/go-jose.v2 v2.6.0
	gopkg.in/yaml.v3 v3.0.1
	gorm.io/driver/postgres v1.3.5
	gorm.io/gorm v1.23.6
	gotest.tools v2.2.0+incompatible
	helm.sh/helm/v3 v3.7.2
	k8s.io/api v0.23.7
	k8s.io/apimachinery v0.23.7
	k8s.io/apiserver v0.23.4
	k8s.io/client-go v0.23.5
	k8s.io/kubectl v0.23.1
	k8s.io/kubelet v0.22.9
	k8s.io/utils v0.0.0-20220210201930-3a6ce19ff2f9
	sigs.k8s.io/controller-runtime v0.11.0
	sigs.k8s.io/yaml v1.3.0
)

replace (
	github.com/blevesearch/bleve => github.com/stackrox/bleve v0.0.0-20200807170555-6c4fa9f5e726

	github.com/facebookincubator/nvdtools => github.com/stackrox/nvdtools v0.0.0-20210326191554-5daeb6395b56
	github.com/fullsailor/pkcs7 => github.com/misberner/pkcs7 v0.0.0-20190417093538-a48bf0f78dea
	github.com/gogo/protobuf => github.com/connorgorman/protobuf v1.2.2-0.20210115205927-b892c1b298f7

	github.com/heroku/docker-registry-client => github.com/stackrox/docker-registry-client v0.0.0-20220204234128-07f109db0819

	// github.com/mikefarah/yaml/v2 is a clone of github.com/go-yaml/yaml/v2.
	// Both github.com/go-yaml/yaml/v2 and github.com/go-yaml/yaml/v3 do not provide go.sum
	// so dependabot is not able to check dependecies.
	// See https://github.com/go-yaml/yaml/issues/772
	// Therefore we point all to our fork of `go-yaml` - github.com/stackrox/yaml/v2|v3
	// where we provide the actual `go.sum`.
	github.com/mikefarah/yaml/v2 => gopkg.in/yaml.v2 v2.4.0

	github.com/nxadm/tail => github.com/stackrox/tail v1.4.9-0.20210831224919-407035634f5d
	github.com/opencontainers/runc => github.com/opencontainers/runc v1.0.0-rc9
	github.com/operator-framework/helm-operator-plugins => github.com/stackrox/helm-operator v0.0.8-0.20220506091602-3764c49abfb3
	// github.com/sigstore/rekor is a transitive dep pulled in by cosign. The version pulled in by cosign is using
	// a vulnerable go-tuf version
	// (https://github.com/theupdateframework/go-tuf/security/advisories/GHSA-66x3-6cw3-v5gj).
	// An upstream patch within rekor bumps this dep, once the upstream patch of rekor has landed within cosign, we can remove this
	// replace redirective.
	github.com/sigstore/rekor => github.com/sigstore/rekor v0.7.1-0.20220531123351-0c1de2a6239c
	// sigstore/sigstore is used as a dependency within cosign and rekor. The version pulled in by cosign is using
	// a vulnerable go-tuf version
	// (https://github.com/theupdateframework/go-tuf/security/advisories/GHSA-66x3-6cw3-v5gj).
	// Once the upstream patches and release has landed, we can remove this replace directive.
	github.com/sigstore/sigstore => github.com/sigstore/sigstore v1.2.1-0.20220528141235-6d98e7d59dee
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
	golang.org/x/oauth2 => github.com/stackrox/oauth2 v0.0.0-20220531064142-8b312376cb4c

	gopkg.in/yaml.v2 => github.com/stackrox/yaml/v2 v2.4.1
	gopkg.in/yaml.v3 => github.com/stackrox/yaml/v3 v3.0.0
)

// Circular rox -> scanner -> rox dependency would pull in a long list of past rox versions, cluttering go.sum
// and the module cache.
// If you upgrade the scanner version, you MUST change this line as well to refer to the rox version included
// from the given scanner version.
exclude github.com/stackrox/rox v0.0.0-20210914215712-9ac265932e28

exclude k8s.io/client-go v12.0.0+incompatible

exclude github.com/openshift/client-go v3.9.0+incompatible
