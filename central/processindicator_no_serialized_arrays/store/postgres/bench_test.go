//go:build sql_integration

package postgres

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
)

func generateArraysIndicator(deploymentID, podUID string) *storage.ProcessIndicatorNoSerializedArrays {
	return &storage.ProcessIndicatorNoSerializedArrays{
		Id:            uuid.NewV4().String(),
		DeploymentId:  deploymentID,
		ContainerName: "container-1",
		PodId:         "pod-name-1",
		PodUid:        podUID,
		ClusterId:     uuid.NewV4().String(),
		Namespace:     "default",
		Signal: &storage.ProcessSignalNoSerializedArrays{
			Id:           uuid.NewV4().String(),
			ContainerId:  "container-id-1",
			Name:         "/bin/bash",
			Args:         "-c echo hello",
			ExecFilePath: "/bin/bash",
			Uid:          1000,
			Gid:          1000,
			LineageInfo: []*storage.LineageInfoNoSerializedArrays{
				{ParentUid: 0, ParentExecFilePath: "/sbin/init"},
				{ParentUid: 1, ParentExecFilePath: "/bin/sh"},
			},
		},
	}
}

func BenchmarkArraysUpsertMany(b *testing.B) {
	ctx := sac.WithAllAccess(context.Background())
	db := pgtest.ForT(b)
	store := New(db.DB)

	// Disable foreign key checks for benchmark isolation
	_, err := db.Exec(ctx, "SET session_replication_role = replica")
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() {
		_, _ = db.Exec(ctx, "SET session_replication_role = DEFAULT")
	})

	for _, size := range []int{1, 10, 100, 500} {
		b.Run(fmt.Sprintf("x%d", size), func(b *testing.B) {
			objs := make([]*storage.ProcessIndicatorNoSerializedArrays, size)
			deploymentID := uuid.NewV4().String()
			podUID := uuid.NewV4().String()
			for i := range objs {
				objs[i] = generateArraysIndicator(deploymentID, podUID)
			}
			b.ResetTimer()
			for b.Loop() {
				if err := store.UpsertMany(ctx, objs); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkArraysGetSingle(b *testing.B) {
	ctx := sac.WithAllAccess(context.Background())
	db := pgtest.ForT(b)
	store := New(db.DB)

	// Disable foreign key checks for benchmark isolation
	_, err := db.Exec(ctx, "SET session_replication_role = replica")
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() {
		_, _ = db.Exec(ctx, "SET session_replication_role = DEFAULT")
	})

	obj := generateArraysIndicator(uuid.NewV4().String(), uuid.NewV4().String())
	if err := store.Upsert(ctx, obj); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for b.Loop() {
		_, _, err := store.Get(ctx, obj.GetId())
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkArraysGetMany(b *testing.B) {
	ctx := sac.WithAllAccess(context.Background())
	db := pgtest.ForT(b)
	store := New(db.DB)

	// Disable foreign key checks for benchmark isolation
	_, err := db.Exec(ctx, "SET session_replication_role = replica")
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() {
		_, _ = db.Exec(ctx, "SET session_replication_role = DEFAULT")
	})

	for _, size := range []int{10, 100, 500} {
		b.Run(fmt.Sprintf("x%d", size), func(b *testing.B) {
			objs := make([]*storage.ProcessIndicatorNoSerializedArrays, size)
			ids := make([]string, size)
			deploymentID := uuid.NewV4().String()
			podUID := uuid.NewV4().String()
			for i := range objs {
				objs[i] = generateArraysIndicator(deploymentID, podUID)
				ids[i] = objs[i].GetId()
			}
			if err := store.UpsertMany(ctx, objs); err != nil {
				b.Fatal(err)
			}

			b.ResetTimer()
			for b.Loop() {
				_, _, err := store.GetMany(ctx, ids)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
