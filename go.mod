module github.com/stackrox/rox

go 1.21

toolchain go1.21.7

require (
	cloud.google.com/go/artifactregistry v1.14.6
	cloud.google.com/go/compute/metadata v0.2.3
	cloud.google.com/go/containeranalysis v0.11.3
	cloud.google.com/go/securitycenter v1.24.3
	cloud.google.com/go/storage v1.37.0
	github.com/BurntSushi/toml v1.3.2
	github.com/ComplianceAsCode/compliance-operator v1.4.0
	github.com/Masterminds/semver v1.5.0
	github.com/Masterminds/sprig/v3 v3.2.3
	github.com/NYTimes/gziphandler v1.1.1
	github.com/PagerDuty/go-pagerduty v1.8.0
	github.com/RoaringBitmap/roaring v1.9.1
	github.com/Shopify/toxiproxy/v2 v2.8.0
	github.com/VividCortex/ewma v1.2.0
	github.com/adhocore/gronx v1.8.0
	github.com/andygrunwald/go-jira v1.16.0
	github.com/aws/aws-sdk-go v1.51.12
	github.com/cenkalti/backoff/v3 v3.2.2
	github.com/cloudflare/cfssl v1.6.5
	github.com/cockroachdb/pebble v1.1.0
	github.com/containers/image/v5 v5.29.2
	github.com/coreos/go-oidc/v3 v3.9.0
	github.com/coreos/go-systemd/v22 v22.5.0
	github.com/dave/jennifer v1.7.0
	github.com/docker/distribution v2.8.3+incompatible
	github.com/facebookincubator/nvdtools v0.1.5
	github.com/fatih/color v1.16.0
	github.com/georgysavva/scany/v2 v2.1.2
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-jose/go-jose/v3 v3.0.3
	github.com/go-logr/logr v1.4.1
	github.com/go-logr/zapr v1.3.0
	github.com/godbus/dbus/v5 v5.1.0
	github.com/gofrs/uuid v4.4.0+incompatible
	github.com/gogo/protobuf v1.3.2
	github.com/golang-jwt/jwt/v4 v4.5.0
	github.com/golang/mock v1.6.0
	github.com/golang/protobuf v1.5.4
	github.com/google/certificate-transparency-go v1.1.7
	github.com/google/gnostic-models v0.6.9-0.20230804172637-c7be7c783f49
	github.com/google/go-cmp v0.6.0
	github.com/google/go-containerregistry v0.19.1
	github.com/google/uuid v1.6.0
	github.com/googleapis/gax-go/v2 v2.12.0
	github.com/gorilla/schema v1.3.0
	github.com/graph-gophers/graphql-go v1.5.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/hashicorp/go-multierror v1.1.1
	github.com/hashicorp/go-retryablehttp v0.7.5
	github.com/hashicorp/go-version v1.6.0
	github.com/hashicorp/golang-lru/v2 v2.0.7
	github.com/heimdalr/dag v1.4.0
	github.com/helm/helm-mapkubeapis v0.4.1
	github.com/heroku/docker-registry-client v0.0.0
	github.com/jackc/pgtype v1.14.3
	github.com/jackc/pgx/v4 v4.18.3
	github.com/jackc/pgx/v5 v5.5.5
	github.com/joshdk/go-junit v1.0.0
	github.com/klauspost/compress v1.17.7
	github.com/lib/pq v1.10.9
	github.com/machinebox/graphql v0.2.2
	github.com/mailru/easyjson v0.7.7
	github.com/mitchellh/go-wordwrap v1.0.1
	github.com/mitchellh/hashstructure/v2 v2.0.2
	github.com/np-guard/cluster-topology-analyzer/v2 v2.2.1
	github.com/np-guard/netpol-analyzer v1.1.0
	github.com/nxadm/tail v1.4.8
	github.com/olekukonko/tablewriter v0.0.5
	github.com/onsi/ginkgo/v2 v2.16.0
	github.com/onsi/gomega v1.31.1
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.1.0
	github.com/openshift-online/ocm-sdk-go v0.1.413
	github.com/openshift/api v0.0.0-20231117201702-2ea16bbab164
	github.com/openshift/client-go v0.0.0-20230926161409-848405da69e1
	github.com/openshift/runtime-utils v0.0.0-20230921210328-7bdb5b9c177b
	github.com/operator-framework/helm-operator-plugins v0.0.0-00010101000000-000000000000
	github.com/owenrumney/go-sarif/v2 v2.3.0
	github.com/pkg/browser v0.0.0-20210911075715-681adbf594b8
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.19.0
	github.com/prometheus/client_model v0.6.0
	github.com/prometheus/common v0.51.1
	github.com/quay/claircore v1.5.25
	github.com/quay/claircore/toolkit v1.1.1
	github.com/quay/goval-parser v0.8.8
	github.com/quay/zlog v1.1.8
	github.com/remind101/migrate v0.0.0-20170729031349-52c1edff7319
	github.com/rs/zerolog v1.32.0
	github.com/russellhaering/gosaml2 v0.9.1
	github.com/russellhaering/goxmldsig v1.4.0
	github.com/segmentio/analytics-go/v3 v3.3.0
	github.com/sergi/go-diff v1.3.1
	github.com/sigstore/cosign/v2 v2.2.2
	github.com/sigstore/sigstore v1.8.2
	github.com/sourcegraph/conc v0.3.0
	github.com/spf13/cobra v1.8.0
	github.com/spf13/pflag v1.0.6-0.20210604193023-d5e0c0615ace
	github.com/stackrox/external-network-pusher v0.0.0-20231115153210-b82d72f500a2
	github.com/stackrox/helmtest v0.0.1
	github.com/stackrox/k8s-overlay-patch v0.0.0-20230704092804-d66fa79cb2d1
	github.com/stackrox/pkcs7 v0.0.0-20240314170115-841ca6b5f88d
	github.com/stackrox/scanner v0.0.0-20240110222630-351caa1e0024
	github.com/stretchr/testify v1.9.0
	github.com/tidwall/gjson v1.17.1
	github.com/tkuchiki/go-timezone v0.2.2
	github.com/travelaudience/go-promhttp v1.0.1
	github.com/vbauerster/mpb/v4 v4.12.2
	go.etcd.io/bbolt v1.3.9
	go.uber.org/atomic v1.11.0
	go.uber.org/goleak v1.3.0
	go.uber.org/mock v0.4.0
	go.uber.org/zap v1.26.0
	golang.org/x/crypto v0.21.0
	golang.org/x/exp v0.0.0-20240110193028-0dcbfd608b1e
	golang.org/x/net v0.22.0
	golang.org/x/oauth2 v0.18.0
	golang.org/x/sync v0.6.0
	golang.org/x/sys v0.18.0
	golang.org/x/text v0.14.0
	golang.org/x/time v0.5.0
	golang.org/x/tools v0.18.0
	golang.stackrox.io/grpc-http1 v0.3.9
	google.golang.org/api v0.157.0
	google.golang.org/genproto v0.0.0-20240116215550-a9fa1716bcac
	google.golang.org/grpc v1.61.1
	google.golang.org/grpc/examples v0.0.0-20210902184326-c93e472777b9
	google.golang.org/protobuf v1.33.0
	gopkg.in/mcuadros/go-syslog.v2 v2.3.0
	gopkg.in/robfig/cron.v2 v2.0.0-20150107220207-be2e0b0deed5
	gopkg.in/yaml.v3 v3.0.1
	gorm.io/driver/postgres v1.5.7
	gorm.io/gorm v1.25.7-0.20240204074919-46816ad31dde
	helm.sh/helm/v3 v3.14.3
	k8s.io/api v0.29.3
	k8s.io/apiextensions-apiserver v0.29.3
	k8s.io/apimachinery v0.29.3
	k8s.io/apiserver v0.29.3
	k8s.io/cli-runtime v0.29.3
	k8s.io/client-go v0.29.3
	k8s.io/kubectl v0.29.3
	k8s.io/kubelet v0.29.3
	k8s.io/utils v0.0.0-20240102154912-e7106e64919e
	sigs.k8s.io/controller-runtime v0.17.2
	sigs.k8s.io/controller-tools v0.14.0
	sigs.k8s.io/e2e-framework v0.3.0
	sigs.k8s.io/yaml v1.4.0
)

require (
	cloud.google.com/go v0.112.0 // indirect
	cloud.google.com/go/compute v1.23.3 // indirect
	cloud.google.com/go/iam v1.1.5 // indirect
	cloud.google.com/go/longrunning v0.5.4 // indirect
	dario.cat/mergo v1.0.0 // indirect
	github.com/AdaLogics/go-fuzz-headers v0.0.0-20230811130428-ced1acdcaa24 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20230124172434-306776ec8161 // indirect
	github.com/DataDog/zstd v1.4.5 // indirect
	github.com/MakeNowJust/heredoc v1.0.0 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.2.1 // indirect
	github.com/Masterminds/squirrel v1.5.4 // indirect
	github.com/Microsoft/go-winio v0.6.1 // indirect
	github.com/Microsoft/hcsshim v0.12.0-rc.1 // indirect
	github.com/ProtonMail/go-crypto v0.0.0-20230923063757-afb1ddc0824c // indirect
	github.com/acarl005/stripansi v0.0.0-20180116102854-5a71ef0e047d // indirect
	github.com/andybalholm/brotli v1.0.4 // indirect
	github.com/antlr/antlr4/runtime/Go/antlr/v4 v4.0.0-20230305170008-8188dc5388df // indirect
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2 // indirect
	github.com/aymerick/douceur v0.2.0 // indirect
	github.com/beevik/etree v1.2.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bits-and-blooms/bitset v1.12.0 // indirect
	github.com/blang/semver v3.5.1+incompatible // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869 // indirect
	github.com/cenkalti/backoff/v4 v4.2.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/chai2010/gettext-go v1.0.2 // indirect
	github.com/cloudflare/circl v1.3.7 // indirect
	github.com/cockroachdb/cockroach-go/v2 v2.3.5 // indirect
	github.com/cockroachdb/errors v1.11.1 // indirect
	github.com/cockroachdb/logtags v0.0.0-20230118201751-21c54148d20b // indirect
	github.com/cockroachdb/redact v1.1.5 // indirect
	github.com/cockroachdb/tokenbucket v0.0.0-20230807174530-cc333fc44b06 // indirect
	github.com/containerd/containerd v1.7.12 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/containerd/stargz-snapshotter/estargz v0.15.1 // indirect
	github.com/containers/storage v1.51.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.3 // indirect
	github.com/cyberphone/json-canonicalization v0.0.0-20231011164504-785e29786b46 // indirect
	github.com/cyphar/filepath-securejoin v0.2.4 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/digitorus/pkcs7 v0.0.0-20230818184609-3a137a874352 // indirect
	github.com/digitorus/timestamp v0.0.0-20230902153158-687734543647 // indirect
	github.com/distribution/distribution/v3 v3.0.0-20230511163743-f7717b7855ca // indirect
	github.com/distribution/reference v0.5.0 // indirect
	github.com/docker/cli v24.0.7+incompatible // indirect
	github.com/docker/docker v24.0.9+incompatible // indirect
	github.com/docker/docker-credential-helpers v0.8.0 // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-metrics v0.0.1 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/docker/libtrust v0.0.0-20160708172513-aabc10ec26b7 // indirect
	github.com/doug-martin/goqu/v8 v8.6.0 // indirect
	github.com/dsnet/compress v0.0.2-0.20210315054119-f66993602bf5 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/emicklei/go-restful/v3 v3.11.0 // indirect
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/evanphx/json-patch v5.7.0+incompatible // indirect
	github.com/evanphx/json-patch/v5 v5.8.0 // indirect
	github.com/exponent-io/jsonpath v0.0.0-20210407135951-1de76d718b3f // indirect
	github.com/facebookincubator/flog v0.0.0-20190930132826-d2511d0ce33c // indirect
	github.com/fatih/camelcase v1.0.0 // indirect
	github.com/fatih/structs v1.1.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/getsentry/sentry-go v0.18.0 // indirect
	github.com/go-chi/chi v4.1.2+incompatible // indirect
	github.com/go-errors/errors v1.4.2 // indirect
	github.com/go-git/gcfg v1.5.1-0.20230307220236-3a3c6141e376 // indirect
	github.com/go-git/go-billy/v5 v5.5.0 // indirect
	github.com/go-git/go-git/v5 v5.11.0 // indirect
	github.com/go-gorp/gorp/v3 v3.1.0 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/analysis v0.21.4 // indirect
	github.com/go-openapi/errors v0.20.4 // indirect
	github.com/go-openapi/jsonpointer v0.20.0 // indirect
	github.com/go-openapi/jsonreference v0.20.2 // indirect
	github.com/go-openapi/loads v0.21.2 // indirect
	github.com/go-openapi/runtime v0.26.0 // indirect
	github.com/go-openapi/spec v0.20.11 // indirect
	github.com/go-openapi/strfmt v0.21.8 // indirect
	github.com/go-openapi/swag v0.22.4 // indirect
	github.com/go-openapi/validate v0.22.3 // indirect
	github.com/go-task/slim-sprig v0.0.0-20230315185526-52ccab3ef572 // indirect
	github.com/gobuffalo/flect v1.0.2 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/golang/glog v1.2.0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/btree v1.1.2 // indirect
	github.com/google/cel-go v0.17.7 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/pprof v0.0.0-20231023181126-ff6d637d2a7b // indirect
	github.com/google/s2a-go v0.1.7 // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.2 // indirect
	github.com/gorilla/css v1.0.0 // indirect
	github.com/gorilla/mux v1.8.1 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/gosuri/uitable v0.0.4 // indirect
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.18.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/hcl v1.0.1-vault-5 // indirect
	github.com/huandu/xstrings v1.4.0 // indirect
	github.com/imdario/mergo v0.3.16 // indirect
	github.com/in-toto/in-toto-golang v0.9.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/itchyny/gojq v0.12.14 // indirect
	github.com/itchyny/timefmt-go v0.1.5 // indirect
	github.com/jackc/chunkreader/v2 v2.0.1 // indirect
	github.com/jackc/pgconn v1.14.3 // indirect
	github.com/jackc/pgio v1.0.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgproto3/v2 v2.3.3 // indirect
	github.com/jackc/pgservicefile v0.0.0-20231201235250-de7065d80cb9 // indirect
	github.com/jackc/puddle v1.3.0 // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/jedisct1/go-minisign v0.0.0-20230811132847-661be99b8267 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/jmoiron/sqlx v1.3.5 // indirect
	github.com/jonboulle/clockwork v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kevinburke/ssh_config v1.2.0 // indirect
	github.com/klauspost/pgzip v1.2.6 // indirect
	github.com/knqyf263/go-apk-version v0.0.0-20200609155635-041fdbb8563f // indirect
	github.com/knqyf263/go-deb-version v0.0.0-20190517075300-09fca494f03d // indirect
	github.com/knqyf263/go-rpm-version v0.0.0-20220614171824-631e686d1075 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/lann/builder v0.0.0-20180802200727-47ae307949d0 // indirect
	github.com/lann/ps v0.0.0-20150810152359-62de8c46ede0 // indirect
	github.com/letsencrypt/boulder v0.0.0-20231026200631-000cd05d5491 // indirect
	github.com/liggitt/tabwriter v0.0.0-20181228230101-89fcab3d43de // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/matryer/is v1.2.0 // indirect
	github.com/mattermost/xml-roundtrip-validator v0.1.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.15 // indirect
	github.com/mholt/archiver/v3 v3.5.1 // indirect
	github.com/microcosm-cc/bluemonday v1.0.21 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/moby/locker v1.0.1 // indirect
	github.com/moby/spdystream v0.2.0 // indirect
	github.com/moby/sys/mountinfo v0.7.1 // indirect
	github.com/moby/term v0.5.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/monochromegane/go-gitignore v0.0.0-20200626010858-205db1a8cc00 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/mschoch/smat v0.2.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/mxk/go-flowrate v0.0.0-20140419014527-cca7078d478f // indirect
	github.com/ncruces/go-strftime v0.1.9 // indirect
	github.com/nozzle/throttler v0.0.0-20180817012639-2ea982251481 // indirect
	github.com/nwaples/rardecode v1.1.0 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/opencontainers/runc v1.1.12 // indirect
	github.com/opencontainers/runtime-spec v1.1.0 // indirect
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/operator-framework/operator-lib v0.11.1-0.20230607132417-ecb9be488378 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/pelletier/go-toml/v2 v2.1.0 // indirect
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/pierrec/lz4/v4 v4.1.15 // indirect
	github.com/pjbgf/sha1cd v0.3.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/procfs v0.12.0 // indirect
	github.com/quay/claircore/updater/driver v1.0.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/rivo/uniseg v0.4.4 // indirect
	github.com/rogpeppe/go-internal v1.11.0 // indirect
	github.com/rubenv/sql-migrate v1.5.2 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/sagikazarmark/locafero v0.3.0 // indirect
	github.com/sagikazarmark/slog-shim v0.1.0 // indirect
	github.com/sassoftware/relic v7.2.1+incompatible // indirect
	github.com/secure-systems-lab/go-securesystemslib v0.8.0 // indirect
	github.com/segmentio/backo-go v1.0.1 // indirect
	github.com/shibumi/go-pathspec v1.3.0 // indirect
	github.com/shopspring/decimal v1.3.1 // indirect
	github.com/sigstore/rekor v1.3.4 // indirect
	github.com/sigstore/timestamp-authority v1.2.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/skeema/knownhosts v1.2.1 // indirect
	github.com/skratchdot/open-golang v0.0.0-20200116055534-eef842397966 // indirect
	github.com/spf13/afero v1.10.0 // indirect
	github.com/spf13/cast v1.6.0 // indirect
	github.com/spf13/viper v1.17.0 // indirect
	github.com/stackrox/dotnet-scraper v0.0.0-20201023051640-72ef543323dd // indirect
	github.com/stackrox/istio-cves v0.0.0-20221007013142-0bde9b541ec8 // indirect
	github.com/stackrox/k8s-cves v0.0.0-20220818200547-7d0d1420c58d // indirect
	github.com/stoewer/go-strcase v1.2.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/syndtr/gocapability v0.0.0-20200815063812-42c35b437635 // indirect
	github.com/syndtr/goleveldb v1.0.1-0.20220721030215-126854af5e6d // indirect
	github.com/theupdateframework/go-tuf v0.7.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/titanous/rocacheck v0.0.0-20171023193734-afe73141d399 // indirect
	github.com/transparency-dev/merkle v0.0.2 // indirect
	github.com/trivago/tgo v1.0.7 // indirect
	github.com/ulikunitz/xz v0.5.11 // indirect
	github.com/vbatts/tar-split v0.11.5 // indirect
	github.com/weppos/publicsuffix-go v0.30.1-0.20230620154423-38c92ad2d5c6 // indirect
	github.com/xanzy/ssh-agent v0.3.3 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xeipuuv/gojsonschema v1.2.0 // indirect
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
	github.com/xlab/treeprint v1.2.0 // indirect
	github.com/zmap/zcrypto v0.0.0-20230310154051-c8b263fd8300 // indirect
	github.com/zmap/zlint/v3 v3.5.0 // indirect
	go.mongodb.org/mongo-driver v1.12.1 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.46.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.46.1 // indirect
	go.opentelemetry.io/otel v1.24.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.19.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.19.0 // indirect
	go.opentelemetry.io/otel/metric v1.24.0 // indirect
	go.opentelemetry.io/otel/sdk v1.21.0 // indirect
	go.opentelemetry.io/otel/trace v1.24.0 // indirect
	go.opentelemetry.io/proto/otlp v1.0.0 // indirect
	go.starlark.net v0.0.0-20230612165344-9532f5667272 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/mod v0.15.0 // indirect
	golang.org/x/term v0.18.0 // indirect
	gomodules.xyz/jsonpatch/v2 v2.4.0 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240122161410-6c6643bf1457 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240116215550-a9fa1716bcac // indirect
	gopkg.in/go-jose/go-jose.v2 v2.6.3 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	k8s.io/component-base v0.29.3 // indirect
	k8s.io/klog/v2 v2.120.0 // indirect
	k8s.io/kube-openapi v0.0.0-20240105020646-a37d4de58910 // indirect
	modernc.org/gc/v3 v3.0.0-20240107210532-573471604cb6 // indirect
	modernc.org/libc v1.41.0 // indirect
	modernc.org/mathutil v1.6.0 // indirect
	modernc.org/memory v1.7.2 // indirect
	modernc.org/sqlite v1.29.2 // indirect
	modernc.org/strutil v1.2.0 // indirect
	modernc.org/token v1.1.0 // indirect
	nhooyr.io/websocket v1.8.10 // indirect
	oras.land/oras-go v1.2.4 // indirect
	sigs.k8s.io/apiserver-network-proxy/konnectivity-client v0.28.0 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/kustomize/api v0.13.5-0.20230601165947-6ce0bf390ce3 // indirect
	sigs.k8s.io/kustomize/kyaml v0.16.0 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.4.1 // indirect
)

// HOW TO BUMP
// ===========
//
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

// @stackrox/core-workflows
replace github.com/nxadm/tail => github.com/stackrox/tail v1.4.9-0.20210831224919-407035634f5d

// @stackrox/draco
// github.com/stackrox/helm-operator is a modified fork of github.com/operator-framework/helm-operator-plugins that
// we currently depend on.
replace github.com/operator-framework/helm-operator-plugins => github.com/stackrox/helm-operator v0.0.12-0.20230825152000-1361e2f7db46

// @stackrox/merlin
replace (
	// Our fork has following features:
	// - generate MessageClone() and Clone() functions
	// - error on nil unmarshal instead of panic
	// - trailing comments propagation to generated code
	github.com/gogo/protobuf => github.com/stackrox/protobuf v1.2.2-0.20240207122816-e936d453291c

	// Our fork has the following changes:
	// - fetch signatures without fetching the image manifest
	// This has been added since we already fetch the image manifest
	// in a previous step as a prereq.
	github.com/sigstore/cosign/v2 => github.com/stackrox/cosign/v2 v2.0.0-20231206143943-626efb842da7

	// Our fok has following features:
	// - console log field ordering
	// - not verbose error logging
	// TODO(ROX-23217): upgrade to latest version
	go.uber.org/zap => github.com/stackrox/zap v1.18.2-0.20240314134248-5f932edd0404

	// Our fork has a change exposing a method to do generic POST requests
	// against the OAuth server in order to realize the refresh token flow.
	// The problem is that:
	//   (a) the oauth2 library doesn’t support token refresh out of the box;
	//   (b) authenticating with an OAuth server is super complicated because
	//       there is a mix of header auth and body auth in existence, which
	//       the library solves with autosensing + caching, and what we don't
	//       want to reimplement in our code.
	golang.org/x/oauth2 => github.com/stackrox/oauth2 v0.0.0-20240305225512-3737c3c758df

	// Both github.com/go-yaml/yaml/v2 and github.com/go-yaml/yaml/v3 do not provide go.sum
	// so dependabot is not able to check dependecies.
	// See https://github.com/go-yaml/yaml/issues/772
	// Therefore we point all to our fork of `go-yaml` - github.com/stackrox/yaml/v2|v3
	// where we provide the actual `go.sum`.
	gopkg.in/yaml.v2 => github.com/stackrox/yaml/v2 v2.4.1
	gopkg.in/yaml.v3 => github.com/stackrox/yaml/v3 v3.0.0
)

// @stackrox/scanner
replace (
	github.com/facebookincubator/nvdtools => github.com/stackrox/nvdtools v0.0.0-20231111002313-57e262e4797e
	github.com/heroku/docker-registry-client => github.com/stackrox/docker-registry-client v0.0.0-20230714151239-78b1f5f70b8a
)
