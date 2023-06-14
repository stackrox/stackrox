package listener

import (
	"context"

	"github.com/cloudflare/cfssl/log"
	timestamp "github.com/gogo/protobuf/types"
	systemInfoStorage "github.com/stackrox/rox/central/systeminfo/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
)

// BackupListener provides functionality to listen on backup operations.
type BackupListener interface {
	OnBackupFail(ctx context.Context)
	OnBackupSuccess(ctx context.Context)
}

type backupListenerImpl struct {
	systemInfoStore systemInfoStorage.Store
	lock            sync.Mutex
}

func newBackupListener(systemInfoStore systemInfoStorage.Store) BackupListener {
	return &backupListenerImpl{
		systemInfoStore: systemInfoStore,
	}
}

func (l *backupListenerImpl) OnBackupFail(ctx context.Context) {
	l.lock.Lock()
	defer l.lock.Unlock()

	l.updateSystemInfo(ctx, storage.OperationStatus_FAIL)
}

func (l *backupListenerImpl) OnBackupSuccess(ctx context.Context) {
	l.lock.Lock()
	defer l.lock.Unlock()

	l.updateSystemInfo(ctx, storage.OperationStatus_PASS)
}

func (l *backupListenerImpl) updateSystemInfo(ctx context.Context, backupStatus storage.OperationStatus) {
	backupInfo := &storage.BackupInfo{
		BackupLastRunAt: timestamp.TimestampNow(),
		Status:          backupStatus,
		Requestor:       authn.UserFromContext(ctx),
	}

	// This is a system op.
	ctx = sac.WithAllAccess(context.Background())
	storedInfo, _, err := l.systemInfoStore.Get(ctx)
	if err != nil {
		log.Errorf("Could not store backup metadata: %v", err)
		return
	}

	if storedInfo == nil {
		storedInfo = &storage.SystemInfo{}
	}
	storedInfo.BackupInfo = backupInfo
	if err := l.systemInfoStore.Upsert(ctx, storedInfo); err != nil {
		log.Errorf("Could not store backup metadata: %v", err)
	}
}
