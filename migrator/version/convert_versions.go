package version

import (
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/timestamp"
)

// ConvertVersionFromProto converts a `*storage.Version` to Gorm model
func ConvertVersionFromProto(obj *storage.Version) (*schema.Versions, error) {
	model := &schema.Versions{
		Version:       obj.GetVersion(),
		SeqNum:        obj.GetSeqNum(),
		MinSeqNum:     obj.GetMinSeqNum(),
		LastPersisted: pgutils.NilOrTime(obj.GetLastPersisted()),
	}
	return model, nil
}

// ConvertVersionToProto converts Gorm model `Versions` to its protobuf type object
func ConvertVersionToProto(m *schema.Versions) (*storage.Version, error) {
	var msg storage.Version

	// During the transition to not use serialized, we may be coming from a database
	// that uses it.  So if serialized is not nil, we will need to use that
	if m.Serialized != nil {
		if err := msg.Unmarshal(m.Serialized); err != nil {
			return nil, err
		}
		return &msg, nil
	}

	msg = storage.Version{
		Version:   m.Version,
		SeqNum:    m.SeqNum,
		MinSeqNum: m.MinSeqNum,
	}

	if m.LastPersisted != nil {
		ts := protoconv.MustConvertTimeToTimestamp(*m.LastPersisted)
		timestamp.RoundTimestamp(ts, time.Microsecond)
		msg.LastPersisted = ts
	}

	return &msg, nil
}
