package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"path/filepath"

	"github.com/google/uuid"

	imageCVEPostgresStore "github.com/stackrox/rox/central/cve/image/datastore/store/postgres"
	dDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/globaldb"
	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/central/image/datastore/keyfence"
	imageStore "github.com/stackrox/rox/central/image/datastore/store"
	imagePostgresStore "github.com/stackrox/rox/central/image/datastore/store/postgres"
	imageComponentPostgresStore "github.com/stackrox/rox/central/imagecomponent/datastore/store/postgres"
	imageComponentEdgeDataStore "github.com/stackrox/rox/central/imagecomponentedge/datastore"
	imageCVEEdgeDataStore "github.com/stackrox/rox/central/imagecveedge/datastore"
	nsDataStore "github.com/stackrox/rox/central/namespace/datastore"
	podDataStore "github.com/stackrox/rox/central/pod/datastore"
	piDataStore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/central/processindicator/filter"
	piPgStore "github.com/stackrox/rox/central/processindicator/store/postgres"
	plopStore "github.com/stackrox/rox/central/processlisteningonport/store/postgres"
	"github.com/stackrox/rox/central/ranking"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	pgGorm "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	// Data for processes is taken from the code for fake Sensor
	processAncestors = []*storage.ProcessSignal_LineageInfo{
		{
			ParentExecFilePath: "bash",
		},
	}

	goodProcessNames = []string{
		"ssl-tools",
		"ansible-tower-s",
		"apache2",
		"apache2-foregro",
		"arangod",
		"asd",
		"awk",
		"awx-manage",
		"basename",
		"beam.smp",
		"bootstrap.sh",
		"cadvisor",
		"calico-node",
		"calico-typha",
		"cat",
		"catalina.sh",
		"central",
		"cfssl",
		"cfssl-helper",
		"cfssljson",
		"chgrp",
		"child_setup",
		"chmod",
		"chown",
		"chpst",
		"chronograf",
		"cluster-proport",
		"collector",
		"compliance",
		"consul",
		"couchbase-serve",
		"cp",
		"cpu_sup",
		"cpvpa",
		"crate",
		"cut",
		"daphne",
		"date",
		"debconf-set-sel",
		"df",
		"dirname",
		"dnsmasq",
		"dnsmasq-nanny",
		"docker-entrypoi",
		"docker-php-entr",
		"egrep",
		"entrypoint.sh",
		"env",
		"epmd",
		"erl",
		"erl_child_setup",
		"erlexec",
		"etcd",
		"expr",
		"failure-event-h",
		"find",
		"free",
		"gateway.start",
		"generate_cert",
		"getconf",
		"getent",
		"getopt",
		"git",
		"gnatsd",
		"goport",
		"gosecrets",
		"gosu",
		"grafana-server",
		"grep",
		"gunicorn",
		"head",
		"heapster",
		"hostname",
		"id",
		"import-addition",
		"inet_gethost",
		"initctl",
		"install-cni.sh",
		"invoke-rc.d",
		"ip-masq-agent",
		"ipset",
		"java",
		"kube-dns",
		"kube-proxy",
		"kubernetes-sens",
		"ldapadd",
		"ldapmodify",
		"ldapsearch",
		"ldconfig",
		"ldconfig.real",
		"ln",
		"log-helper",
		"ls",
		"memsup",
		"metrics-server",
		"mkdir",
		"mktemp",
		"monitor",
		"mosquitto",
		"mv",
		"mysql",
		"mysql_ssl_rsa_s",
		"mysql_tzinfo_to",
		"mysqladmin",
		"mysqld",
		"nats-server",
		"nats-streaming-",
		"nginx",
		"node",
		"openssl",
		"perl",
		"pg_ctlcluster",
		"php",
		"pod_nanny",
		"policy-rc.d",
		"postgres",
		"postgresql",
		"ps",
		"psql",
		"pwgen",
		"python",
		"rabbitmq-server",
		"rabbitmqctl",
		"readlink",
		"redis-server",
		"restore-all-dir",
		"rm",
		"run",
		"run-parts",
		"run.sh",
		"runsv",
		"runsvdir",
		"runsvdir-start",
		"scanner",
		"schema-to-ldif.",
		"sed",
		"server",
		"service",
		"sidecar",
		"slapadd",
		"slapd",
		"slapd.config",
		"slapd.postinst",
		"slapd.prerm",
		"slappasswd",
		"slaptest",
		"sleep",
		"sort",
		"ssl-helper",
		"start-confluenc",
		"start-stop-daem",
		"stat",
		"su",
		"su-exec",
		"supervisor",
		"supervisord",
		"tail",
		"tar",
		"tini",
		"touch",
		"tr",
		"uname",
		"update-rc.d",
		"uwsgi",
		"wc",
		"webproc",
		"whoami",
	}

	badProcessNames = []string{
		"wget",
		"curl",
		"bash",
		"sh",
		"zsh",
		"nmap",
		"groupadd",
		"addgroup",
		"useradd",
		"adduser",
		"usermod",
		"apk",
		"apt-get",
		"apt",
		"chkconfig",
		"anacron",
		"cron",
		"crond",
		"crontab",
		"rpm",
		"dnf",
		"yum",
		"iptables",
		"make",
		"gcc",
		"llc",
		"llvm-gcc",
		"sgminer",
		"cgminer",
		"cpuminer",
		"minerd",
		"geth",
		"ethminer",
		"xmr-stak-cpu",
		"xmr-stak-amd",
		"xmr-stak-nvidia",
		"xmrminer",
		"cpuminer-multi",
		"ifrename",
		"ethtool",
		"ifconfig",
		"ipmaddr",
		"iptunnel",
		"route",
		"nameif",
		"mii-tool",
		"nc",
		"nmap",
		"scp",
		"sshfs",
		"ssh-copy-id",
		"rsync",
		"sshd",
		"systemctl",
		"systemd",
	}

	activeProcessNames = []string{
		"/bin/bash",
		"/bin/busybox",
		"/bin/cat",
		"/bin/chgrp",
		"/bin/chmod",
		"/bin/chown",
		"/bin/chvt",
		"/bin/cp",
		"/bin/cpio",
		"/bin/dash",
		"/bin/date",
		"/bin/dd",
		"/bin/df",
		"/bin/dir",
		"/bin/echo",
		"/bin/egrep",
		"/bin/false",
		"/bin/fgrep",
		"/bin/fusermount",
		"/bin/grep",
		"/bin/gunzip",
		"/bin/gzexe",
		"/bin/gzip",
		"/bin/hostname",
		"/bin/ip",
		"/bin/journalctl",
		"/bin/kill",
		"/bin/ln",
		"/bin/ls",
		"/bin/mkdir",
		"/bin/mknod",
		"/bin/mktemp",
		"/bin/mount",
		"/bin/mountpoint",
		"/bin/mv",
		"/bin/ping",
		"/bin/pwd",
		"/bin/readlink",
		"/bin/rm",
		"/bin/rmdir",
		"/bin/sed",
		"/bin/sleep",
		"/bin/stty",
		"/bin/su",
		"/bin/sync",
		"/bin/tar",
		"/bin/touch",
		"/bin/true",
		"/bin/uname",
		"/bin/uncompress",
		"/bin/vdir",
		"/bin/whiptail",
		"/bin/zcat",
		"/bin/zcmp",
		"/bin/zdiff",
		"/bin/zegrep",
		"/bin/zfgrep",
		"/bin/zforce",
		"/bin/zgrep",
		"/bin/zless",
		"/bin/zmore",
		"/bin/znew",
		"/etc/cron.daily/apt",
		"/etc/cron.daily/dpkg",
		"/etc/security/namespace.init",
		"/etc/ssl/misc/CA.pl",
		"/lib/ld-musl-x86_64.so.1",
		"/lib/libcrypto.so.1.0.0",
		"/lib/libcrypto.so.1.1",
		"/lib/libcrypto.so.42.0.0",
		"/lib/libssl.so.1.1",
		"/lib/libssl.so.45.0.1",
		"/lib/libtls.so.17.0.1",
		"/lib/libz.so.1.2.8",
		"/lib/libz.so.1.2.11",
		"/sbin/apk",
		"/sbin/badblocks",
		"/sbin/ldconfig",
		"/sbin/mkmntdirs",
		"/sbin/tini",
		"/sbin/xtables-multi",
		"/usr/bin/apt",
		"/usr/bin/arch",
		"/usr/bin/b2sum",
		"/usr/bin/base32",
		"/usr/bin/base64",
		"/usr/bin/basename",
		"/usr/bin/bash",
		"/usr/bin/cal",
		"/usr/bin/certutil",
		"/usr/bin/chcon",
		"/usr/bin/cksum",
		"/usr/bin/clear",
		"/usr/bin/cmp",
		"/usr/bin/cpio",
		"/usr/bin/comm",
		"/usr/bin/csplit",
		"/usr/bin/curl",
		"/usr/bin/cut",
		"/usr/bin/diff",
		"/usr/bin/diff3",
		"/usr/bin/dircolors",
		"/usr/bin/dirname",
		"/usr/bin/doveadm",
		"/usr/bin/dpkg",
		"/usr/bin/du",
		"/usr/bin/eject",
		"/usr/bin/env",
		"/usr/bin/expand",
		"/usr/bin/expr",
		"/usr/bin/factor",
		"/usr/bin/file",
		"/usr/bin/find",
		"/usr/bin/fmt",
		"/usr/bin/fold",
		"/usr/bin/gdbus",
		"/usr/bin/git",
		"/usr/bin/git-lfs",
		"/usr/bin/gpgv",
		"/usr/bin/groups",
		"/usr/bin/head",
		"/usr/bin/hostid",
		"/usr/bin/id",
		"/usr/bin/info",
		"/usr/bin/install",
		"/usr/bin/join",
		"/usr/bin/jq",
		"/usr/bin/ldd",
		"/usr/bin/less",
		"/usr/bin/link",
		"/usr/bin/logname",
		"/usr/bin/make",
		"/usr/bin/mawk",
		"/usr/bin/md5sum",
		"/usr/bin/mkfifo",
		"/usr/bin/nice",
		"/usr/bin/nl",
		"/usr/bin/nohup",
		"/usr/bin/nproc",
		"/usr/bin/numfmt",
		"/usr/bin/od",
		"/usr/bin/openssl",
		"/usr/bin/paste",
		"/usr/bin/pathchk",
		"/usr/bin/perl",
		"/usr/bin/php7",
		"/usr/bin/pinky",
		"/usr/bin/pip",
		"/usr/bin/pr",
		"/usr/bin/printenv",
		"/usr/bin/printf",
		"/usr/bin/ptx",
		"/usr/bin/realpath",
		"/usr/bin/rgrep",
		"/usr/bin/runcon",
		"/usr/bin/scanelf",
		"/usr/bin/seq",
		"/usr/bin/sdiff",
		"/usr/bin/sha1sum",
		"/usr/bin/sha224sum",
		"/usr/bin/sha256sum",
		"/usr/bin/sha384sum",
		"/usr/bin/sha512sum",
		"/usr/bin/shred",
		"/usr/bin/shuf",
		"/usr/bin/sort",
		"/usr/bin/split",
		"/usr/bin/ssl_client",
		"/usr/bin/sshfs",
		"/usr/bin/stat",
		"/usr/bin/stdbuf",
		"/usr/bin/sum",
		"/usr/bin/tac",
		"/usr/bin/tail",
		"/usr/bin/tee",
		"/usr/bin/test",
		"/usr/bin/timeout",
		"/usr/bin/tr",
		"/usr/bin/truncate",
		"/usr/bin/tsort",
		"/usr/bin/tty",
		"/usr/bin/unexpand",
		"/usr/bin/uniq",
		"/usr/bin/unlink",
		"/usr/bin/unzip",
		"/usr/bin/update-ca-trust",
		"/usr/bin/users",
		"/usr/bin/vi",
		"/usr/bin/wc",
		"/usr/bin/wget",
		"/usr/bin/who",
		"/usr/bin/whoami",
		"/usr/bin/xargs",
		"/usr/bin/yes",
		"/usr/bin/zip",
		"/usr/lib/libcurl.so.4.6.0",
		"/usr/lib/libsqlite3.so.0.8.6",
		"/usr/lib/node_modules/npm/bin/npm",
		"/usr/lib64/libpython2.7.so.1.0",
		"/usr/lib64/libz.so.1.2.7",
		"/usr/sbin/chroot",
		"/usr/sbin/rmt-tar",
		"/usr/sbin/sshd",
		"/usr/sbin/tarcat",
		"/usr/sbin/tzconfig",
		"/usr/sbin/update-ca-certificates",
	}
)

func Rand(n int) (str string) {
	b := make([]byte, n)
	rand.Read(b)
	str = fmt.Sprintf("%x", b)
	return
}

func RandPath(length int, n int) (str string) {
	path := ""

	for i := 0; i < length; i++ {
		path += fmt.Sprintf("/%s", Rand(n))
	}

	return path
}

func RandList(length int, n int) (list []string) {
	list = []string{}

	for i := 0; i < length; i++ {
		list = append(list, Rand(n))
	}

	return list
}

func RandMap(length int, n int) (result map[string]string) {
	result = map[string]string{}

	for i := 0; i < length; i++ {
		result[Rand(n)] = Rand(n)
	}

	return result
}

type NameUuidPair struct {
	Name string
	Uuid string
}

func RandPairList(length int, n int) (list []NameUuidPair) {
	list = []NameUuidPair{}

	for i := 0; i < length; i++ {
		list = append(list, NameUuidPair{
			Name: Rand(n),
			Uuid: uuid.New().String(),
		})
	}

	return list
}

func genDeployment(clusters []NameUuidPair,
	namespaces []NameUuidPair) *storage.Deployment {

	ns := namespaces[rand.Intn(len(namespaces))]
	cluster := clusters[rand.Intn(len(clusters))]

	return &storage.Deployment{
		Id:          uuid.New().String(),
		Name:        Rand(10),
		Namespace:   ns.Name,
		NamespaceId: ns.Uuid,
		ClusterId:   cluster.Uuid,
		ClusterName: cluster.Name,
	}
}

func genPod(deployments []*storage.Deployment) *storage.Pod {

	deployment := deployments[rand.Intn(len(deployments))]

	return &storage.Pod{
		Id:           uuid.New().String(),
		Name:         Rand(10),
		DeploymentId: deployment.Id,
		ClusterId:    deployment.ClusterId,
		Namespace:    deployment.Namespace,
		Started:      protocompat.GetProtoTimestampFromSeconds(0),
	}
}

func genImage() *storage.Image {
	return &storage.Image{
		Id: uuid.New().String(),
		Name: &storage.ImageName{
			FullName: Rand(10),
		},
		Metadata: &storage.ImageMetadata{
			V1: &storage.V1Metadata{
				Created:    protocompat.TimestampNow(),
				User:       Rand(10),
				Command:    RandList(10, 2),
				Entrypoint: RandList(10, 2),
				Volumes:    RandList(10, 2),
				Labels:     RandMap(10, 2),
			},
		},
		Scan: &storage.ImageScan{
			ScanTime:        protocompat.TimestampNow(),
			OperatingSystem: Rand(10),
		},
		Signature: &storage.ImageSignature{
			Fetched: protocompat.TimestampNow(),
		},
	}
}

func genImageComponent() *storage.ImageComponent {
	return &storage.ImageComponent{
		Id:   uuid.New().String(),
		Name: Rand(10),
	}
}

func genImageComponentEdge(
	image *storage.Image,
	component *storage.ImageComponent,
) *storage.ImageComponentEdge {

	return &storage.ImageComponentEdge{
		Id:               uuid.New().String(),
		ImageId:          image.Id,
		ImageComponentId: component.Id,
	}
}

func genImageCVE() *storage.ImageCVE {
	return &storage.ImageCVE{
		Id:              uuid.New().String(),
		OperatingSystem: Rand(10),
		CveBaseInfo: &storage.CVEInfo{
			Cve:          Rand(10),
			CreatedAt:    protocompat.TimestampNow(),
			LastModified: protocompat.TimestampNow(),
			PublishedOn:  protocompat.TimestampNow(),
		},
	}
}

func genImageCVEEdge(
	image *storage.Image,
	cve *storage.ImageCVE,
) *storage.ImageCVEEdge {

	return &storage.ImageCVEEdge{
		Id:         uuid.New().String(),
		ImageId:    image.Id,
		ImageCveId: cve.Id,
		State:      storage.VulnerabilityState(rand.Intn(3)),
	}
}

func genComponentCVEEdge(
	image *storage.Image,
	cve *storage.ImageCVE,
) *storage.ImageCVEEdge {

	return &storage.ImageCVEEdge{
		Id:         uuid.New().String(),
		ImageId:    image.Id,
		ImageCveId: cve.Id,
		State:      storage.VulnerabilityState(rand.Intn(3)),
	}
}

func main() {
	var batchSize = flag.Int("batch-size", 10000, "Size of a batch to insert")
	var nrBatches = flag.Int("batch-count", 10000, "Number of batches to insert")
	var nrDeployments = flag.Int("deployments-count", 1000, "Number of deployments")
	var nrPods = flag.Int("pods-count", 10000, "Number of pods")
	var nrNamespaces = flag.Int("ns-count", 100, "Number of namespaces")
	var nrClusters = flag.Int("clusters-count", 100, "Number of clusters")
	var nrImages = flag.Int("images-count", 100, "Number of images")
	var nrImageComponents = flag.Int("image-components-count", 100, "Number of image components")
	var nrImageCVEs = flag.Int("image-cves-count", 100, "Number of image CVEs")

	var connectionString = flag.String("connection-string", "host=localhost", "DB Connection string")
	var migrations = flag.Bool("migrations", false, "Whether to apply migrations first")
	var useGlobalDB = flag.Bool("globaldb", false, "Whether to use global database configuration")

	var processIndicatorDS piDataStore.DataStore
	var deploymentDS dDataStore.DataStore
	var podDS podDataStore.DataStore
	var nsDS nsDataStore.DataStore
	var imageDS imageDataStore.DataStore
	//var imageComponentDS imageComponentDataStore.DataStore
	var imagePgStore imageStore.Store
	var imageComponentPgStore imageComponentPostgresStore.Store
	var imageCVEPgStore imageCVEPostgresStore.Store
	var imageComponentEdgeStore imageComponentEdgeDataStore.DataStore
	var imageCVEEdgeStore imageCVEEdgeDataStore.DataStore

	var db postgres.DB
	var err error

	flag.Parse()
	fmt.Printf("batch-size %d, batch-count %d\n", *batchSize, *nrBatches)

	if *migrations {
		gormDB, err := gorm.Open(pgGorm.Open(*connectionString), &gorm.Config{
			NamingStrategy:    pgutils.NamingStrategy,
			CreateBatchSize:   1000,
			AllowGlobalUpdate: true,
			Logger:            logger.Discard,
			QueryFields:       true,
		})

		if err != nil {
			fmt.Println(err)
			return
		}

		pkgSchema.ApplyAllSchemas(context.Background(), gormDB)
		rawDB, err := gormDB.DB()
		rawDB.Close()
	}

	if *useGlobalDB {
		ctx := context.Background()
		globaldb.InitializePostgres(ctx)

		processIndicatorDS = piDataStore.Singleton()
		deploymentDS = dDataStore.Singleton()
		podDS = podDataStore.Singleton()
		nsDS = nsDataStore.Singleton()
	} else {
		db, err = postgres.Connect(context.TODO(), *connectionString)
		if err != nil {
			fmt.Println(err)
			return
		}

		store := piPgStore.New(db)
		plopStorage := plopStore.New(db)
		nsStorage := nsDataStore.NewStorage(db)

		processIndicatorDS, err = piDataStore.New(store, plopStorage, nil, nil)
		deploymentDS, err = dDataStore.New(db, nil, nil, nil, nil, nil, nil,
			nil, nil, ranking.DeploymentRanker())

		podDS, err = podDataStore.NewPostgresDB(db, nil, nil, filter.Singleton())

		nsDS = nsDataStore.New(nsStorage, nil, ranking.NamespaceRanker())

		if err != nil {
			fmt.Println(err)
			return
		}

		imagePgStore = imagePostgresStore.New(db, false, keyfence.ImageKeyFenceSingleton())
		imageDS = imageDataStore.NewWithPostgres(imagePgStore, nil, nil, nil)

		imageComponentPgStore = imageComponentPostgresStore.New(db)
		//imageComponentDS = imageComponentDataStore.New(imageComponentPgStore, nil, nil, nil)

		imageCVEPgStore = imageCVEPostgresStore.New(db)

		imageComponentEdgePgStore := imageComponentEdgeDataStore.NewStorage(db)
		imageComponentEdgeStore, err = imageComponentEdgeDataStore.New(imageComponentEdgePgStore, nil)

		imageCVEEdgePgStore := imageCVEEdgeDataStore.NewStorage(db)
		imageCVEEdgeStore = imageCVEEdgeDataStore.New(imageCVEEdgePgStore, nil)

		if err != nil {
			fmt.Println(err)
			return
		}
	}

	namespaces := RandPairList(*nrNamespaces, 10)
	clusters := RandPairList(*nrClusters, 10)

	deployments := []*storage.Deployment{}
	for i := 0; i < *nrDeployments; i++ {
		deployments = append(deployments, genDeployment(clusters, namespaces))
	}

	pods := []*storage.Pod{}
	for i := 0; i < *nrPods; i++ {
		pods = append(pods, genPod(deployments))
	}

	containers := RandList(10, 10)

	lifecycleMgmt := sac.WithAllAccess(context.Background())

	images := []*storage.Image{}
	for i := 0; i < *nrImages; i++ {
		images = append(images, genImage())
	}

	imageComponents := []*storage.ImageComponent{}
	for i := 0; i < *nrImageComponents; i++ {
		imageComponents = append(imageComponents, genImageComponent())
	}

	imageComponentEdges := []*storage.ImageComponentEdge{}
	for i := 0; i < *nrImageComponents; i++ {
		image := images[i]
		imageComponent := imageComponents[i]
		imageComponentEdges = append(imageComponentEdges,
			genImageComponentEdge(image, imageComponent))
	}

	imageCVEs := []*storage.ImageCVE{}
	for i := 0; i < *nrImageCVEs; i++ {
		imageCVEs = append(imageCVEs, genImageCVE())
	}

	imageCVEEdges := []*storage.ImageCVEEdge{}
	for i := 0; i < *nrImageCVEs; i++ {
		image := images[rand.Intn(len(images))]
		cve := imageCVEs[i]
		imageCVEEdges = append(imageCVEEdges, genImageCVEEdge(image, cve))
	}

	//for _, image := range images {
	//imagePgStore.Upsert(lifecycleMgmt, image)
	//}

	imageDS.UpsertMany(db, lifecycleMgmt, images)

	imageComponentPgStore.UpsertMany(lifecycleMgmt, imageComponents)

	imageCVEPgStore.UpsertMany(lifecycleMgmt, imageCVEs)

	imageComponentEdgeStore.UpsertMany(db, lifecycleMgmt, imageComponentEdges)

	imageCVEEdgeStore.UpsertMany(db, lifecycleMgmt, imageCVEEdges)

	for i := 0; i < *nrDeployments; i++ {
		fmt.Printf("Insert deployment %s\n", deployments[i].Id)
		err := deploymentDS.UpsertDeployment(lifecycleMgmt, deployments[i])

		if err != nil {
			fmt.Println(err)
			return
		}
	}

	for i := 0; i < *nrPods; i++ {
		fmt.Printf("Insert pod %s\n", pods[i].Id)
		err := podDS.UpsertPod(lifecycleMgmt, pods[i])

		if err != nil {
			fmt.Println(err)
			return
		}
	}

	for i := 0; i < *nrNamespaces; i++ {
		fmt.Printf("Insert namespace %s\n", namespaces[i].Uuid)
		ns := &storage.NamespaceMetadata{
			Id:   namespaces[i].Uuid,
			Name: namespaces[i].Name,
		}

		err := nsDS.AddNamespace(lifecycleMgmt, ns)

		if err != nil {
			fmt.Println(err)
			return
		}
	}

	for i := 0; i < *nrBatches; i++ {
		indicators := []*storage.ProcessIndicator{}

		for j := 0; j < *batchSize; j++ {
			name := goodProcessNames[rand.Intn(len(goodProcessNames))]

			pod := pods[rand.Intn(len(pods))]
			container := containers[rand.Intn(len(containers))]

			pi := &storage.ProcessIndicator{
				Id:                 uuid.New().String(),
				DeploymentId:       pod.DeploymentId,
				PodId:              pod.Id,
				PodUid:             pod.Id,
				ClusterId:          pod.ClusterId,
				Namespace:          pod.Namespace,
				ContainerName:      container,
				ContainerStartTime: protocompat.TimestampNow(),
				Signal: &storage.ProcessSignal{
					Uid:          rand.Uint32(),
					Time:         protocompat.TimestampNow(),
					Name:         name,
					Args:         fmt.Sprintf("%s %s %s", Rand(5), Rand(5), Rand(5)),
					ExecFilePath: filepath.Clean("/usr/bin/" + name),
					LineageInfo:  processAncestors,
				},
			}
			indicators = append(indicators, pi)
		}

		fmt.Printf("Insert process indicators batch %s\n", indicators[0].Id)
		processIndicatorDS.AddProcessIndicators(lifecycleMgmt, indicators...)
	}
}
