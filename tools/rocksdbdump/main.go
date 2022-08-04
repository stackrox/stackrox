package main

import (
	"archive/zip"
	"bytes"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/jsonpb"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/tecbot/gorocksdb"
)

func main() {
	var dbPath string
	var backupFile string
	var outputDir string
	cmd := &cobra.Command{
		Use:   "rocksdbdump",
		Short: "Dump rocks DB",
		Long:  "Dump rocks DB split into per bucket JSON files",
		RunE: func(cmd *cobra.Command, args []string) error {
			return loadAndDump(dbPath, backupFile, outputDir)
		},
		SilenceUsage: true,
	}
	cmd.PersistentFlags().StringVarP(&dbPath, "from-database", "d", "", "Database directory to read from")
	cmd.PersistentFlags().StringVarP(&backupFile, "from-backup-file", "b", "", "Read the database from a backup (roxctl central backup)")
	cmd.PersistentFlags().StringVarP(&outputDir, "output-dir", "o", "", "Dump output to this directory. The directory must exist. Any files with the same names as the rocks DB filterToBuckets will be overwritten.")
	utils.Must(cmd.MarkPersistentFlagRequired("output-dir"))
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var bucketToProtoInterface = map[string]proto.Message{
	"active_components":                    (*storage.ActiveComponent)(nil),
	"alerts":                               (*storage.Alert)(nil),
	"alerts_list":                          (*storage.ListAlert)(nil),
	"apiTokens":                            (*storage.TokenMetadata)(nil),
	"cluster_to_vuln":                      (*storage.ClusterCVEEdge)(nil),
	"clusterinitbundles":                   (*storage.InitBundleMeta)(nil),
	"clusters":                             (*storage.Cluster)(nil),
	"clusters_health_status":               (*storage.ClusterHealthStatus)(nil),
	"comp_to_vuln":                         (*storage.ComponentCVEEdge)(nil),
	"complianceoperatorcheckresults":       (*storage.ComplianceOperatorCheckResult)(nil),
	"complianceoperatorprofiles":           (*storage.ComplianceOperatorProfile)(nil),
	"complianceoperatorrules":              (*storage.ComplianceOperatorRule)(nil),
	"complianceoperatorscansettingbinding": (*storage.ComplianceOperatorScanSettingBinding)(nil),
	"complianceoperatorscans":              (*storage.ComplianceOperatorScan)(nil),
	"deployments":                          (*storage.Deployment)(nil),
	"deployments_list":                     (*storage.ListDeployment)(nil),
	"imageBucket":                          (*storage.Image)(nil),
	"image_component":                      (*storage.ImageComponent)(nil),
	"image_to_comp":                        (*storage.ImageComponentEdge)(nil),
	"image_to_cve":                         (*storage.ImageCVEEdge)(nil),
	"image_vuln":                           (*storage.CVE)(nil),
	"images_list":                          (*storage.ListImage)(nil),
	"integrationhealth":                    (*storage.IntegrationHealth)(nil),
	"k8sroles":                             (*storage.K8SRole)(nil),
	"namespaces":                           (*storage.NamespaceMetadata)(nil),
	"networkFlows2":                        (*storage.NetworkFlow)(nil),
	"networkbaseline":                      (*storage.NetworkBaseline)(nil),
	"networkentity":                        (*storage.NetworkEntity)(nil),
	"networkgraphconfig":                   (*storage.NetworkGraphConfig)(nil),
	"node_to_comp":                         (*storage.NodeComponentEdge)(nil),
	"nodes":                                (*storage.Node)(nil),
	"permission_sets":                      (*storage.PermissionSet)(nil),
	"pods":                                 (*storage.Pod)(nil),
	"policy_categories":                    (*storage.PolicyCategory)(nil),
	"processWhitelistResults":              (*storage.ProcessBaselineResults)(nil),
	"processWhitelists2":                   (*storage.ProcessBaseline)(nil),
	"process_indicators2":                  (*storage.ProcessIndicator)(nil),
	"report_configurations":                (*storage.ReportConfiguration)(nil),
	"risk":                                 (*storage.Risk)(nil),
	"rolebindings":                         (*storage.K8SRoleBinding)(nil),
	"roles":                                (*storage.Role)(nil),
	"signature_integrations":               (*storage.SignatureIntegration)(nil),
	"secrets":                              (*storage.Secret)(nil),
	"service_accounts":                     (*storage.ServiceAccount)(nil),
	"simple_access_scopes":                 (*storage.SimpleAccessScope)(nil),
	"version":                              (*storage.Version)(nil),
	"vuln_req":                             (*storage.VulnerabilityRequest)(nil),
}

var knownUnhandledBuckets = set.NewStringSet(
	"compliance-run-results", // multiple types
	"dackbox_graph",
	"dackbox_dirty",
	"dackbox_reindex",
	"node_to_cve",
)

var ignoreUnmarshallErrors = set.NewStringSet(
	"networkFlows2",
)

func loadAndDump(dbPath string, backupFile string, outputDir string) error {
	if (dbPath == "") == (backupFile == "") {
		return errors.New("A database or backup of a database are required (but not both).")
	}

	if backupFile != "" {
		tmpDbPath, err := restoreRocksDBBackup(backupFile)
		if err != nil {
			return err
		}
		defer func() {
			_ = os.RemoveAll(tmpDbPath)
		}()
		dbPath = tmpDbPath
	}

	log.Printf("Will load and dump the rocks DB at: %s", dbPath)

	db, err := gorocksdb.OpenDb(gorocksdb.NewDefaultOptions(), dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	log.Printf("Opened: %s\n", db.Name())

	marshaller := &jsonpb.Marshaler{
		Indent: "  ",
	}

	bucketsToObjects := make(map[string][]string)

	readOpts := gorocksdb.NewDefaultReadOptions()
	it := db.NewIterator(readOpts)

	foundUnhandledBuckets := set.NewStringSet()
	var fatalErrorsFound bool

	for it.SeekToFirst(); it.Valid(); it.Next() {
		key := it.Key()
		if len(key.Data()) == 0 {
			log.Println("A zero length key was found in the DB")
			fatalErrorsFound = true
			continue
		}

		// Expects a null separated bucket + id form as is typically the case.
		keyPieces := bytes.Split(key.Data(), []byte{0})
		if len(keyPieces[0]) == 0 {
			log.Printf("A bucket name was not found in: %s\n", string(key.Data()))
			fatalErrorsFound = true
			continue
		}

		bucketName := string(keyPieces[0])
		if strings.HasPrefix(bucketName, "transactions") {
			// skip transactions buckets
			continue
		}
		var possibleObjectID string
		if len(keyPieces) > 1 {
			possibleObjectID = string(keyPieces[1])
		}
		pbInterface, ok := bucketToProtoInterface[bucketName]
		if !ok {
			if added := foundUnhandledBuckets.Add(bucketName); added {
				log.Printf("A bucket is missing from the protobuf map: %s\n", bucketName)
				fatalErrorsFound = fatalErrorsFound || !knownUnhandledBuckets.Contains(bucketName)
			}
			continue
		}
		pbType := reflect.TypeOf(pbInterface)
		value := reflect.New(pbType.Elem()).Interface()
		pb, _ := value.(proto.Message)
		err = proto.Unmarshal(it.Value().Data(), pb)
		if err != nil {
			log.Printf("An object cannot be unmarshalled. Bucket: %s, possible ID: %s", bucketName, possibleObjectID)
			log.Println(err)
			log.Printf("len: %d, bytes: %v", len(it.Value().Data()), it.Value().Data())
			fatalErrorsFound = fatalErrorsFound || !ignoreUnmarshallErrors.Contains(bucketName)
			continue
		}

		jsonResult, err := marshaller.MarshalToString(pb)
		if err != nil {
			log.Printf("An object cannot be serialized to JSON. Bucket: %s, possible ID: %s", bucketName, possibleObjectID)
			log.Println(err)
			fatalErrorsFound = fatalErrorsFound || !ignoreUnmarshallErrors.Contains(bucketName)
			continue
		}
		bucketsToObjects[bucketName] = append(bucketsToObjects[bucketName], jsonResult)
	}

	for bucketName, jsonObjects := range bucketsToObjects {
		f, err := os.Create(filepath.Join(outputDir, bucketName+".json"))
		if err != nil {
			return err
		}
		_, err = f.WriteString(
			"[\n" + strings.Join(jsonObjects, ",\n") + "]\n",
		)
		if err != nil {
			return err
		}
		err = f.Close()
		if err != nil {
			return err
		}
		log.Println("Wrote " + bucketName + ".json")
	}

	if fatalErrorsFound {
		return errors.New("The database has unresolved issues")
	}

	return nil
}

func restoreRocksDBBackup(backupFile string) (string, error) {
	parentDir := os.TempDir()
	tmpDbPath, err := os.MkdirTemp(parentDir, "*-rocksdb")
	if err != nil {
		return "", err
	}

	log.Printf("Writing the DB backup to %s for temporary restore\n", tmpDbPath)

	zr, err := zip.OpenReader(backupFile)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = zr.Close()
	}()

	for _, zfile := range zr.File {
		if zfile.Name != "rocks.db" {
			continue
		}
		rocksReader, err := zfile.Open()
		if err != nil {
			return "", err
		}
		tmpDBFilename := filepath.Join(tmpDbPath, zfile.Name)
		rocksWriter, err := os.OpenFile(tmpDBFilename, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return "", err
		}
		_, err = io.Copy(rocksWriter, rocksReader)
		if err != nil {
			return "", err
		}

		untarredDBDir := filepath.Join(tmpDbPath, "untarred")

		cmd := exec.Command("sh", "-c", strings.Join([]string{
			"mkdir -p " + untarredDBDir,
			"tar -C " + untarredDBDir + " -xvf " + tmpDBFilename,
		}, " && "))

		err = cmd.Run()
		if err != nil {
			return "", err
		}

		restoredDBDir := filepath.Join(tmpDbPath, "restored")

		bo := gorocksdb.NewDefaultOptions()
		be, err := gorocksdb.OpenBackupEngine(bo, untarredDBDir)
		if err != nil {
			return "", err
		}
		ro := gorocksdb.NewRestoreOptions()
		defer ro.Destroy()
		err = be.RestoreDBFromLatestBackup(restoredDBDir, restoredDBDir, ro)
		if err != nil {
			return "", err
		}

		return restoredDBDir, nil
	}

	return "", errors.New("Cannot find rocks.db in the backup file")
}
