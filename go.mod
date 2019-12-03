module github.com/stackrox/rox

go 1.13

require (
	cloud.google.com/go v0.47.0
	github.com/BurntSushi/toml v0.3.1
	github.com/NYTimes/gziphandler v1.1.1
	github.com/PagerDuty/go-pagerduty v0.0.0-20191002190746-f60f4fc45222
	github.com/RoaringBitmap/roaring v0.4.17
	github.com/VividCortex/ewma v1.1.1
	github.com/andygrunwald/go-jira v1.10.0
	github.com/antihax/optional v0.0.0-20180407024304-ca021399b1a6
	github.com/aws/aws-sdk-go v1.19.37
	github.com/blevesearch/bleve v0.0.0
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/cloudflare/cfssl v0.0.0-20190510060611-9c027c93ba9e
	github.com/containers/image v3.0.2+incompatible
	github.com/coreos/go-oidc v2.0.0+incompatible
	github.com/coreos/go-systemd v0.0.0-20181031085051-9002847aa142
	github.com/dave/jennifer v1.3.0
	github.com/deckarep/golang-set v1.7.1
	github.com/dgraph-io/badger v0.0.0-20190131175406-28ef9bfd2438
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker v0.0.0-20170906102241-7c9e64a2e189
	github.com/docker/go-connections v0.4.0
	github.com/docker/go-units v0.4.0
	github.com/etcd-io/bbolt v1.3.3
	github.com/fullsailor/pkcs7 v0.0.0
	github.com/ghodss/yaml v1.0.0
	github.com/gobuffalo/packd v0.3.0
	github.com/gobuffalo/packr v1.30.1
	github.com/godbus/dbus v0.0.0-20181101234600-2ff6f7ffd60f
	github.com/gogo/protobuf v1.2.1
	github.com/golang/mock v1.3.1
	github.com/golang/protobuf v1.3.2
	github.com/google/certificate-transparency-go v1.0.21
	github.com/googleapis/gnostic v0.2.0
	github.com/graph-gophers/graphql-go v0.0.0-20190513003547-158e7b876106
	github.com/grpc-ecosystem/go-grpc-middleware v1.1.0
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/grpc-ecosystem/grpc-gateway v1.11.4-0.20191004150533-c677e419aa5c
	github.com/hako/durafmt v0.0.0-20180520121703-7b7ae1e72ead
	github.com/hashicorp/go-version v1.2.0
	github.com/hashicorp/golang-lru v0.5.3
	github.com/heroku/docker-registry-client v0.0.0
	github.com/jstemmer/go-junit-report v0.9.1
	github.com/mailru/easyjson v0.0.0-20180823135443-60711f1a8329
	github.com/mattn/goveralls v0.0.2
	github.com/mauricelam/genny v0.0.0-20190320071652-0800202903e5
	github.com/mitchellh/go-wordwrap v1.0.0
	github.com/mitchellh/hashstructure v1.0.0
	github.com/nilslice/protolock v0.0.0
	github.com/opencontainers/go-digest v1.0.0-rc1
	github.com/openshift/api v3.9.0+incompatible
	github.com/openshift/client-go v3.9.0+incompatible
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v0.9.1
	github.com/russellhaering/gosaml2 v0.3.1
	github.com/russellhaering/goxmldsig v0.0.0-20180430223755-7acd5e4a6ef7
	github.com/satori/go.uuid v1.1.0
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.3
	github.com/stackrox/anchore-client v0.0.0-20190929180200-981e05834836
	github.com/stackrox/default-authz-plugin v0.0.0-20190708153800-070801f52e6e
	github.com/stackrox/k8s-istio-cve-pusher v0.0.0-20191029220117-2a73008e51a9
	github.com/stackrox/scanner v0.0.0-20191202203519-a2a15f33f41a
	github.com/stretchr/testify v1.4.0
	github.com/tkuchiki/go-timezone v0.1.3
	github.com/vbauerster/mpb/v4 v4.9.0
	golang.org/x/crypto v0.0.0-20190701094942-4def268fd1a4
	golang.org/x/lint v0.0.0-20190930215403-16217165b5de
	golang.org/x/net v0.0.0-20191014212845-da9a3fd4c582
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45
	golang.org/x/sync v0.0.0-20190423024810-112230192c58
	golang.org/x/sys v0.0.0-20191010194322-b09406accb47
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4
	golang.org/x/tools v0.0.0-20191018203202-04252eccb9d5
	google.golang.org/api v0.11.0
	google.golang.org/genproto v0.0.0-20191009194640-548a555dbc03
	google.golang.org/grpc v1.24.0
	gopkg.in/robfig/cron.v2 v2.0.0-20150107220207-be2e0b0deed5
	gopkg.in/square/go-jose.v2 v2.3.1
	gopkg.in/yaml.v2 v2.2.3
	honnef.co/go/tools v0.0.1-2019.2.3
	k8s.io/api v0.0.0-20180308224125-73d903622b73
	k8s.io/apimachinery v0.0.0-20180228050457-302974c03f7e
	k8s.io/apiserver v0.0.0-20180327025904-5ae41ac86efd
	k8s.io/client-go v7.0.0+incompatible
	k8s.io/helm v2.14.0+incompatible
	k8s.io/kubernetes v1.14.2
)

replace (
	github.com/PagerDuty/go-pagerduty => github.com/stackrox/go-pagerduty v0.0.0-20191021101800-15cb77365cca
	github.com/blevesearch/bleve => github.com/stackrox/bleve v0.0.0-20190918030150-5ebdc2278ffe
	github.com/dgraph-io/badger => github.com/stackrox/badger v1.6.1-0.20191025195058-f2b50b9f079c
	github.com/facebookincubator/nvdtools => github.com/stackrox/nvdtools v0.0.0-20191120225537-fe4e9a7e467f
	github.com/fullsailor/pkcs7 => github.com/misberner/pkcs7 v0.0.0-20190417093538-a48bf0f78dea
	github.com/go-resty/resty => gopkg.in/resty.v1 v1.11.0
	github.com/gogo/protobuf => github.com/connorgorman/protobuf v1.2.2-0.20190220010025-a81e5c3a5053
	github.com/heroku/docker-registry-client => github.com/stackrox/docker-registry-client v0.0.0-20181115184320-3d98b2b79d1b
	github.com/mattn/goveralls => github.com/viswajithiii/goveralls v0.0.3-0.20190917224517-4dd02c532775
	github.com/nilslice/protolock => github.com/viswajithiii/protolock v0.10.1-0.20190117180626-43bb8a9ba4e8
)
