package creation

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/administration/events/codes"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/logging"
)

func LogTokenCreation(log interface{ Infow(string, ...any) }, id authn.Identity, md *storage.TokenMetadata) {
	fields := []any{logging.ErrCode(codes.TokenCreated),
		logging.APITokenName(md.Name),
		logging.APITokenID(md.Id),
		logging.Strings("roles", md.Roles),
		logging.String("user", id.User().GetUsername()),
		logging.String("user_id", id.UID()),
	}

	if ap := id.ExternalAuthProvider(); ap != nil {
		fields = append(fields,
			logging.String("user_auth_provider",
				fmt.Sprintf("%s %q %s",
					ap.Type(), ap.Name(), ap.ID())))
	}
	log.Infow("An API token has been issued", fields...)
}
