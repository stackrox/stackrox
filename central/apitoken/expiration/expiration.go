package expiration

import (
	"context"
	"fmt"
	"time"

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
	nofiticationInterval = 12 * time.Hour     // 12 hours
	staleNotificationAge = 24 * time.Hour     // 1 day
	expirationWindow     = 7 * 24 * time.Hour // 1 week

	// The timestamp format / layout is borrowed from `pkg/search/postgres/query/time_query.go`. It would be worth exporting.
	timestampLayout = "2006-01-02 15:04:05 -07:00"
)

var (
	log = logging.LoggerForModule()

	expirySearchCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Integration),
		),
	)
	updateTokenCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Integration),
		),
	)
)

// ExpirationNotifier is the interface for a background task that notifies about API token expiration
type ExpirationNotifier interface {
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
	go n.runExpiryNotifier()
}

func (n *expirationNotifierImpl) Stop() {
	n.stopper.Client().Stop()
	_ = n.stopper.Client().Stopped().Wait()
}

func (n *expirationNotifierImpl) runExpiryNotifier() {
	defer n.stopper.Flow().ReportStopped()

	n.checkAndNotifyExpirations()

	t := time.NewTicker(nofiticationInterval)
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

	expiringTokenIDs, err := n.listItemsToNotify(now, aboutToExpireDate, staleNotificationDate)
	if err != nil {
		log.Error(err)
		return
	}
	tokenIDs := convertSearchResultsToIDList(expiringTokenIDs)
	err = n.notify(tokenIDs)
	if err != nil {
		log.Error(err)
		return
	}
	for _, identifier := range tokenIDs {
		err := n.updateTokenNotificationTimestamp(identifier, now)
		if err != nil {
			log.Error(err)
		}
	}
}

func convertSearchResultsToIDList(results []search.Result) []string {
	tokenIDs := make([]string, 0, len(results))
	for _, r := range results {
		tokenIDs = append(tokenIDs, r.ID)
	}
	return tokenIDs
}

func (n *expirationNotifierImpl) listItemsToNotify(now time.Time, expiresUntil time.Time, notifiedUntil time.Time) ([]search.Result, error) {
	formattedNow := now.Format(timestampLayout)
	formattedExpiresUntil := expiresUntil.Format(timestampLayout)
	formattedNotifiedUntil := notifiedUntil.Format(timestampLayout)
	// Search tokens that expire before expiresUntil, that have not expired yet,
	// and that have not been notified since notifiedUntil.
	// That is Expiration < expiresUntil and LastNotified < notifiedUntil.
	queryNotExpired := search.NewQueryBuilder().
		AddStrings(search.Expiration, fmt.Sprintf(">%s", formattedNow)).
		ProtoQuery()
	queryAboutToExpire := search.NewQueryBuilder().
		AddStrings(search.Expiration, fmt.Sprintf("<%s", formattedExpiresUntil)).
		ProtoQuery()
	queryNotRecentlyNotified := search.NewQueryBuilder().
		AddStrings(search.LastNotified, fmt.Sprintf("<%s", formattedNotifiedUntil)).
		ProtoQuery()
	queryNotRevoked := search.NewQueryBuilder().
		AddBools(search.Revoked, false).
		ProtoQuery()
	query := search.ConjunctionQuery(
		queryNotExpired,
		queryAboutToExpire,
		queryNotRecentlyNotified,
		queryNotRevoked,
	)
	response, err := n.store.Search(expirySearchCtx, query)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (n *expirationNotifierImpl) notify(itemIDs []string) error {
	for _, identifier := range itemIDs {
		log.Warnf("API Token about to expire %s", identifier)
	}
	return nil
}

func (n *expirationNotifierImpl) updateTokenNotificationTimestamp(tokenID string, timestamp time.Time) error {
	token, err := n.store.GetTokenOrNil(expirySearchCtx, tokenID)
	if err != nil {
		return err
	}
	if token == nil {
		return nil
	}
	token.ExpirationNotifiedAt = protoconv.ConvertTimeToTimestamp(timestamp)
	return n.store.AddToken(updateTokenCtx, token)
}
