module github.com/stackrox/rox

go 1.17

// CAVEAT: This introduces a circular dependency. If you change this line, you MUST change the "exclude"
// directive at the bottom of the file as well.
require github.com/stackrox/scanner v0.0.0-20220426001230-9ab6777c9581

require (
	cloud.google.com/go/compute v1.9.0
	cloud.google.com/go/containeranalysis v0.4.0
	cloud.google.com/go/storage v1.26.0
	github.com/BurntSushi/toml v1.2.0
	github.com/Masterminds/semver v1.5.0
	github.com/Masterminds/sprig/v3 v3.2.2
	github.com/NYTimes/gziphandler v1.1.1
	github.com/PagerDuty/go-pagerduty v1.5.1
	github.com/RoaringBitmap/roaring v1.2.1
	github.com/VividCortex/ewma v1.2.0
	github.com/andygrunwald/go-jira v1.16.0
	github.com/aws/aws-sdk-go v1.44.86
	github.com/blevesearch/bleve v1.0.14
	github.com/cenkalti/backoff/v3 v3.2.2
	github.com/ckaznocha/protoc-gen-lint v0.2.4
	github.com/cloudflare/cfssl v1.6.2
	github.com/containers/image/v5 v5.20.0
	github.com/coreos/go-oidc/v3 v3.2.0
	github.com/coreos/go-systemd/v22 v22.3.2
	github.com/couchbase/moss v0.1.0 // indirect
	github.com/dave/jennifer v1.5.1
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
	github.com/golang-jwt/jwt/v4 v4.4.2
	github.com/golang/mock v1.6.0
	github.com/golang/protobuf v1.5.2
	github.com/google/certificate-transparency-go v1.1.3
	github.com/google/go-cmp v0.5.8
	github.com/google/go-containerregistry v0.11.0
	github.com/googleapis/gnostic v0.5.5
	github.com/gorilla/schema v1.2.0
	github.com/graph-gophers/graphql-go v1.3.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/hashicorp/go-multierror v1.1.1
	github.com/hashicorp/go-version v1.6.0
	github.com/hashicorp/golang-lru v0.5.4
	github.com/heroku/docker-registry-client v0.0.0
	github.com/jackc/pgtype v1.12.0
	github.com/jackc/pgx/v4 v4.17.1
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
	github.com/np-guard/cluster-topology-analyzer v1.2.3-0.20220802140408-c0ab819afba6
	github.com/nxadm/tail v1.4.8
	github.com/olekukonko/tablewriter v0.0.5
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.0.3-0.20220114050600-8b9d41f48198
	github.com/openshift/api v3.9.1-0.20191201231411-9f834e337466+incompatible
	github.com/openshift/client-go v0.0.0-20200623090625-83993cebb5ae
	github.com/operator-framework/helm-operator-plugins v0.0.7
	github.com/operator-framework/operator-sdk v0.19.4
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.13.0
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.37.0
	github.com/russellhaering/gosaml2 v0.8.0
	github.com/russellhaering/goxmldsig v1.2.0
	github.com/sergi/go-diff v1.2.0
	github.com/sigstore/cosign v1.8.1-0.20220530190726-3a43ddc93914
	github.com/sigstore/sigstore v1.2.1-0.20220528141235-6d98e7d59dee
	github.com/spf13/cobra v1.5.0
	github.com/spf13/pflag v1.0.5
	github.com/stackrox/default-authz-plugin v0.0.0-20210608105219-00ad9c9f3855
	github.com/stackrox/external-network-pusher v0.0.0-20210419192707-074af92bbfa7
	github.com/stackrox/helmtest v0.0.0-20220118100812-1ad97c4de347
	github.com/stackrox/k8s-istio-cve-pusher v0.0.0-20210422200002-d89f671ac4f5
	github.com/stretchr/testify v1.8.0
	github.com/tecbot/gorocksdb v0.0.0-20191217155057-f0fad39f321c
	github.com/tidwall/gjson v1.14.1
	github.com/tkuchiki/go-timezone v0.2.2
	github.com/travelaudience/go-promhttp v1.0.1
	github.com/vbauerster/mpb/v4 v4.12.2
	go.etcd.io/bbolt v1.3.6
	go.uber.org/atomic v1.10.0
	go.uber.org/zap v1.21.0
	golang.org/x/crypto v0.0.0-20220824171710-5757bc0c5503
	golang.org/x/net v0.0.0-20220722155237-a158d28d115b
	golang.org/x/oauth2 v0.0.0-20220822191816-0ebed06d0094
	golang.org/x/sync v0.0.0-20220722155255-886fb9371eb4
	golang.org/x/sys v0.0.0-20220722155257-8c9f86f7a55f
	golang.org/x/time v0.0.0-20220411224347-583f2d630306
	golang.org/x/tools v0.1.12
	golang.stackrox.io/grpc-http1 v0.2.4
	google.golang.org/api v0.94.0
	google.golang.org/genproto v0.0.0-20220810155839-1856144b1d9c
	google.golang.org/grpc v1.49.0
	google.golang.org/grpc/examples v0.0.0-20210902184326-c93e472777b9
	gopkg.in/robfig/cron.v2 v2.0.0-20150107220207-be2e0b0deed5
	gopkg.in/square/go-jose.v2 v2.6.0
	gopkg.in/yaml.v3 v3.0.1
	gorm.io/driver/postgres v1.3.9
	gorm.io/gorm v1.23.8
	gotest.tools v2.2.0+incompatible
	helm.sh/helm/v3 v3.7.2
	k8s.io/api v0.23.10
	k8s.io/apimachinery v0.23.10
	k8s.io/apiserver v0.23.10
	k8s.io/client-go v0.23.10
	k8s.io/kubectl v0.23.1
	k8s.io/kubelet v0.22.13
	k8s.io/utils v0.0.0-20220210201930-3a6ce19ff2f9
	sigs.k8s.io/controller-runtime v0.11.2
	sigs.k8s.io/e2e-framework v0.0.7
	sigs.k8s.io/yaml v1.3.0
)

require (
	bitbucket.org/creachadair/shell v0.0.7 // indirect
	cloud.google.com/go v0.102.1 // indirect
	cloud.google.com/go/iam v0.3.0 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20210617225240-d185dfc1b5a1 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest v0.11.27 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.9.18 // indirect
	github.com/Azure/go-autorest/autorest/date v0.3.0 // indirect
	github.com/Azure/go-autorest/logger v0.2.1 // indirect
	github.com/Azure/go-autorest/tracing v0.6.0 // indirect
	github.com/MakeNowJust/heredoc v0.0.0-20170808103936-bb23615498cd // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.1.1 // indirect
	github.com/Masterminds/squirrel v1.5.2 // indirect
	github.com/Microsoft/go-winio v0.5.2 // indirect
	github.com/Microsoft/hcsshim v0.9.2 // indirect
	github.com/ProtonMail/go-crypto v0.0.0-20220113124808-70ae35bab23f // indirect
	github.com/acarl005/stripansi v0.0.0-20180116102854-5a71ef0e047d // indirect
	github.com/acomagu/bufpipe v1.0.3 // indirect
	github.com/andybalholm/brotli v1.0.3 // indirect
	github.com/asaskevich/govalidator v0.0.0-20210307081110-f21760c49a8d // indirect
	github.com/beevik/etree v1.1.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bgentry/speakeasy v0.1.0 // indirect
	github.com/bits-and-blooms/bitset v1.2.0 // indirect
	github.com/blang/semver v3.5.1+incompatible // indirect
	github.com/blevesearch/go-porterstemmer v1.0.3 // indirect
	github.com/blevesearch/mmap-go v1.0.2 // indirect
	github.com/blevesearch/segment v0.9.0 // indirect
	github.com/census-instrumentation/opencensus-proto v0.3.0 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/chai2010/gettext-go v0.0.0-20160711120539-c6fed771bfd5 // indirect
	github.com/cncf/udpa/go v0.0.0-20210930031921-04548b0d99d4 // indirect
	github.com/cncf/xds/go v0.0.0-20211130200136-a8f946100490 // indirect
	github.com/containerd/cgroups v1.0.1 // indirect
	github.com/containerd/containerd v1.5.9 // indirect
	github.com/containerd/continuity v0.1.0 // indirect
	github.com/containerd/stargz-snapshotter/estargz v0.12.0 // indirect
	github.com/containers/storage v1.38.2 // indirect
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/couchbase/ghistogram v0.1.0 // indirect
	github.com/couchbase/vellum v1.0.2 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.2 // indirect
	github.com/cyberphone/json-canonicalization v0.0.0-20210823021906-dc406ceaf94b // indirect
	github.com/cyphar/filepath-securejoin v0.2.3 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/docker/cli v20.10.17+incompatible // indirect
	github.com/docker/docker-credential-helpers v0.6.4 // indirect
	github.com/docker/go-metrics v0.0.1 // indirect
	github.com/docker/libtrust v0.0.0-20160708172513-aabc10ec26b7 // indirect
	github.com/dsnet/compress v0.0.2-0.20210315054119-f66993602bf5 // indirect
	github.com/dustin/go-humanize v1.0.0 // indirect
	github.com/edsrzf/mmap-go v1.0.0 // indirect
	github.com/emirpasic/gods v1.12.0 // indirect
	github.com/envoyproxy/go-control-plane v0.10.2-0.20220325020618-49ff273808a1 // indirect
	github.com/envoyproxy/protoc-gen-validate v0.6.2 // indirect
	github.com/evanphx/json-patch v4.12.0+incompatible // indirect
	github.com/exponent-io/jsonpath v0.0.0-20151013193312-d6023ce2651d // indirect
	github.com/facebookincubator/flog v0.0.0-20190930132826-d2511d0ce33c // indirect
	github.com/fatih/structs v1.1.0 // indirect
	github.com/form3tech-oss/jwt-go v3.2.5+incompatible // indirect
	github.com/fsnotify/fsnotify v1.5.4 // indirect
	github.com/fullstorydev/grpcurl v1.8.6 // indirect
	github.com/go-chi/chi v4.1.2+incompatible // indirect
	github.com/go-errors/errors v1.0.1 // indirect
	github.com/go-git/gcfg v1.5.0 // indirect
	github.com/go-git/go-billy/v5 v5.3.1 // indirect
	github.com/go-git/go-git/v5 v5.4.2 // indirect
	github.com/go-logr/zapr v1.2.0 // indirect
	github.com/go-openapi/analysis v0.21.2 // indirect
	github.com/go-openapi/errors v0.20.2 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.20.0 // indirect
	github.com/go-openapi/loads v0.21.1 // indirect
	github.com/go-openapi/runtime v0.24.1 // indirect
	github.com/go-openapi/spec v0.20.6 // indirect
	github.com/go-openapi/strfmt v0.21.2 // indirect
	github.com/go-openapi/swag v0.21.1 // indirect
	github.com/go-openapi/validate v0.21.0 // indirect
	github.com/go-playground/locales v0.14.0 // indirect
	github.com/go-playground/universal-translator v0.18.0 // indirect
	github.com/go-playground/validator/v10 v10.11.0 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/golang/glog v1.0.0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/btree v1.0.1 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	github.com/google/trillian v1.4.1 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.1.0 // indirect
	github.com/googleapis/gax-go/v2 v2.4.0 // indirect
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/gosuri/uitable v0.0.4 // indirect
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/huandu/xstrings v1.3.2 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/in-toto/in-toto-golang v0.3.4-0.20211211042327-af1f9fb822bf // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/itchyny/gojq v0.12.5 // indirect
	github.com/itchyny/timefmt-go v0.1.3 // indirect
	github.com/jackc/chunkreader/v2 v2.0.1 // indirect
	github.com/jackc/pgconn v1.13.0 // indirect
	github.com/jackc/pgio v1.0.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgproto3/v2 v2.3.1 // indirect
	github.com/jackc/pgservicefile v0.0.0-20200714003250-2b9c44734f2b // indirect
	github.com/jackc/puddle v1.3.0 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/jedisct1/go-minisign v0.0.0-20211028175153-1c139d1cc84b // indirect
	github.com/jhump/protoreflect v1.10.3 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/jmoiron/sqlx v1.3.4 // indirect
	github.com/jonboulle/clockwork v0.3.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kevinburke/ssh_config v1.1.0 // indirect
	github.com/klauspost/compress v1.15.8 // indirect
	github.com/klauspost/pgzip v1.2.5 // indirect
	github.com/knqyf263/go-rpm-version v0.0.0-20170716094938-74609b86c936 // indirect
	github.com/lann/builder v0.0.0-20180802200727-47ae307949d0 // indirect
	github.com/lann/ps v0.0.0-20150810152359-62de8c46ede0 // indirect
	github.com/leodido/go-urn v1.2.1 // indirect
	github.com/letsencrypt/boulder v0.0.0-20220331220046-b23ab962616e // indirect
	github.com/liggitt/tabwriter v0.0.0-20181228230101-89fcab3d43de // indirect
	github.com/mattermost/xml-roundtrip-validator v0.1.0 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/mattn/go-runewidth v0.0.13 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/mholt/archiver/v3 v3.5.1 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/moby/locker v1.0.1 // indirect
	github.com/moby/spdystream v0.2.0 // indirect
	github.com/moby/sys/mountinfo v0.5.0 // indirect
	github.com/moby/sys/symlink v0.1.0 // indirect
	github.com/moby/term v0.0.0-20210619224110-3f7ff695adc6 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/monochromegane/go-gitignore v0.0.0-20200626010858-205db1a8cc00 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/mschoch/smat v0.2.0 // indirect
	github.com/nwaples/rardecode v1.1.0 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/opencontainers/runc v1.1.0 // indirect
	github.com/opencontainers/runtime-spec v1.0.3-0.20210326190908-1c3f411f0417 // indirect
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/operator-framework/operator-lib v0.3.0 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/pelletier/go-toml/v2 v2.0.1 // indirect
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/pierrec/lz4/v4 v4.1.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/procfs v0.8.0 // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/rogpeppe/go-internal v1.8.1 // indirect
	github.com/rubenv/sql-migrate v0.0.0-20210614095031-55d5740dbbcc // indirect
	github.com/russross/blackfriday v1.6.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/sassoftware/relic v0.0.0-20210427151427-dfb082b79b74 // indirect
	github.com/secure-systems-lab/go-securesystemslib v0.3.1 // indirect
	github.com/shibumi/go-pathspec v1.3.0 // indirect
	github.com/shopspring/decimal v1.2.0 // indirect
	github.com/sigstore/rekor v0.4.1-0.20220114213500-23f583409af3 // indirect
	github.com/sirupsen/logrus v1.9.0 // indirect
	github.com/soheilhy/cmux v0.1.5 // indirect
	github.com/spf13/afero v1.8.2 // indirect
	github.com/spf13/cast v1.5.0 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/viper v1.12.0 // indirect
	github.com/stackrox/dotnet-scraper v0.0.0-20201023051640-72ef543323dd // indirect
	github.com/stackrox/k8s-cves v0.0.0-20201110001126-cc333981eaab // indirect
	github.com/steveyen/gtreap v0.1.0 // indirect
	github.com/stretchr/objx v0.4.0 // indirect
	github.com/subosito/gotenv v1.3.0 // indirect
	github.com/syndtr/gocapability v0.0.0-20200815063812-42c35b437635 // indirect
	github.com/syndtr/goleveldb v1.0.1-0.20210819022825-2ae1ddf74ef7 // indirect
	github.com/tent/canonical-json-go v0.0.0-20130607151641-96e4ba3a7613 // indirect
	github.com/theupdateframework/go-tuf v0.3.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	github.com/titanous/rocacheck v0.0.0-20171023193734-afe73141d399 // indirect
	github.com/tmc/grpc-websocket-proxy v0.0.0-20201229170055-e5319fda7802 // indirect
	github.com/transparency-dev/merkle v0.0.1 // indirect
	github.com/trivago/tgo v1.0.7 // indirect
	github.com/ulikunitz/xz v0.5.10 // indirect
	github.com/urfave/cli v1.22.7 // indirect
	github.com/vbatts/tar-split v0.11.2 // indirect
	github.com/weppos/publicsuffix-go v0.15.1-0.20220329081811-9a40b608a236 // indirect
	github.com/willf/bitset v1.1.11 // indirect
	github.com/xanzy/ssh-agent v0.3.1 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xeipuuv/gojsonschema v1.2.0 // indirect
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
	github.com/xiang90/probing v0.0.0-20190116061207-43a291ad63a2 // indirect
	github.com/xlab/treeprint v0.0.0-20181112141820-a009c3971eca // indirect
	github.com/zmap/zcrypto v0.0.0-20210811211718-6f9bc4aff20f // indirect
	github.com/zmap/zlint/v3 v3.3.1-0.20211019173530-cb17369b4628 // indirect
	go.etcd.io/etcd/api/v3 v3.5.4 // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.5.4 // indirect
	go.etcd.io/etcd/client/v2 v2.305.4 // indirect
	go.etcd.io/etcd/client/v3 v3.5.4 // indirect
	go.etcd.io/etcd/etcdctl/v3 v3.5.4 // indirect
	go.etcd.io/etcd/etcdutl/v3 v3.5.4 // indirect
	go.etcd.io/etcd/pkg/v3 v3.5.4 // indirect
	go.etcd.io/etcd/raft/v3 v3.5.4 // indirect
	go.etcd.io/etcd/server/v3 v3.5.4 // indirect
	go.etcd.io/etcd/tests/v3 v3.5.4 // indirect
	go.etcd.io/etcd/v3 v3.5.4 // indirect
	go.mongodb.org/mongo-driver v1.8.3 // indirect
	go.opencensus.io v0.23.0 // indirect
	go.opentelemetry.io/contrib v1.6.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.20.0 // indirect
	go.opentelemetry.io/otel v0.20.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp v0.20.0 // indirect
	go.opentelemetry.io/otel/metric v0.20.0 // indirect
	go.opentelemetry.io/otel/sdk v0.20.0 // indirect
	go.opentelemetry.io/otel/sdk/export/metric v0.20.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v0.20.0 // indirect
	go.opentelemetry.io/otel/trace v0.20.0 // indirect
	go.opentelemetry.io/proto/otlp v0.12.0 // indirect
	go.starlark.net v0.0.0-20200306205701-8dd3e2ee1dd5 // indirect
	go.uber.org/multierr v1.8.0 // indirect
	golang.org/x/mod v0.6.0-dev.0.20220419223038-86c51ed26bb4 // indirect
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211 // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/xerrors v0.0.0-20220609144429-65e65417b02f // indirect
	gomodules.xyz/jsonpatch/v2 v2.2.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
	gopkg.in/cheggaaa/pb.v1 v1.0.28 // indirect
	gopkg.in/gorp.v1 v1.7.2 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.66.5 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gotest.tools/v3 v3.1.0 // indirect
	k8s.io/apiextensions-apiserver v0.23.5 // indirect
	k8s.io/cli-runtime v0.23.1 // indirect
	k8s.io/component-base v0.23.10 // indirect
	k8s.io/klog/v2 v2.60.1 // indirect
	k8s.io/kube-openapi v0.0.0-20220124234850-424119656bbf // indirect
	knative.dev/pkg v0.0.0-20220325200448-1f7514acd0c2 // indirect
	nhooyr.io/websocket v1.8.7 // indirect
	oras.land/oras-go v0.4.0 // indirect
	sigs.k8s.io/json v0.0.0-20211208200746-9f7c6b3444d2 // indirect
	sigs.k8s.io/kustomize/api v0.10.1 // indirect
	sigs.k8s.io/kustomize/kyaml v0.13.0 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.1 // indirect
)

// To bump the version of a replacement package, use:
//
//   $ go mod edit -replace <package>=<replacement>@<branch or commit reference>
//   $ go mod tidy
//
// For example:
//
//   $ go mod edit -replace github.com/operator-framework/helm-operator-plugins=github.com/stackrox/helm-operator@main
//   $ go mod tidy
//
// The `go mod tidy` takes care of normalizing the symbol version information (e.g. branch name) which is required
// for Go build tools to accept the `go.mod`.
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

	// github.com/stackrox/helm-operator is a modified fork of github.com/operator-framework/helm-operator-plugins that
	// we currently depend on.
	github.com/operator-framework/helm-operator-plugins => github.com/stackrox/helm-operator v0.0.8-0.20220804162433-be98f831243c
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
