package utils

import (
	"context"
	"io/ioutil"
	"path/filepath"
	"time"

	"cloud.google.com/go/storage"
	"github.com/pkg/errors"
	"google.golang.org/api/iterator"
)

const (
	// gCloudClientTimeout is the timeout value we use for
	// clients connected to Google Cloud
	gCloudClientTimeout = 3 * time.Minute
)

// WriteToBucket writes the specified content to the specified bucket.
//  - bucketName: Name of the bucket this fn writes to
//  - objectPrefix: Prefix of the file. EX: folder1
//  - objectName: Name for the file that this fn creates for writing out data.
//  - data: Content of the file.
func WriteToBucket(
	bucketName string,
	objectPrefix string,
	objectName string,
	data []byte,
) error {
	objectPath := filepath.Join(objectPrefix, objectName)

	ctx, cancel := context.WithTimeout(context.Background(), gCloudClientTimeout)
	defer cancel()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}

	writer := client.Bucket(bucketName).Object(objectPath).NewWriter(ctx)
	if _, err = writer.Write(data); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return nil
}

// DeleteObjectWithPrefix deletes any object in the specified bucket that
// starts with the specified prefix.
func DeleteObjectWithPrefix(bucketName, prefix string) error {
	names, err := GetAllObjectNamesWithPrefix(bucketName, prefix)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), gCloudClientTimeout)
	defer cancel()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}

	bucket := client.Bucket(bucketName)

	for _, name := range names {
		err = bucket.Object(name).Delete(ctx)
		if err != nil {
			errors.Wrapf(
				err,
				"failed to delete object with name %s under bucket %s. Please clean up manually",
				name,
				bucketName)
			// Return early
			return err
		}
	}

	return nil
}

// GetAllObjectNamesWithPrefix returns all object start with specified prefix
// under the specified bucket. If no prefix is supplied, it returns all objects
// under that bucket.
func GetAllObjectNamesWithPrefix(bucketName, prefix string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gCloudClientTimeout)
	defer cancel()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	bucket := client.Bucket(bucketName)
	var query *storage.Query
	if prefix != "" {
		query = &storage.Query{Prefix: prefix}
	}

	var names []string
	it := bucket.Objects(ctx, query)
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, errors.Wrapf(err, "failed while trying to list and traverse objects in bucket %s", bucketName)
		}
		names = append(names, attrs.Name)
	}

	return names, nil
}

// Read returns the content of the specified file under specified bucket
func Read(bucketName, objectName string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gCloudClientTimeout)
	defer cancel()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	reader, err := client.Bucket(bucketName).Object(objectName).NewReader(ctx)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// Copy copies the specified object into the specified target
func Copy(srcBucketName, srcObjectName, dstBucketName, dstObjectName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), gCloudClientTimeout)
	defer cancel()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}

	src := client.Bucket(srcBucketName).Object(srcObjectName)
	dst := client.Bucket(dstBucketName).Object(dstObjectName)

	if _, err := dst.CopierFrom(src).Run(ctx); err != nil {
		return errors.Wrapf(err, "failed while copying to %s from %s", dstObjectName, srcObjectName)
	}
	return nil
}

// GetAllPrefixesUnderBucketWithPrefix returns all the prefixes (sub-folders at the bottom most layer) within
// the specified bucket.
func GetAllPrefixesUnderBucketWithPrefix(bucketName, prefix string) ([]string, error) {
	objectNames, err := GetAllObjectNamesWithPrefix(bucketName, prefix)
	if err != nil {
		return nil, err
	}

	prefixes := make(map[string]struct{})
	for _, name := range objectNames {
		prefix := filepath.Dir(name)
		prefixes[prefix] = struct{}{}
	}

	result := make([]string, 0, len(prefixes))
	for prefix := range prefixes {
		result = append(result, prefix)
	}
	return result, nil
}
