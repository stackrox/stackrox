package creation

import (
	"fmt"

	"github.com/stackrox/rox/central/administration/events"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/administration/events/codes"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/logging"
)

var logger interface{ Warnw(string, ...any) } = logging.LoggerForModule(events.EnableAdministrationEvents())

func LogTokenCreation(id authn.Identity, md *storage.TokenMetadata) {
	if md == nil {
		md = &storage.TokenMetadata{}
	}

	fields := []any{
		logging.ErrCode(codes.APITokenCreated),
		logging.APITokenName(md.Name),
		logging.APITokenID(md.Id),
		logging.Strings("roles", md.Roles),
		logging.String("user_id", id.UID()),
	}
	if ap := id.ExternalAuthProvider(); ap != nil {
		fields = append(fields, logging.String("user_auth_provider",
			fmt.Sprintf("%s %s", ap.Type(), ap.Name())))
	}
	logger.Warnw("An API token has been created", fields...)
}
