package version

import (
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/protoutils"
)

// ConvertVersionFromProto converts a `*storage.Version` to Gorm model
func ConvertVersionFromProto(obj *storage.Version) (*schema.Versions, error) {
	model := &schema.Versions{
		Version:       obj.GetVersion(),
		SeqNum:        obj.GetSeqNum(),
		MinSeqNum:     obj.GetMinSeqNum(),
		LastPersisted: protocompat.NilOrTime(obj.GetLastPersisted()),
	}
	return model, nil
}

// ConvertVersionToProto converts Gorm model `Versions` to its protobuf type object
func ConvertVersionToProto(m *schema.Versions) (*storage.Version, error) {
	var msg storage.Version

	// During the transition to not use serialized, we may be coming from a database
	// that uses it.  So if serialized is not nil, we will need to use that
	if m.Serialized != nil {
		if err := msg.UnmarshalVTUnsafe(m.Serialized); err != nil {
			return nil, err
		}
		return &msg, nil
	}

	msg = &storage.Version{}
	msg.SetVersion(m.Version)
	msg.SetSeqNum(m.SeqNum)
	msg.SetMinSeqNum(m.MinSeqNum)

	if m.LastPersisted != nil {
		ts := protoconv.MustConvertTimeToTimestamp(*m.LastPersisted)
		msg.SetLastPersisted(protoutils.RoundTimestamp(ts, time.Microsecond))
	}

	return &msg, nil
}
