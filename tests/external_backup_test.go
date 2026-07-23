//go:build externalbackups

package tests

import (
	"context"
	"fmt"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"

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
	"github.com/stackrox/rox/tests/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type backupTestCase struct {
	name  string
	setup func(t *testing.T) (
		backup *storage.ExternalBackup,
		countBackups func(ctx context.Context, t *testing.T, prefix string) int,
		cleanupBackups func(ctx context.Context, t *testing.T, prefix string),
	)
}

var defaultSchedule = &storage.Schedule{
	IntervalType: storage.Schedule_DAILY,
	Hour:         3,
	Minute:       0,
}

func newObjectPrefix() string {
	return uuid.NewString()
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

func gcsTestCases() []backupTestCase {
	return []backupTestCase{
		{
			name: "GCS/service_account_key",
			setup: func(t *testing.T) (*storage.ExternalBackup, func(context.Context, *testing.T, string) int, func(context.Context, *testing.T, string)) {
				env := testutils.EnvOrFail(t,
					"GCP_GCS_BACKUP_TEST_BUCKET_NAME_V2",
					"GOOGLE_GCS_BACKUP_SERVICE_ACCOUNT_V2",
				)
				bucket := env["GCP_GCS_BACKUP_TEST_BUCKET_NAME_V2"]
				serviceAccount := env["GOOGLE_GCS_BACKUP_SERVICE_ACCOUNT_V2"]
				countFn, cleanupFn := newGCSBucketFuncs(t, bucket, serviceAccount)
				prefix := newObjectPrefix()
				return &storage.ExternalBackup{
					Name: "GCS/service_account_key", Type: "gcs", BackupsToKeep: 2,
					Schedule: defaultSchedule,
					Config:   &storage.ExternalBackup_Gcs{Gcs: &storage.GCSConfig{Bucket: bucket, ServiceAccount: serviceAccount, ObjectPrefix: prefix}},
				}, countFn, cleanupFn
			},
		},
		{
			name: "GCS/workload_identity",
			setup: func(t *testing.T) (*storage.ExternalBackup, func(context.Context, *testing.T, string) int, func(context.Context, *testing.T, string)) {
				env := testutils.EnvOrSkip(t, "GCP_GCS_BACKUP_TEST_BUCKET_NAME_V2", "SETUP_WORKLOAD_IDENTITIES")
				if env["SETUP_WORKLOAD_IDENTITIES"] != "true" {
					t.Skip("SETUP_WORKLOAD_IDENTITIES not set to true")
				}
				bucket := env["GCP_GCS_BACKUP_TEST_BUCKET_NAME_V2"]
				countFn, cleanupFn := newGCSBucketFuncs(t, bucket, "")
				prefix := newObjectPrefix()
				return &storage.ExternalBackup{
					Name: "GCS/workload_identity", Type: "gcs", BackupsToKeep: 2,
					Schedule: defaultSchedule,
					Config:   &storage.ExternalBackup_Gcs{Gcs: &storage.GCSConfig{Bucket: bucket, UseWorkloadId: true, ObjectPrefix: prefix}},
				}, countFn, cleanupFn
			},
		},
	}
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

func newAWSS3Setup(name string, makeConfig func(t *testing.T) *storage.S3Config) backupTestCase {
	fullName := "AWS_S3/" + name
	return backupTestCase{
		name: fullName,
		setup: func(t *testing.T) (*storage.ExternalBackup, func(context.Context, *testing.T, string) int, func(context.Context, *testing.T, string)) {
			config := makeConfig(t)
			config.ObjectPrefix = newObjectPrefix()
			countFn, cleanupFn := newS3BucketFuncs(t, "", config.GetRegion(),
				config.GetAccessKeyId(), config.GetSecretAccessKey(), config.GetBucket(), false)
			return &storage.ExternalBackup{
				Name: fullName, Type: "s3", BackupsToKeep: 2,
				Schedule: defaultSchedule,
				Config:   &storage.ExternalBackup_S3{S3: config},
			}, countFn, cleanupFn
		},
	}
}

func awsS3Config(withEndpoint bool) func(t *testing.T) *storage.S3Config {
	return func(t *testing.T) *storage.S3Config {
		env := testutils.EnvOrFail(t,
			"AWS_S3_BACKUP_TEST_BUCKET_NAME",
			"AWS_S3_BACKUP_TEST_BUCKET_REGION",
			"AWS_ACCESS_KEY_ID",
			"AWS_SECRET_ACCESS_KEY",
		)
		config := &storage.S3Config{
			Bucket:          env["AWS_S3_BACKUP_TEST_BUCKET_NAME"],
			Region:          env["AWS_S3_BACKUP_TEST_BUCKET_REGION"],
			AccessKeyId:     env["AWS_ACCESS_KEY_ID"],
			SecretAccessKey: env["AWS_SECRET_ACCESS_KEY"],
		}
		if withEndpoint {
			config.Endpoint = fmt.Sprintf("s3.%s.amazonaws.com", env["AWS_S3_BACKUP_TEST_BUCKET_REGION"])
		}
		return config
	}
}

func awsS3TestCases() []backupTestCase {
	return []backupTestCase{
		newAWSS3Setup("with_endpoint", awsS3Config(true)),
		newAWSS3Setup("without_endpoint", awsS3Config(false)),
	}
}

func newS3CompatibleSetup(name string, makeConfig func(t *testing.T) *storage.S3Compatible) backupTestCase {
	fullName := "S3Compatible/" + name
	return backupTestCase{
		name: fullName,
		setup: func(t *testing.T) (*storage.ExternalBackup, func(context.Context, *testing.T, string) int, func(context.Context, *testing.T, string)) {
			config := makeConfig(t)
			config.ObjectPrefix = newObjectPrefix()
			pathStyle := config.GetUrlStyle() == storage.S3URLStyle_S3_URL_STYLE_PATH
			countFn, cleanupFn := newS3BucketFuncs(t, config.GetEndpoint(), config.GetRegion(),
				config.GetAccessKeyId(), config.GetSecretAccessKey(), config.GetBucket(), pathStyle)
			return &storage.ExternalBackup{
				Name: fullName, Type: "s3compatible", BackupsToKeep: 2,
				Schedule: defaultSchedule,
				Config:   &storage.ExternalBackup_S3Compatible{S3Compatible: config},
			}, countFn, cleanupFn
		},
	}
}

func r2Config(withScheme bool, urlStyle storage.S3URLStyle) func(t *testing.T) *storage.S3Compatible {
	return func(t *testing.T) *storage.S3Compatible {
		env := testutils.EnvOrFail(t,
			"CLOUDFLARE_R2_BACKUP_TEST_ACCOUNT_ID",
			"CLOUDFLARE_R2_BACKUP_TEST_BUCKET_NAME",
			"CLOUDFLARE_R2_BACKUP_TEST_REGION",
			"CLOUDFLARE_R2_BACKUP_TEST_ACCESS_KEY_ID",
			"CLOUDFLARE_R2_BACKUP_TEST_SECRET_ACCESS_KEY",
		)
		endpoint := env["CLOUDFLARE_R2_BACKUP_TEST_ACCOUNT_ID"] + ".r2.cloudflarestorage.com"
		if withScheme {
			endpoint = "https://" + endpoint
		}
		return &storage.S3Compatible{
			Bucket: env["CLOUDFLARE_R2_BACKUP_TEST_BUCKET_NAME"], Region: env["CLOUDFLARE_R2_BACKUP_TEST_REGION"],
			Endpoint: endpoint, AccessKeyId: env["CLOUDFLARE_R2_BACKUP_TEST_ACCESS_KEY_ID"],
			SecretAccessKey: env["CLOUDFLARE_R2_BACKUP_TEST_SECRET_ACCESS_KEY"], UrlStyle: urlStyle,
		}
	}
}

func odfConfig(withScheme bool) func(t *testing.T) *storage.S3Compatible {
	return func(t *testing.T) *storage.S3Compatible {
		env := testutils.EnvOrFail(t,
			"ODF_S3_BACKUP_TEST_ENDPOINT",
			"ODF_S3_BACKUP_TEST_BUCKET_NAME",
			"ODF_S3_BACKUP_TEST_REGION",
			"ODF_S3_BACKUP_TEST_ACCESS_KEY_ID",
			"ODF_S3_BACKUP_TEST_SECRET_ACCESS_KEY",
		)
		endpoint := urlfmt.TrimHTTPPrefixes(env["ODF_S3_BACKUP_TEST_ENDPOINT"])
		if withScheme {
			endpoint = "https://" + endpoint
		}
		return &storage.S3Compatible{
			Bucket: env["ODF_S3_BACKUP_TEST_BUCKET_NAME"], Region: env["ODF_S3_BACKUP_TEST_REGION"],
			Endpoint: endpoint, AccessKeyId: env["ODF_S3_BACKUP_TEST_ACCESS_KEY_ID"],
			SecretAccessKey: env["ODF_S3_BACKUP_TEST_SECRET_ACCESS_KEY"], UrlStyle: storage.S3URLStyle_S3_URL_STYLE_PATH,
		}
	}
}

func s3CompatibleTestCases() []backupTestCase {
	return []backupTestCase{
		newS3CompatibleSetup("CloudflareR2/path-based/endpoint-without-scheme",
			r2Config(false, storage.S3URLStyle_S3_URL_STYLE_PATH)),
		newS3CompatibleSetup("CloudflareR2/path-based/endpoint-with-https",
			r2Config(true, storage.S3URLStyle_S3_URL_STYLE_PATH)),
		newS3CompatibleSetup("CloudflareR2/virtual-hosted/endpoint-without-scheme",
			r2Config(false, storage.S3URLStyle_S3_URL_STYLE_VIRTUAL_HOSTED)),
		newS3CompatibleSetup("CloudflareR2/virtual-hosted/endpoint-with-https",
			r2Config(true, storage.S3URLStyle_S3_URL_STYLE_VIRTUAL_HOSTED)),
		newS3CompatibleSetup("ODF/path-based/endpoint-without-scheme", odfConfig(false)),
		newS3CompatibleSetup("ODF/path-based/endpoint-with-https", odfConfig(true)),
	}
}

func runBackupLifecycleTest(
	t *testing.T,
	service v1.ExternalBackupServiceClient,
	backup *storage.ExternalBackup,
	countBackups func(ctx context.Context, t *testing.T, prefix string) int,
	cleanupBackups func(ctx context.Context, t *testing.T, prefix string),
) {
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
	t.Logf("Using object prefix: %s", prefix)

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
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	created, err := service.PostExternalBackup(ctx, backup)
	require.NoError(t, err)

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		if cleanupBackups != nil {
			cleanupBackups(cleanupCtx, t, prefix)
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
	cases := slices.Concat(gcsTestCases(), awsS3TestCases(), s3CompatibleTestCases())

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			backup, countFn, cleanupFn := tc.setup(t)
			runBackupLifecycleTest(t, service, backup, countFn, cleanupFn)
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
