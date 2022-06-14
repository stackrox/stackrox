package dackbox

import (
	"strconv"
	"testing"

	"github.com/stackrox/stackrox/pkg/dbhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	bucket1 = &dbhelper.BucketHandler{BucketPrefix: []byte("bucket1:")}
	bucket2 = &dbhelper.BucketHandler{BucketPrefix: []byte("bucket2:")}
	bucket3 = &dbhelper.BucketHandler{BucketPrefix: []byte("bucket3:")}
	bucket4 = &dbhelper.BucketHandler{BucketPrefix: []byte("bucket4:")}
)

func TestConcatenatePaths_Success(t *testing.T) {
	cases := []struct {
		Inputs   []BucketPath
		Expected BucketPath
	}{
		{
			Inputs:   []BucketPath{ForwardBucketPath(bucket1, bucket2, bucket3)},
			Expected: ForwardBucketPath(bucket1, bucket2, bucket3),
		},
		{
			Inputs:   []BucketPath{ForwardBucketPath(bucket1, bucket2), ForwardBucketPath(bucket2, bucket3)},
			Expected: ForwardBucketPath(bucket1, bucket2, bucket3),
		},
		{
			Inputs:   []BucketPath{ForwardBucketPath(bucket1), ForwardBucketPath(bucket1, bucket2), ForwardBucketPath(bucket2), ForwardBucketPath(bucket2, bucket3), ForwardBucketPath(bucket3)},
			Expected: ForwardBucketPath(bucket1, bucket2, bucket3),
		},
		{
			Inputs:   []BucketPath{BackwardsBucketPath(bucket1), ForwardBucketPath(bucket1, bucket2), BackwardsBucketPath(bucket2), ForwardBucketPath(bucket2, bucket3), BackwardsBucketPath(bucket3)},
			Expected: ForwardBucketPath(bucket1, bucket2, bucket3),
		},
	}

	for i, c := range cases {
		testCase := c
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			result, err := ConcatenatePaths(testCase.Inputs...)
			require.NoError(t, err)
			assert.Equal(t, testCase.Expected, result)
		})
	}
}

func TestConcatenatePaths_Errors(t *testing.T) {
	cases := [][]BucketPath{
		{},
		{BucketPath{}},
		{ForwardBucketPath(bucket1, bucket2), ForwardBucketPath(bucket3, bucket4)},
		{BackwardsBucketPath(bucket4, bucket3), BackwardsBucketPath(bucket2, bucket1)},
		{ForwardBucketPath(bucket1, bucket2), BackwardsBucketPath(bucket2, bucket3)},
		{BackwardsBucketPath(bucket4, bucket3), ForwardBucketPath(bucket3, bucket1)},
		{ForwardBucketPath(bucket1, bucket2), ForwardBucketPath(bucket3)},
		{ForwardBucketPath(bucket1, bucket2), BackwardsBucketPath(bucket3), BackwardsBucketPath(bucket3, bucket4)},
	}

	for i, c := range cases {
		testCase := c
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			_, err := ConcatenatePaths(testCase...)
			assert.Error(t, err)
		})
	}
}
