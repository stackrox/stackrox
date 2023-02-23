package expiration

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/apitoken/datastore"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
)

const (
	// notificationInterval = 1 * time.Hour      // 1 hour
	// staleNotificationAge = 24 * time.Hour     // 1 day
	// expirationWindow     = 7 * 24 * time.Hour // 1 week
	// expirationSlice      = 24 * time.Hour     // 1 day
	notificationInterval = 10 * time.Minute // 1 hour
	staleNotificationAge = 1 * time.Hour    // 1 day
	expirationWindow     = 6 * time.Hour    // 1 week
	expirationSlice      = 1 * time.Hour    // 1 day

	// The timestamp format / layout is borrowed from `pkg/search/postgres/query/time_query.go`. It would be worth exporting.
	timestampLayout = "01/02/2006 3:04:05 PM MST"
)

var (
	log = logging.LoggerForModule()

	expirySearchCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Integration),
		),
	)
)

// Notifier is the interface for a background task that notifies about API token expiration
type Notifier interface {
	Start()
	Stop()
}

// ExpiringItemNotifier is the interface to list the tokens about to expire, and to send notifications
// for items about to expire.
type ExpiringItemNotifier interface {
	ListItemsAboutToExpire() ([]search.Result, error)
	Notify(identifiersToNotify []string) error
}

type expirationNotifierImpl struct {
	store datastore.DataStore

	stopper concurrency.Stopper
}

func newExpirationNotifier(store datastore.DataStore) *expirationNotifierImpl {
	return &expirationNotifierImpl{
		store:   store,
		stopper: concurrency.NewStopper(),
	}
}

func (n *expirationNotifierImpl) Start() {
	go n.runExpirationNotifier()
}

func (n *expirationNotifierImpl) Stop() {
	n.stopper.Client().Stop()
	_ = n.stopper.Client().Stopped().Wait()
}

func (n *expirationNotifierImpl) runExpirationNotifier() {
	defer n.stopper.Flow().ReportStopped()

	n.checkAndNotifyExpirations()

	t := time.NewTicker(notificationInterval)
	for {
		select {
		case <-t.C:
			n.checkAndNotifyExpirations()
		case <-n.stopper.Flow().StopRequested():
			return
		}
	}
}

func (n *expirationNotifierImpl) checkAndNotifyExpirations() {
	// Only works in Postgres Mode
	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		return
	}

	now := time.Now()
	aboutToExpireDate := now.Add(expirationWindow)
	staleNotificationDate := now.Add(-staleNotificationAge)
	staleNotificationTimestamp := protoconv.ConvertTimeToTimestamp(staleNotificationDate)

	scheduleCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Notifications),
		),
	)
	var notificationSchedule *storage.NotificationSchedule
	notificationSchedule, found, err := n.store.GetNotificationSchedule(scheduleCtx)
	if err != nil {
		log.Error(err)
		return
	}
	if !found || notificationSchedule == nil {
		notificationSchedule = &storage.NotificationSchedule{
			LastRun: protoconv.ConvertTimeToTimestamp(staleNotificationDate.Add(-1 * time.Hour)),
		}
	}
	if notificationSchedule.GetLastRun().Compare(staleNotificationTimestamp) >= 0 {
		return
	}
	notificationSchedule.LastRun = protoconv.ConvertTimeToTimestamp(now)
	err = n.store.UpsertNotificationSchedule(scheduleCtx, notificationSchedule)
	if err != nil {
		log.Error(errors.Wrap(err, "failed to update expired API token notification last run"))
	}

	expiringTokens, err := n.listItemsToNotify(now, aboutToExpireDate)
	if err != nil {
		log.Error(err)
		return
	}
	err = n.notify(expiringTokens)
	if err != nil {
		log.Error(err)
		return
	}
}

func (n *expirationNotifierImpl) listItemsToNotify(now time.Time, expiresUntil time.Time) ([]*storage.TokenMetadata, error) {
	formattedNow := now.Format(timestampLayout)
	formattedExpiresUntil := expiresUntil.Format(timestampLayout)
	// Search tokens that expire before expiresUntil, that have not expired yet,
	// and that have not been notified since notifiedUntil.
	// That is Expiration < expiresUntil and LastNotified < notifiedUntil.
	queryNotExpired := search.NewQueryBuilder().
		AddStrings(search.Expiration, fmt.Sprintf(">%s", formattedNow)).
		ProtoQuery()
	queryAboutToExpire := search.NewQueryBuilder().
		AddStrings(search.Expiration, fmt.Sprintf("<%s", formattedExpiresUntil)).
		ProtoQuery()
	queryNotRevoked := search.NewQueryBuilder().
		AddBools(search.Revoked, false).
		ProtoQuery()
	query := search.ConjunctionQuery(
		queryNotExpired,
		queryAboutToExpire,
		queryNotRevoked,
	)
	response, err := n.store.SearchRawTokens(expirySearchCtx, query)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (n *expirationNotifierImpl) notify(items []*storage.TokenMetadata) error {
	now := time.Now()
	for _, token := range items {
		expiration := protoconv.ConvertTimestampToTimeOrNow(token.GetExpiration())
		ttl := expiration.Sub(now)
		timeUnit := int(expirationSlice.Seconds())
		ttlSeconds := int(ttl.Seconds())
		sliceCount := ttlSeconds / timeUnit
		if ttlSeconds%timeUnit != 0 {
			sliceCount++
		}
		sliceDuration := "hours"
		if sliceCount == 1 {
			sliceDuration = "hour"
		}
		log.Warnf("API Token %s (ID %s) will expire in less than %d %s.", token.GetName(), token.GetId(), sliceCount, sliceDuration)
	}
	return nil
}
