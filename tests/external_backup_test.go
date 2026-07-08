//go:build externalbackups

package tests

import (
	"context"
	"fmt"
	"os"
	"slices"
	"testing"
	"time"

	googleStorage "cloud.google.com/go/storage"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type backupTestCase struct {
	name           string
	skip           string
	backup         *storage.ExternalBackup
	countBackups   func(ctx context.Context, t *testing.T, prefix string) int
	cleanupBackups func(ctx context.Context, t *testing.T, prefix string)
}

var defaultSchedule = &storage.Schedule{
	IntervalType: storage.Schedule_DAILY,
	Hour:         3,
	Minute:       0,
}

var prefixSeq int

func newObjectPrefix() string {
	prefixSeq++
	if id := os.Getenv("BUILD_ID"); id != "" {
		return fmt.Sprintf("%s/%d", id, prefixSeq)
	}
	return fmt.Sprintf("test-%d-%d", time.Now().UnixNano(), prefixSeq)
}

// envOrSkip returns the value of the env var, or sets skip reason if any required var is missing.
func envOrSkip(vars map[string]string, names ...string) (skip string) {
	for _, n := range names {
		v := os.Getenv(n)
		if v == "" {
			return fmt.Sprintf("env var %s not set", n)
		}
		vars[n] = v
	}
	return ""
}

func newGCSBucketFuncs(t *testing.T, bucket, serviceAccount string) (
	countFn func(ctx context.Context, t *testing.T, prefix string) int,
	cleanupFn func(ctx context.Context, t *testing.T, prefix string),
) {
	var opts []option.ClientOption
	if serviceAccount != "" {
		opts = append(opts, option.WithAuthCredentialsJSON(option.ServiceAccount, []byte(serviceAccount)))
	}
	client, err := googleStorage.NewClient(context.Background(), opts...)
	require.NoError(t, err)

	bkt := client.Bucket(bucket)
	countFn = func(ctx context.Context, t *testing.T, prefix string) int {
		it := bkt.Objects(ctx, &googleStorage.Query{Prefix: prefix})
		count := 0
		var iterErr error
		for _, iterErr = it.Next(); iterErr == nil; _, iterErr = it.Next() {
			count++
		}
		require.Equal(t, iterator.Done, iterErr)
		return count
	}
	cleanupFn = func(ctx context.Context, t *testing.T, prefix string) {
		it := bkt.Objects(ctx, &googleStorage.Query{Prefix: prefix})
		for {
			attrs, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				t.Logf("Warning: listing objects for cleanup: %v", err)
				return
			}
			if err := bkt.Object(attrs.Name).Delete(ctx); err != nil {
				t.Logf("Warning: deleting object %s: %v", attrs.Name, err)
			}
		}
	}
	return countFn, cleanupFn
}

func gcsTestCases(t *testing.T) []backupTestCase {
	saEnv := make(map[string]string)
	saSkip := envOrSkip(saEnv,
		"GCP_GCS_BACKUP_TEST_BUCKET_NAME_V2",
		"GOOGLE_GCS_BACKUP_SERVICE_ACCOUNT_V2",
	)
	bucket := saEnv["GCP_GCS_BACKUP_TEST_BUCKET_NAME_V2"]
	serviceAccount := saEnv["GOOGLE_GCS_BACKUP_SERVICE_ACCOUNT_V2"]

	wifEnv := make(map[string]string)
	wifSkip := envOrSkip(wifEnv,
		"GCP_GCS_BACKUP_TEST_BUCKET_NAME_V2",
		"SETUP_WORKLOAD_IDENTITIES",
	)
	if wifSkip == "" && wifEnv["SETUP_WORKLOAD_IDENTITIES"] != "true" {
		wifSkip = "SETUP_WORKLOAD_IDENTITIES not set to true"
	}

	providers := []struct {
		name   string
		skip   string
		config *storage.GCSConfig
	}{
		{"GCS/service_account_key", saSkip, &storage.GCSConfig{
			Bucket: bucket, ServiceAccount: serviceAccount,
		}},
		{"GCS/workload_identity", wifSkip, &storage.GCSConfig{
			Bucket: bucket, UseWorkloadId: true,
		}},
	}

	var countFn func(ctx context.Context, t *testing.T, prefix string) int
	var cleanupFn func(ctx context.Context, t *testing.T, prefix string)
	if bucket != "" {
		countFn, cleanupFn = newGCSBucketFuncs(t, bucket, serviceAccount)
	}

	var cases []backupTestCase
	for _, p := range providers {
		p.config.ObjectPrefix = newObjectPrefix()
		cases = append(cases, backupTestCase{
			name: p.name,
			skip: p.skip,
			backup: &storage.ExternalBackup{
				Name:          p.name,
				Type:          "gcs",
				BackupsToKeep: 2,
				Schedule:      defaultSchedule,
				Config:        &storage.ExternalBackup_Gcs{Gcs: p.config},
			},
			countBackups:   countFn,
			cleanupBackups: cleanupFn,
		})
	}
	return cases
}

func newS3BucketFuncs(t *testing.T, endpoint, region, accessKeyID, secretAccessKey, bucket string, pathStyle bool) (
	countFn func(ctx context.Context, t *testing.T, prefix string) int,
	cleanupFn func(ctx context.Context, t *testing.T, prefix string),
) {
	cfg, err := awsConfig.LoadDefaultConfig(context.Background(),
		awsConfig.WithRegion(region),
		awsConfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, ""),
		),
		awsConfig.WithRequestChecksumCalculation(aws.RequestChecksumCalculationWhenRequired),
		awsConfig.WithResponseChecksumValidation(aws.ResponseChecksumValidationWhenRequired),
	)
	require.NoError(t, err)

	var clientOpts []func(*s3.Options)
	if endpoint != "" {
		ep := urlfmt.FormatURL(endpoint, urlfmt.HTTPS, urlfmt.HonorInputSlash)
		clientOpts = append(clientOpts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(ep)
		})
	}
	if pathStyle {
		clientOpts = append(clientOpts, func(o *s3.Options) {
			o.UsePathStyle = true
		})
	}
	client := s3.NewFromConfig(cfg, clientOpts...)

	countFn = func(ctx context.Context, t *testing.T, prefix string) int {
		out, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket: aws.String(bucket),
			Prefix: aws.String(prefix),
		})
		require.NoError(t, err)
		return int(aws.ToInt32(out.KeyCount))
	}
	cleanupFn = func(ctx context.Context, t *testing.T, prefix string) {
		out, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket: aws.String(bucket),
			Prefix: aws.String(prefix),
		})
		if err != nil {
			t.Logf("Warning: listing objects for cleanup: %v", err)
			return
		}
		for _, obj := range out.Contents {
			if _, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{
				Bucket: aws.String(bucket),
				Key:    obj.Key,
			}); err != nil {
				t.Logf("Warning: deleting object %s: %v", aws.ToString(obj.Key), err)
			}
		}
	}
	return countFn, cleanupFn
}

func awsS3TestCases(t *testing.T) []backupTestCase {
	env := make(map[string]string)
	if skip := envOrSkip(env,
		"AWS_S3_BACKUP_TEST_BUCKET_NAME",
		"AWS_S3_BACKUP_TEST_BUCKET_REGION",
		"AWS_ACCESS_KEY_ID",
		"AWS_SECRET_ACCESS_KEY",
	); skip != "" {
		return []backupTestCase{
			{name: "AWS_S3/with_endpoint", skip: skip},
			{name: "AWS_S3/without_endpoint", skip: skip},
		}
	}
	bucket := env["AWS_S3_BACKUP_TEST_BUCKET_NAME"]
	region := env["AWS_S3_BACKUP_TEST_BUCKET_REGION"]
	accessKeyID := env["AWS_ACCESS_KEY_ID"]
	secretAccessKey := env["AWS_SECRET_ACCESS_KEY"]
	endpoint := fmt.Sprintf("s3.%s.amazonaws.com", region)

	providers := []struct {
		name   string
		config *storage.S3Config
	}{
		{"AWS_S3/with_endpoint", &storage.S3Config{
			Bucket: bucket, Region: region, Endpoint: endpoint,
			AccessKeyId: accessKeyID, SecretAccessKey: secretAccessKey,
		}},
		{"AWS_S3/without_endpoint", &storage.S3Config{
			Bucket: bucket, Region: region,
			AccessKeyId: accessKeyID, SecretAccessKey: secretAccessKey,
		}},
	}

	countFn, cleanupFn := newS3BucketFuncs(t, "", region, accessKeyID, secretAccessKey, bucket, false)
	var cases []backupTestCase
	for _, p := range providers {
		p.config.ObjectPrefix = newObjectPrefix()
		cases = append(cases, backupTestCase{
			name: p.name,
			backup: &storage.ExternalBackup{
				Name:          p.name,
				Type:          "s3",
				BackupsToKeep: 2,
				Schedule:      defaultSchedule,
				Config:        &storage.ExternalBackup_S3{S3: p.config},
			},
			countBackups:   countFn,
			cleanupBackups: cleanupFn,
		})
	}
	return cases
}

func s3CompatibleTestCases(t *testing.T) []backupTestCase {
	r2Env := make(map[string]string)
	r2Skip := envOrSkip(r2Env,
		"CLOUDFLARE_R2_BACKUP_TEST_ACCOUNT_ID",
		"CLOUDFLARE_R2_BACKUP_TEST_BUCKET_NAME",
		"CLOUDFLARE_R2_BACKUP_TEST_REGION",
		"CLOUDFLARE_R2_BACKUP_TEST_ACCESS_KEY_ID",
		"CLOUDFLARE_R2_BACKUP_TEST_SECRET_ACCESS_KEY",
	)
	r2Endpoint := r2Env["CLOUDFLARE_R2_BACKUP_TEST_ACCOUNT_ID"] + ".r2.cloudflarestorage.com"
	r2Bucket := r2Env["CLOUDFLARE_R2_BACKUP_TEST_BUCKET_NAME"]
	r2Region := r2Env["CLOUDFLARE_R2_BACKUP_TEST_REGION"]
	r2KeyID := r2Env["CLOUDFLARE_R2_BACKUP_TEST_ACCESS_KEY_ID"]
	r2Secret := r2Env["CLOUDFLARE_R2_BACKUP_TEST_SECRET_ACCESS_KEY"]

	odfEnv := make(map[string]string)
	odfSkip := envOrSkip(odfEnv,
		"ODF_S3_BACKUP_TEST_ENDPOINT",
		"ODF_S3_BACKUP_TEST_BUCKET_NAME",
		"ODF_S3_BACKUP_TEST_REGION",
		"ODF_S3_BACKUP_TEST_ACCESS_KEY_ID",
		"ODF_S3_BACKUP_TEST_SECRET_ACCESS_KEY",
	)
	odfEndpoint := urlfmt.TrimHTTPPrefixes(odfEnv["ODF_S3_BACKUP_TEST_ENDPOINT"])
	odfBucket := odfEnv["ODF_S3_BACKUP_TEST_BUCKET_NAME"]
	odfRegion := odfEnv["ODF_S3_BACKUP_TEST_REGION"]
	odfKeyID := odfEnv["ODF_S3_BACKUP_TEST_ACCESS_KEY_ID"]
	odfSecret := odfEnv["ODF_S3_BACKUP_TEST_SECRET_ACCESS_KEY"]

	providers := []struct {
		name   string
		skip   string
		config *storage.S3Compatible
	}{
		{"CloudflareR2/path-based/endpoint-without-scheme", r2Skip, &storage.S3Compatible{
			Bucket: r2Bucket, Region: r2Region, Endpoint: r2Endpoint,
			AccessKeyId: r2KeyID, SecretAccessKey: r2Secret, UrlStyle: storage.S3URLStyle_S3_URL_STYLE_PATH,
		}},
		{"CloudflareR2/path-based/endpoint-with-https", r2Skip, &storage.S3Compatible{
			Bucket: r2Bucket, Region: r2Region, Endpoint: "https://" + r2Endpoint,
			AccessKeyId: r2KeyID, SecretAccessKey: r2Secret, UrlStyle: storage.S3URLStyle_S3_URL_STYLE_PATH,
		}},
		{"CloudflareR2/virtual-hosted/endpoint-without-scheme", r2Skip, &storage.S3Compatible{
			Bucket: r2Bucket, Region: r2Region, Endpoint: r2Endpoint,
			AccessKeyId: r2KeyID, SecretAccessKey: r2Secret, UrlStyle: storage.S3URLStyle_S3_URL_STYLE_VIRTUAL_HOSTED,
		}},
		{"CloudflareR2/virtual-hosted/endpoint-with-https", r2Skip, &storage.S3Compatible{
			Bucket: r2Bucket, Region: r2Region, Endpoint: "https://" + r2Endpoint,
			AccessKeyId: r2KeyID, SecretAccessKey: r2Secret, UrlStyle: storage.S3URLStyle_S3_URL_STYLE_VIRTUAL_HOSTED,
		}},
		{"ODF/path-based/endpoint-without-scheme", odfSkip, &storage.S3Compatible{
			Bucket: odfBucket, Region: odfRegion, Endpoint: odfEndpoint,
			AccessKeyId: odfKeyID, SecretAccessKey: odfSecret, UrlStyle: storage.S3URLStyle_S3_URL_STYLE_PATH,
		}},
		{"ODF/path-based/endpoint-with-https", odfSkip, &storage.S3Compatible{
			Bucket: odfBucket, Region: odfRegion, Endpoint: "https://" + odfEndpoint,
			AccessKeyId: odfKeyID, SecretAccessKey: odfSecret, UrlStyle: storage.S3URLStyle_S3_URL_STYLE_PATH,
		}},
	}

	var cases []backupTestCase
	for _, p := range providers {
		name := "S3Compatible/" + p.name
		p.config.ObjectPrefix = newObjectPrefix()

		var countFn func(ctx context.Context, t *testing.T, prefix string) int
		var cleanupFn func(ctx context.Context, t *testing.T, prefix string)
		if p.skip == "" {
			pathStyle := p.config.GetUrlStyle() == storage.S3URLStyle_S3_URL_STYLE_PATH
			countFn, cleanupFn = newS3BucketFuncs(t, p.config.GetEndpoint(), p.config.GetRegion(),
				p.config.GetAccessKeyId(), p.config.GetSecretAccessKey(), p.config.GetBucket(), pathStyle)
		}

		cases = append(cases, backupTestCase{
			name: name,
			skip: p.skip,
			backup: &storage.ExternalBackup{
				Name:          name,
				Type:          "s3compatible",
				BackupsToKeep: 2,
				Schedule:      defaultSchedule,
				Config:        &storage.ExternalBackup_S3Compatible{S3Compatible: p.config},
			},
			countBackups:   countFn,
			cleanupBackups: cleanupFn,
		})
	}
	return cases
}

func runBackupLifecycleTest(t *testing.T, service v1.ExternalBackupServiceClient, tc backupTestCase) {
	backup := tc.backup
	countBackups := tc.countBackups

	var prefix string
	switch cfg := backup.GetConfig().(type) {
	case *storage.ExternalBackup_Gcs:
		prefix = cfg.Gcs.GetObjectPrefix()
	case *storage.ExternalBackup_S3:
		prefix = cfg.S3.GetObjectPrefix()
	case *storage.ExternalBackup_S3Compatible:
		prefix = cfg.S3Compatible.GetObjectPrefix()
	}
	require.NotEmpty(t, prefix, "objectPrefix must be set on the backup config")

	// Retry TestExternalBackup in case Central is not fully ready yet.
	err := retry.WithRetry(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, err := service.TestExternalBackup(ctx, backup)
		return err
	},
		retry.Tries(10),
		retry.BetweenAttempts(func(_ int) {
			time.Sleep(10 * time.Second)
		}),
		retry.OnFailedAttempts(func(err error) {
			t.Logf("Error testing external backup: %v", err)
		}),
	)
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	created, err := service.PostExternalBackup(ctx, backup)
	require.NoError(t, err)
	cancel()

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		if tc.cleanupBackups != nil {
			tc.cleanupBackups(cleanupCtx, t, prefix)
		}
		if created != nil {
			if _, err := service.DeleteExternalBackup(cleanupCtx, &v1.ResourceByID{Id: created.GetId()}); err != nil {
				t.Logf("Warning: deleting external backup config: %v", err)
			}
		}
	})

	countCtx, countCancel := context.WithTimeout(context.Background(), 10*time.Second)
	assert.Equal(t, 0, countBackups(countCtx, t, prefix))
	countCancel()

	for i := 1; i <= 3; i++ {
		triggerCtx, triggerCancel := context.WithTimeout(context.Background(), 2*time.Minute)
		_, err = service.TriggerExternalBackup(triggerCtx, &v1.ResourceByID{Id: created.GetId()})
		assert.NoError(t, err)
		triggerCancel()

		if i <= 2 {
			verifyCtx, verifyCancel := context.WithTimeout(context.Background(), 10*time.Second)
			assert.Equal(t, i, countBackups(verifyCtx, t, prefix))
			verifyCancel()
		}
	}

	// Third backup should prune the first, keeping only BackupsToKeep=2.
	err = retry.WithRetry(func() error {
		retryCtx, retryCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer retryCancel()
		n := countBackups(retryCtx, t, prefix)
		if n != 2 {
			return fmt.Errorf("expected 2 backups after pruning, got %d", n)
		}
		return nil
	},
		retry.Tries(10),
		retry.BetweenAttempts(func(_ int) {
			time.Sleep(1 * time.Second)
		}),
		retry.OnFailedAttempts(func(err error) {
			t.Logf("Error waiting for backup pruning: %v", err)
		}),
	)
	require.NoError(t, err)
}

func TestExternalBackup(t *testing.T) {
	if os.Getenv("BYODB_TEST") == "true" {
		t.Skip("Backup service is not available with external db")
	}

	conn := centralgrpc.GRPCConnectionToCentral(t)
	service := v1.NewExternalBackupServiceClient(conn)
	cases := slices.Concat(gcsTestCases(t), awsS3TestCases(t), s3CompatibleTestCases(t))

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.skip != "" {
				t.Skip(tc.skip)
			}
			runBackupLifecycleTest(t, service, tc)
		})
	}
}

func TestExternalBackupErrorOnExternalDB(t *testing.T) {
	if os.Getenv("BYODB_TEST") != "true" {
		t.Skip("Only runs with external db (BYODB_TEST=true)")
	}

	conn := centralgrpc.GRPCConnectionToCentral(t)
	service := v1.NewExternalBackupServiceClient(conn)

	backup := &storage.ExternalBackup{
		Name:          "should fail on external db",
		Type:          "s3",
		BackupsToKeep: 1,
		Schedule: &storage.Schedule{
			IntervalType: storage.Schedule_DAILY,
			Hour:         0,
			Minute:       0,
		},
		Config: &storage.ExternalBackup_S3{
			S3: &storage.S3Config{
				Bucket:          "dummy",
				Region:          "us-east-1",
				AccessKeyId:     "dummy",
				SecretAccessKey: "dummy",
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := service.TestExternalBackup(ctx, backup)
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok, "expected gRPC status error")
	assert.Equal(t, codes.NotFound, st.Code())
	assert.Contains(t, st.Message(), "Please manage backups directly with your database provider.")
}
