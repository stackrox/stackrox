package expiration

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/administration/events"
	"github.com/stackrox/rox/central/apitoken/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
)

const (
	// The timestamp format / layout is borrowed from `pkg/search/postgres/query/time_query.go`. It would be worth exporting.
	timestampLayout = "01/02/2006 3:04:05 PM MST"
)

var (
	log = logging.LoggerForModule(events.EnableAdministrationEvents())

	expirySearchCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Integration),
		),
	)

	scheduleCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Notifications),
		),
	)
)

// TokenExpirationLoop is the interface for a background task that notifies about API token expiration
type TokenExpirationLoop interface {
	Start()
	Stop()
}

// TokenExpirationNotifier is the interface for any mechanism that notifies that API Tokens are about to expire
type TokenExpirationNotifier interface {
	Notify([]*storage.TokenMetadata) error
}

type expirationNotifierImpl struct {
	store datastore.DataStore

	stopper concurrency.Stopper

	notificationTicker *time.Ticker

	notifier TokenExpirationNotifier
}

func newExpirationNotifier(store datastore.DataStore) *expirationNotifierImpl {
	return &expirationNotifierImpl{
		store:    store,
		stopper:  concurrency.NewStopper(),
		notifier: &logExpirationNotifier{},
	}
}

func (n *expirationNotifierImpl) Start() {
	n.notificationTicker = time.NewTicker(env.APITokenExpirationNotificationInterval.DurationSetting())
	go n.runExpirationNotifier()
}

func (n *expirationNotifierImpl) Stop() {
	n.stopper.Client().Stop()
	err := n.stopper.Client().Stopped().Wait()
	if err != nil {
		log.Error("Error stopping API Token expiration loop: ", err)
	}
}

func (n *expirationNotifierImpl) runExpirationNotifier() {
	defer n.stopper.Flow().ReportStopped()

	n.checkAndNotifyExpirations()

	defer n.notificationTicker.Stop()
	for {
		select {
		case <-n.notificationTicker.C:
			n.checkAndNotifyExpirations()
		case <-n.stopper.Flow().StopRequested():
			return
		}
	}
}

func (n *expirationNotifierImpl) checkAndNotifyExpirations() {
	now := time.Now()
	aboutToExpireDate := now.Add(env.APITokenExpirationExpirationWindow.DurationSetting())
	staleNotificationDate := now.Add(-env.APITokenExpirationStaleNotificationAge.DurationSetting())
	staleNotificationTimestamp := protoconv.ConvertTimeToTimestamp(staleNotificationDate)

	notificationSchedule, found, err := n.store.GetNotificationSchedule(scheduleCtx)
	if err != nil {
		log.Error("Failed to retrieve API Notification schedule information: ", err)
		return
	}
	if !found || notificationSchedule == nil {
		notificationSchedule = &storage.NotificationSchedule{
			LastRun: protoconv.ConvertTimeToTimestamp(staleNotificationDate.Add(-1 * time.Hour)),
		}
	}
	// API Token expiration should be notified at regular and long intervals.
	// The check below is there to enforce a notification back-off mechanism.
	// The notification schedule stored in database keeps track of when the last notification run took place.
	// The staleNotificationDate and staleNotificationTimestamp contain the point in time until which a notification
	// is considered stale. If the last notification occurred before then, the back-off window has run out and a new
	// notification can be sent. Otherwise, nothing needs to be done (at least) until the next loop cycle.
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
		log.Error("Failed to list API Tokens about to expire: ", err)
		return
	}
	err = n.notifier.Notify(expiringTokens)
	if err != nil {
		log.Error("Failed to send notifications for API Tokens about to expire: ", err)
		return
	}
}

func (n *expirationNotifierImpl) listItemsToNotify(now time.Time, expiresUntil time.Time) ([]*storage.TokenMetadata, error) {
	formattedNow := now.Format(timestampLayout)
	formattedExpiresUntil := expiresUntil.Format(timestampLayout)
	// Search tokens that expire before expiresUntil, that have not expired yet,
	// and that have not been revoked.
	// That is now < Expiration < expiresUntil and Revoked = false.
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

type logExpirationNotifier struct{}

func generateExpiringTokenLog(token *storage.TokenMetadata, now time.Time, expirationSliceDuration time.Duration, sliceName string) string {
	expiration := protoconv.ConvertTimestampToTimeOrNow(token.GetExpiration())
	ttl := expiration.Sub(now)
	timeUnit := int(expirationSliceDuration.Seconds())
	ttlSeconds := int(ttl.Seconds())
	sliceCount := ttlSeconds / timeUnit
	if ttlSeconds%timeUnit != 0 {
		sliceCount++
	}
	sliceDuration := sliceName
	if sliceCount != 1 {
		sliceDuration = sliceName + "s"
	}
	return fmt.Sprintf("API Token will expire in less than %d %s", sliceCount, sliceDuration)
}

func (n *logExpirationNotifier) Notify(items []*storage.TokenMetadata) error {
	now := time.Now()
	expirationSliceDuration := env.APITokenExpirationExpirationSlice.DurationSetting()
	expirationSliceName := env.APITokenExpirationExpirationSliceName.Setting()
	for _, token := range items {
		log.Warnw(generateExpiringTokenLog(token, now, expirationSliceDuration, expirationSliceName),
			logging.APITokenName(token.GetName()), logging.APITokenID(token.GetId()))
	}
	return nil
}
