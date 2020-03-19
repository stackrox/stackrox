module github.com/stackrox/rox

go 1.14

require (
	cloud.google.com/go v0.47.0
	github.com/BurntSushi/toml v0.3.1
	github.com/NYTimes/gziphandler v1.1.1
	github.com/PagerDuty/go-pagerduty v0.0.0-20191002190746-f60f4fc45222
	github.com/RoaringBitmap/roaring v0.4.21
	github.com/VividCortex/ewma v1.1.1
	github.com/andygrunwald/go-jira v1.10.0
	github.com/antihax/optional v0.0.0-20180407024304-ca021399b1a6
	github.com/aws/aws-sdk-go v1.19.37
	github.com/blevesearch/bleve v0.8.0
	github.com/cenkalti/backoff/v3 v3.2.2
	github.com/cloudflare/cfssl v0.0.0-20190510060611-9c027c93ba9e
	github.com/containerd/continuity v0.0.0-20200101070350-669de920ecb0 // indirect
	github.com/containers/image/v5 v5.1.0
	github.com/coreos/go-oidc v2.1.0+incompatible
	github.com/coreos/go-systemd/v22 v22.0.0
	github.com/dave/jennifer v1.3.0
	github.com/deckarep/golang-set v1.7.1
	github.com/dgraph-io/badger v0.0.0-20190131175406-28ef9bfd2438
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker v0.7.3-0.20190327010347-be7ac8be2ae0
	github.com/docker/go-connections v0.4.0
	github.com/docker/go-units v0.4.0
	github.com/etcd-io/bbolt v1.3.3
	github.com/facebookincubator/nvdtools v0.1.4-0.20191024132624-1cb041402875
	github.com/fullsailor/pkcs7 v0.0.0
	github.com/ghodss/yaml v1.0.0
	github.com/gobuffalo/packd v0.3.0
	github.com/gobuffalo/packr v1.30.1
	github.com/godbus/dbus/v5 v5.0.3
	github.com/gogo/protobuf v1.2.2-0.20190723190241-65acae22fc9d
	github.com/golang/mock v1.3.1
	github.com/golang/protobuf v1.3.2
	github.com/google/certificate-transparency-go v1.0.21
	github.com/googleapis/gnostic v0.2.0
	github.com/gorilla/schema v1.1.0
	github.com/graph-gophers/graphql-go v0.0.0-20190513003547-158e7b876106
	github.com/grpc-ecosystem/go-grpc-middleware v1.1.0
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/grpc-ecosystem/grpc-gateway v1.11.4-0.20191004150533-c677e419aa5c
	github.com/hako/durafmt v0.0.0-20180520121703-7b7ae1e72ead
	github.com/hashicorp/go-version v1.2.0
	github.com/hashicorp/golang-lru v0.5.3
	github.com/heroku/docker-registry-client v0.0.0
	github.com/jstemmer/go-junit-report v0.9.1
	github.com/machinebox/graphql v0.2.2
	github.com/mailru/easyjson v0.0.0-20190614124828-94de47d64c63
	github.com/matryer/is v1.3.0 // indirect
	github.com/mattn/goveralls v0.0.2
	github.com/mauricelam/genny v0.0.0-20190320071652-0800202903e5
	github.com/mitchellh/go-wordwrap v1.0.0
	github.com/mitchellh/hashstructure v1.0.0
	github.com/nilslice/protolock v0.0.0
	github.com/opencontainers/go-digest v1.0.0-rc1
	github.com/openshift/api v3.9.1-0.20191201231411-9f834e337466+incompatible
	github.com/openshift/client-go v3.9.0+incompatible
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v0.9.2
	github.com/prometheus/client_model v0.0.0-20190129233127-fd36f4220a90
	github.com/prometheus/common v0.4.1
	github.com/russellhaering/gosaml2 v0.3.1
	github.com/russellhaering/goxmldsig v0.0.0-20180430223755-7acd5e4a6ef7
	github.com/satori/go.uuid v1.2.0
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.3
	github.com/stackrox/anchore-client v0.0.0-20190929180200-981e05834836
	github.com/stackrox/default-authz-plugin v0.0.0-20190708153800-070801f52e6e
	github.com/stackrox/k8s-istio-cve-pusher v0.0.0-20191029220117-2a73008e51a9
	github.com/stackrox/scanner v0.0.0-20191202203519-a2a15f33f41a
	github.com/stretchr/testify v1.4.0
	github.com/tkuchiki/go-timezone v0.1.3
	github.com/vbauerster/mpb/v4 v4.11.1
	golang.org/x/crypto v0.0.0-20200220183623-bac4c82f6975
	golang.org/x/lint v0.0.0-20190930215403-16217165b5de
	golang.org/x/net v0.0.0-20191014212845-da9a3fd4c582
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45
	golang.org/x/sync v0.0.0-20190423024810-112230192c58
	golang.org/x/sys v0.0.0-20191127021746-63cb32ae39b2
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4
	golang.org/x/tools v0.0.0-20191130070609-6e064ea0cf2d
	google.golang.org/api v0.11.0
	google.golang.org/genproto v0.0.0-20191009194640-548a555dbc03
	google.golang.org/grpc v1.24.0
	gopkg.in/robfig/cron.v2 v2.0.0-20150107220207-be2e0b0deed5
	gopkg.in/square/go-jose.v2 v2.3.1
	gopkg.in/yaml.v2 v2.2.4
	honnef.co/go/tools v0.0.1-2019.2.3
	k8s.io/api v0.0.0-20191114100352-16d7abae0d2a
	k8s.io/apimachinery v0.0.0-20191028221656-72ed19daf4bb
	k8s.io/apiserver v0.0.0-20191114103151-9ca1dc586682
	k8s.io/client-go v7.0.0+incompatible
	k8s.io/helm v2.14.0+incompatible
	k8s.io/kubectl v0.0.0-20191114113550-6123e1c827f7
	sigs.k8s.io/yaml v1.1.0
)

replace (
	github.com/PagerDuty/go-pagerduty => github.com/stackrox/go-pagerduty v0.0.0-20191021101800-15cb77365cca
	github.com/blevesearch/bleve => github.com/stackrox/bleve v0.0.0-20200303185535-b6fb59df2573
	github.com/couchbase/ghistogram => github.com/couchbase/ghistogram v0.0.1-0.20170308220240-d910dd063dd6
	github.com/couchbase/vellum => github.com/couchbase/vellum v0.0.0-20190829182332-ef2e028c01fd
	github.com/dgraph-io/badger => github.com/stackrox/badger v1.6.1-0.20191025195058-f2b50b9f079c
	github.com/facebookincubator/nvdtools => github.com/stackrox/nvdtools v0.0.0-20191120225537-fe4e9a7e467f
	github.com/fullsailor/pkcs7 => github.com/misberner/pkcs7 v0.0.0-20190417093538-a48bf0f78dea
	github.com/go-resty/resty => gopkg.in/resty.v1 v1.11.0
	github.com/gogo/protobuf => github.com/connorgorman/protobuf v1.2.2-0.20190220010025-a81e5c3a5053
	github.com/heroku/docker-registry-client => github.com/stackrox/docker-registry-client v0.0.0-20181115184320-3d98b2b79d1b
	github.com/mattn/goveralls => github.com/viswajithiii/goveralls v0.0.3-0.20190917224517-4dd02c532775
	github.com/nilslice/protolock => github.com/viswajithiii/protolock v0.10.1-0.20190117180626-43bb8a9ba4e8
	github.com/opencontainers/runc => github.com/opencontainers/runc v1.0.0-rc9
	github.com/openshift/client-go => github.com/openshift/client-go v0.0.0-20190923092832-6afefc9bb372
	golang.org/x/oauth2 => github.com/misberner/oauth2 v0.0.0-20200208204620-d153c71f6b8d

	honnef.co/go/tools => honnef.co/go/tools v0.0.0-20200207073116-104ba5a2aac5

	k8s.io/api => k8s.io/api v0.0.0-20191114100352-16d7abae0d2a
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20191114105449-027877536833
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20191028221656-72ed19daf4bb
	k8s.io/apiserver => k8s.io/apiserver v0.0.0-20191114103151-9ca1dc586682
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20191114110141-0a35778df828
	k8s.io/client-go => k8s.io/client-go v0.0.0-20191114101535-6c5935290e33
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.0.0-20191114112024-4bbba8331835
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.0.0-20191114111741-81bb9acf592d
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20191004115455-8e001e5d1894
	k8s.io/component-base => k8s.io/component-base v0.0.0-20191114102325-35a9586014f7
	k8s.io/cri-api => k8s.io/cri-api v0.0.0-20190828162817-608eb1dad4ac
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.0.0-20191114112310-0da609c4ca2d
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.0.0-20191114103820-f023614fb9ea
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.0.0-20191114111510-6d1ed697a64b
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.0.0-20191114110717-50a77e50d7d9
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.0.0-20191114111229-2e90afcb56c7
	k8s.io/kubectl => k8s.io/kubectl v0.0.0-20191114113550-6123e1c827f7
	k8s.io/kubelet => k8s.io/kubelet v0.0.0-20191114110954-d67a8e7e2200
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.0.0-20191114112655-db9be3e678bb
	k8s.io/metrics => k8s.io/metrics v0.0.0-20191114105837-a4a2842dc51b
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.0.0-20191114104439-68caf20693ac
)
