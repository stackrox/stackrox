package testutils

import (
	"context"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

// region Read Operations

// ========================================
// Read Operations
// ========================================

func RunGetTests[T protoassert.Message[V], V any](
	t *testing.T,
	testCases map[string]SACCrudTestCase,
	testContexts map[string]context.Context,
	objectIDExtractor func(T) string,
	objectCreator func() T,
	objectInjector func(context.Context, T) error,
	objectGetter func(context.Context, string) (T, bool, error),
	objectRemover func(context.Context, string) error,
) {
	testObject, objID, unrestrictedCtx := injectTestObject(t, objectIDExtractor, objectCreator, objectInjector)

	// Cleanup after all test cases run
	// Use best-effort removal in case TearDownTest already cleaned up
	t.Cleanup(func() {
		_ = objectRemover(unrestrictedCtx, objID)
	})

	for testName, testCase := range testCases {
		t.Run(testName, func(it *testing.T) {
			ctx := testContexts[testCase.ScopeKey]
			fetched, found, err := objectGetter(ctx, objID)
			assertExpectedError(it, testCase, err)
			if testCase.ExpectedFound {
				assert.True(it, found)
				protoassert.Equal(it, testObject, fetched)
			} else {
				assert.False(it, found)
				assert.Nil(it, fetched)
			}
		})
	}
}

func RunExistsTests[T protoassert.Message[V], V any](
	t *testing.T,
	testCases map[string]SACCrudTestCase,
	testContexts map[string]context.Context,
	objectIDExtractor func(T) string,
	objectCreator func() T,
	objectInjector func(context.Context, T) error,
	objectExists func(context.Context, string) (bool, error),
	objectRemover func(context.Context, string) error,
) {
	_, objID, unrestrictedCtx := injectTestObject(t, objectIDExtractor, objectCreator, objectInjector)

	// Cleanup after all test cases run
	// Use best-effort removal in case TearDownTest already cleaned up
	t.Cleanup(func() {
		_ = objectRemover(unrestrictedCtx, objID)
	})

	for testName, testCase := range testCases {
		t.Run(testName, func(it *testing.T) {
			ctx := testContexts[testCase.ScopeKey]
			exists, err := objectExists(ctx, objID)

			assertExpectedError(it, testCase, err)

			if testCase.ExpectedFound {
				assert.True(it, exists, "Expected object to exist with proper access")
			} else {
				// When access is denied, Exists returns (false, nil)
				assert.False(it, exists, "Expected exists to be false when access is denied")
			}
		})
	}
}

// RunGetAllTests tests a GetAll operation against different SAC contexts.
// It creates multiple test objects once, then runs all test cases against the same dataset.
// For contexts with access, it verifies all objects are returned using protoassert.ElementsMatch.
// For contexts without access, it verifies no objects are returned.
func RunGetAllTests[T protoassert.Message[V], V any](
	t *testing.T,
	testCases map[string]SACCrudTestCase,
	testContexts map[string]context.Context,
	objectIDExtractor func(T) string,
	objectCreator func() T,
	objectInjector func(context.Context, T) error,
	objectGetAll func(context.Context) ([]T, error),
	objectRemover func(context.Context, string) error,
) {
	const numObjects = 3

	// Setup: Create multiple test objects once with unrestricted access
	unrestrictedCtx := sac.WithAllAccess(context.Background())
	var createdObjects []T
	var objectIDs []string

	for i := 0; i < numObjects; i++ {
		obj := objectCreator()
		err := objectInjector(unrestrictedCtx, obj)
		require.NoError(t, err)
		createdObjects = append(createdObjects, obj)
		objectIDs = append(objectIDs, objectIDExtractor(obj))
	}

	// Cleanup after all test cases run
	t.Cleanup(func() {
		for _, id := range objectIDs {
			_ = objectRemover(unrestrictedCtx, id)
		}
	})

	// Run all subtests against the same data set
	for testName, testCase := range testCases {
		t.Run(testName, func(it *testing.T) {
			ctx := testContexts[testCase.ScopeKey]
			results, err := objectGetAll(ctx)

			// Read operations with globally scoped store don't return errors
			assert.NoError(it, err)
			if testCase.ExpectedFound {
				protoassert.ElementsMatch(it, createdObjects, results)
			} else {
				assert.Empty(it, results, "Expected no objects when access is denied")
			}
		})
	}
}

func RunGetFilteredTests[T protoassert.Message[V], V any](
	t *testing.T,
	testCases map[string]SACCrudTestCase,
	testContexts map[string]context.Context,
	objectIDExtractor func(T) string,
	objectCreator func() T,
	objectInjector func(context.Context, T) error,
	objectGetFiltered func(context.Context, func(T) bool) ([]T, error),
	objectRemover func(context.Context, string) error,
) {
	// Setup: Create and insert multiple test objects with unrestricted access
	unrestrictedCtx := sac.WithAllAccess(context.Background())
	const numObjects = 3
	objectIDs := make([]string, 0, numObjects)

	for i := 0; i < numObjects; i++ {
		testObject := objectCreator()
		err := objectInjector(unrestrictedCtx, testObject)
		require.NoError(t, err)
		objID := objectIDExtractor(testObject)
		objectIDs = append(objectIDs, objID)
	}

	// Cleanup after all test cases run
	t.Cleanup(func() {
		for _, objID := range objectIDs {
			_ = objectRemover(unrestrictedCtx, objID)
		}
	})

	for testName, testCase := range testCases {
		t.Run(testName, func(it *testing.T) {
			ctx := testContexts[testCase.ScopeKey]
			// Filter to get all objects
			results, err := objectGetFiltered(ctx, func(obj T) bool {
				return true
			})

			assertExpectedError(it, testCase, err)
			if testCase.ExpectedFound {
				assert.GreaterOrEqual(it, len(results), numObjects, "Expected to find at least %d objects", numObjects)
			} else {
				assert.Empty(it, results, "Expected empty results when access is denied")
			}
		})
	}
}

func RunForEachTests[T protoassert.Message[V], V any](
	t *testing.T,
	testCases map[string]SACCrudTestCase,
	testContexts map[string]context.Context,
	objectIDExtractor func(T) string,
	objectCreator func() T,
	objectInjector func(context.Context, T) error,
	objectIterator func(context.Context, func(T) error) error,
	objectRemover func(context.Context, string) error,
) {
	// Setup: Create and insert a test object with unrestricted access
	_, objID, unrestrictedCtx := injectTestObject(t, objectIDExtractor, objectCreator, objectInjector)

	// Cleanup after all test cases run
	t.Cleanup(func() {
		_ = objectRemover(unrestrictedCtx, objID)
	})

	for testName, testCase := range testCases {
		t.Run(testName, func(it *testing.T) {
			ctx := testContexts[testCase.ScopeKey]
			var count int
			err := objectIterator(ctx, func(obj T) error {
				count++
				return nil
			})

			assertExpectedError(it, testCase, err)
			// When access is allowed and expected to find results, we should see at least the test object
			if testCase.ExpectedFound {
				assert.GreaterOrEqual(it, count, 1, "Expected to iterate over at least one object")
			} else {
				// When no results are expected (e.g., no access), count should be 0
				assert.Equal(it, 0, count, "Expected to iterate over zero objects")
			}
		})
	}
}

// endregion Read Operations

// region Search Operations

// ========================================
// Search Operations
// ========================================

// RunCountTests tests a Count operation against different SAC contexts.
// It creates a single test object once, then runs all test cases against the same dataset.
// For contexts with access, it verifies the count equals 1.
// For contexts without access, it verifies the count equals 0.
func RunCountTests[T protoassert.Message[V], V any](
	t *testing.T,
	testCases map[string]SACCrudTestCase,
	testContexts map[string]context.Context,
	objectIDExtractor func(T) string,
	objectCreator func() T,
	objectInjector func(context.Context, T) error,
	objectCounter func(context.Context, *v1.Query) (int, error),
	objectRemover func(context.Context, string) error,
) {
	// Setup: Create a single test object with unrestricted access
	_, objID, unrestrictedCtx := injectTestObject(t, objectIDExtractor, objectCreator, objectInjector)

	// Cleanup after all test cases run
	t.Cleanup(func() {
		_ = objectRemover(unrestrictedCtx, objID)
	})

	// Run all subtests against the same data set
	for testName, testCase := range testCases {
		t.Run(testName, func(it *testing.T) {
			ctx := testContexts[testCase.ScopeKey]
			count, err := objectCounter(ctx, &v1.Query{})

			// Read operations with globally scoped store don't return errors
			assert.NoError(it, err)
			if testCase.ExpectedFound {
				assert.Equal(it, 1, count, "Expected count to be exactly 1")
			} else {
				assert.Equal(it, 0, count, "Expected count to be 0 when access is denied")
			}
		})
	}
}

// RunSearchTests tests a Search operation against different SAC contexts.
// It creates a single test object once, then runs all test cases against the same dataset.
// For contexts with access, it verifies exactly 1 search result is returned.
// For contexts without access, it verifies no search results are returned.
func RunSearchTests[T protoassert.Message[V], V any](
	t *testing.T,
	testCases map[string]SACCrudTestCase,
	testContexts map[string]context.Context,
	objectIDExtractor func(T) string,
	objectCreator func() T,
	objectInjector func(context.Context, T) error,
	objectSearcher func(context.Context, *v1.Query) ([]search.Result, error),
	objectRemover func(context.Context, string) error,
) {
	// Setup: Create a single test object with unrestricted access
	_, objID, unrestrictedCtx := injectTestObject(t, objectIDExtractor, objectCreator, objectInjector)

	// Cleanup after all test cases run
	t.Cleanup(func() {
		_ = objectRemover(unrestrictedCtx, objID)
	})

	// Run all subtests against the same data set
	for testName, testCase := range testCases {
		t.Run(testName, func(it *testing.T) {
			ctx := testContexts[testCase.ScopeKey]
			results, err := objectSearcher(ctx, &v1.Query{})

			// Read operations with globally scoped store don't return errors
			assert.NoError(it, err)
			if testCase.ExpectedFound {
				assert.Equal(it, 1, len(results), "Expected to find exactly one result")
			} else {
				assert.Empty(it, results, "Expected no results when access is denied")
			}
		})
	}
}

// RunSearchResultsTests tests a Search operation that returns v1.SearchResult against different SAC contexts.
// It creates a single test object once, then runs all test cases against the same dataset.
// For contexts with access, it verifies exactly 1 search result is returned.
// For contexts without access, it verifies no search results are returned.
func RunSearchResultsTests[T protoassert.Message[V], V any](
	t *testing.T,
	testCases map[string]SACCrudTestCase,
	testContexts map[string]context.Context,
	objectIDExtractor func(T) string,
	objectCreator func() T,
	objectInjector func(context.Context, T) error,
	objectSearcher func(context.Context, *v1.Query) ([]*v1.SearchResult, error),
	objectRemover func(context.Context, string) error,
) {
	// Setup: Create a single test object with unrestricted access
	_, objID, unrestrictedCtx := injectTestObject(t, objectIDExtractor, objectCreator, objectInjector)

	// Cleanup after all test cases run
	t.Cleanup(func() {
		_ = objectRemover(unrestrictedCtx, objID)
	})

	// Run all subtests against the same data set
	for testName, testCase := range testCases {
		t.Run(testName, func(it *testing.T) {
			ctx := testContexts[testCase.ScopeKey]
			results, err := objectSearcher(ctx, &v1.Query{})

			// Read operations with globally scoped store don't return errors
			assert.NoError(it, err)
			if testCase.ExpectedFound {
				assert.Equal(it, 1, len(results), "Expected to find exactly one result")
			} else {
				assert.Empty(it, results, "Expected no results when access is denied")
			}
		})
	}
}

// endregion Search Operations

// region Write Operations

// ========================================
// Write Operations
// ========================================

func RunUpsertTests[T protoassert.Message[V], V any](
	t *testing.T,
	testCases map[string]SACCrudTestCase,
	testContexts map[string]context.Context,
	objectIDExtractor func(T) string,
	objectCreator func() T,
	objectInjector func(context.Context, T) error,
	mutateObject func(T) T,
	objectGetter func(context.Context, string) (T, bool, error),
	objectRemover func(context.Context, string) error,
) {
	for testName, testCase := range testCases {
		t.Run(testName, func(it *testing.T) {
			// Create a new object for each test case
			testObject := objectCreator()

			// Try to upsert with the test context
			ctx := testContexts[testCase.ScopeKey]
			err := objectInjector(ctx, testObject)

			// Check upsert result
			assertExpectedError(it, testCase, err)

			// Verify the outcome by checking if object exists with unrestricted access
			unrestrictedCtx := sac.WithAllAccess(context.Background())
			objID := objectIDExtractor(testObject)

			fetched, found, err := objectGetter(unrestrictedCtx, objID)
			require.NoError(it, err)

			if testCase.ExpectError {
				// Upsert should have failed, object should NOT exist
				assert.False(it, found)
				assert.Nil(it, fetched)
			} else {
				// Upsert should have succeeded, object should exist
				assert.True(it, found)
				protoassert.Equal(it, testObject, fetched)

				// Cleanup: remove the object
				err := objectRemover(unrestrictedCtx, objID)
				require.NoError(it, err)
			}
		})

		// UpsertOverwrite test: Upsert an existing object to verify overwrite behavior
		t.Run(testName+"/UpsertOverwrite", func(it *testing.T) {
			// First, create and upsert an object with unrestricted access
			initialObject, objID, unrestrictedCtx := injectTestObject(it, objectIDExtractor, objectCreator, objectInjector)

			// Cleanup after test
			defer func() {
				_ = objectRemover(unrestrictedCtx, objID)
			}()

			// Clone the original object to preserve it for comparison
			originalObject := proto.Clone(initialObject).(T)

			// Mutate the object before upserting again
			mutatedObject := mutateObject(initialObject)

			// Try to upsert (overwrite) with the test context
			ctx := testContexts[testCase.ScopeKey]
			err := objectInjector(ctx, mutatedObject)

			// Check upsert result
			assertExpectedError(it, testCase, err)

			// Verify the outcome by checking if object was updated with unrestricted access
			fetched, found, err := objectGetter(unrestrictedCtx, objID)
			require.NoError(it, err)
			require.True(it, found)

			if testCase.ExpectError {
				// Upsert should have failed, object should NOT be mutated (should match original)
				assert.NotNil(it, fetched)
				protoassert.Equal(it, originalObject, fetched)
			} else {
				// Upsert should have succeeded, object should be mutated
				assert.NotNil(it, fetched)
				protoassert.Equal(it, mutatedObject, fetched)
			}
		})
	}
}

func RunRemoveTests[T protoassert.Message[V], V any](
	t *testing.T,
	testCases map[string]SACCrudTestCase,
	testContexts map[string]context.Context,
	objectIDExtractor func(T) string,
	objectCreator func() T,
	objectInjector func(context.Context, T) error,
	objectGetter func(context.Context, string) (T, bool, error),
	objectRemover func(context.Context, string) error,
) {
	for testName, testCase := range testCases {
		t.Run(testName, func(it *testing.T) {
			// Setup: Create and upsert a new object with unrestricted access
			testObject, objID, unrestrictedCtx := injectTestObject(it, objectIDExtractor, objectCreator, objectInjector)

			// Try to remove with the test context
			ctx := testContexts[testCase.ScopeKey]
			err := objectRemover(ctx, objID)

			// Check removal result
			assertExpectedError(it, testCase, err)

			// Verify the outcome by checking if object exists with unrestricted access
			fetched, found, err := objectGetter(unrestrictedCtx, objID)
			require.NoError(it, err)

			if testCase.ExpectError {
				// Removal should have failed, object should still exist
				assert.True(it, found)
				assert.NotNil(it, fetched)
				protoassert.Equal(it, testObject, fetched)

				// Cleanup: remove the object with unrestricted access
				err := objectRemover(unrestrictedCtx, objID)
				require.NoError(it, err)
			} else {
				// Removal should have succeeded, object should NOT exist
				assert.False(it, found)
				assert.Nil(it, fetched)
			}
		})
	}
}

func RunAddTests[T protoassert.Message[V], V any](
	t *testing.T,
	testCases map[string]SACCrudTestCase,
	testContexts map[string]context.Context,
	objectIDExtractor func(T) string,
	objectCreator func() T,
	objectAdder func(context.Context, T) error,
	objectGetter func(context.Context, string) (T, bool, error),
	objectRemover func(context.Context, string) error,
) {
	for testName, testCase := range testCases {
		t.Run(testName, func(it *testing.T) {
			// Create a new object for each test case
			testObject := objectCreator()

			// Try to add with the test context
			ctx := testContexts[testCase.ScopeKey]
			err := objectAdder(ctx, testObject)

			// Check add result
			assertExpectedError(it, testCase, err)

			// Verify the outcome by checking if object exists with unrestricted access
			unrestrictedCtx := sac.WithAllAccess(context.Background())
			objID := objectIDExtractor(testObject)

			fetched, found, err := objectGetter(unrestrictedCtx, objID)
			require.NoError(it, err)

			if testCase.ExpectError {
				// Add should have failed, object should NOT exist
				assert.False(it, found)
				assert.Nil(it, fetched)
			} else {
				// Add should have succeeded, object should exist
				assert.True(it, found)
				protoassert.Equal(it, testObject, fetched)

				// Cleanup: remove the object
				err := objectRemover(unrestrictedCtx, objID)
				require.NoError(it, err)
			}
		})

		// AddOnExisting test: Try to add an object that already exists
		t.Run(testName+"/AddOnExisting", func(it *testing.T) {
			// First, create and add an object with unrestricted access
			existingObject, objID, unrestrictedCtx := injectTestObject(it, objectIDExtractor, objectCreator, objectAdder)

			// Cleanup after test
			defer func() {
				_ = objectRemover(unrestrictedCtx, objID)
			}()

			// Now try to add it again with the test context
			ctx := testContexts[testCase.ScopeKey]
			err := objectAdder(ctx, existingObject)

			// Check add result
			if testCase.ExpectError {
				// When there's no access, should still get access denied error
				assertExpectedError(it, testCase, err)
			} else {
				// When there is access, add should fail but with a different error (not access denied)
				assert.Error(it, err, "Adding duplicate object should fail")
				assert.NotErrorIs(it, err, sac.ErrResourceAccessDenied, "Error should not be ErrResourceAccessDenied for duplicate add")
			}
		})
	}
}

func RunUpdateTests[T protoassert.Message[V], V any](
	t *testing.T,
	testCases map[string]SACCrudTestCase,
	testContexts map[string]context.Context,
	objectIDExtractor func(T) string,
	objectCreator func() T,
	objectInjector func(context.Context, T) error,
	mutateObject func(T) T,
	objectUpdater func(context.Context, T) error,
	objectGetter func(context.Context, string) (T, bool, error),
	objectRemover func(context.Context, string) error,
) {
	for testName, testCase := range testCases {
		t.Run(testName, func(it *testing.T) {
			// Setup: Create and upsert a new object with unrestricted access
			testObject, objID, unrestrictedCtx := injectTestObject(it, objectIDExtractor, objectCreator, objectInjector)

			// Clone the original object to preserve it for comparison
			originalObject := proto.Clone(testObject).(T)

			// Mutate the object before updating
			mutatedObject := mutateObject(testObject)

			// Try to update with the test context
			ctx := testContexts[testCase.ScopeKey]
			err := objectUpdater(ctx, mutatedObject)

			// Check update result
			assertExpectedError(it, testCase, err)

			// Verify the outcome by checking if object was updated with unrestricted access
			fetched, found, err := objectGetter(unrestrictedCtx, objID)
			require.NoError(it, err)
			require.True(it, found)

			if testCase.ExpectError {
				// Update should have failed, object should NOT be mutated (should match original)
				assert.NotNil(it, fetched)
				protoassert.Equal(it, originalObject, fetched)
			} else {
				// Update should have succeeded, object should be mutated
				assert.NotNil(it, fetched)
				protoassert.Equal(it, mutatedObject, fetched)
			}

			// Cleanup: remove the object with unrestricted access
			err = objectRemover(unrestrictedCtx, objID)
			require.NoError(it, err)
		})
	}
}

// endregion Write Operations

// region Helpers

// ========================================
// Helper Functions
// ========================================

// assertExpectedError validates that the error matches the test case expectations.
// If the test case expects an error, it asserts that err is not nil and optionally
// matches the expected error. If the test case does not expect an error, it asserts
// that err is nil.
func assertExpectedError(t *testing.T, testCase SACCrudTestCase, err error) {
	if testCase.ExpectError {
		assert.Error(t, err)
		if testCase.ExpectedError != nil {
			assert.ErrorIs(t, err, testCase.ExpectedError)
		}
	} else {
		assert.NoError(t, err)
	}
}

// injectTestObject creates a test object, injects it using an unrestricted context,
// and returns the object, its ID, and the unrestricted context for cleanup purposes.
func injectTestObject[T protoassert.Message[V], V any](
	t *testing.T,
	objectIDExtractor func(T) string,
	objectCreator func() T,
	objectInjector func(context.Context, T) error,
) (T, string, context.Context) {
	testObject := objectCreator()
	unrestrictedCtx := sac.WithAllAccess(context.Background())
	require.NoError(t, objectInjector(unrestrictedCtx, testObject))
	objID := objectIDExtractor(testObject)
	return testObject, objID, unrestrictedCtx
}

// endregion Helpers
