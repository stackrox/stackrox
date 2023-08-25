// Package uuid is a wrapper for an external uuid library.
// It is to be used for all uuid's within stackrox.
package uuid

import (
	"crypto/sha256"
	"database/sql/driver"

	"github.com/gofrs/uuid"
)

// UUID in a universally unique identifier. The type is a wrapper around the uuid library.
type UUID struct {
	uuid uuid.UUID
}

// Nil UUID is special form of UUID that is specified to have all
// 128 bits set to zero.
var Nil = UUID{
	uuid: uuid.Nil,
}

// Equal returns true if u1 and u2 equals, otherwise returns false.
func Equal(u1 UUID, u2 UUID) bool {
	return u1.uuid == u2.uuid
}

// Bytes returns bytes slice representation of UUID.
func (u UUID) Bytes() []byte {
	return u.uuid.Bytes()
}

// String returns the canonical string representation of UUID:
// xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx.
func (u UUID) String() string {
	return u.uuid.String()
}

// MarshalText implements the encoding.TextMarshaler interface.
// The encoding is the same as returned by String.
func (u UUID) MarshalText() (text []byte, err error) {
	return u.uuid.MarshalText()
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
// Following formats are supported:
// "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
// "{6ba7b810-9dad-11d1-80b4-00c04fd430c8}",
// "urn:uuid:6ba7b810-9dad-11d1-80b4-00c04fd430c8"
func (u *UUID) UnmarshalText(text []byte) (err error) {
	return u.uuid.UnmarshalText(text)
}

// MarshalBinary implements the encoding.BinaryMarshaler interface.
func (u UUID) MarshalBinary() (data []byte, err error) {
	return u.uuid.MarshalBinary()
}

// UnmarshalBinary implements the encoding.BinaryUnmarshaler interface.
// It will return error if the slice isn't 16 bytes long.
func (u *UUID) UnmarshalBinary(data []byte) (err error) {
	return u.uuid.UnmarshalBinary(data)
}

// Value implements the driver.Valuer interface.
func (u UUID) Value() (driver.Value, error) {
	return u.uuid.Value()
}

// Scan implements the sql.Scanner interface.
// A 16-byte slice is handled by UnmarshalBinary, while
// a longer byte slice or a string is handled by UnmarshalText.
func (u *UUID) Scan(src interface{}) error {
	return u.uuid.Scan(src)
}

// FromBytes returns UUID converted from raw byte slice input.
// It will return error if the slice isn't 16 bytes long.
func FromBytes(input []byte) (u UUID, err error) {
	err = u.UnmarshalBinary(input)
	return
}

// FromBytesOrNil returns UUID converted from raw byte slice input.
// Same behavior as FromBytes, but returns a Nil UUID on error.
func FromBytesOrNil(input []byte) UUID {
	id, err := FromBytes(input)
	if err != nil {
		return Nil
	}
	return id
}

// FromString returns UUID parsed from string input.
// Input is expected in a form accepted by UnmarshalText.
func FromString(input string) (u UUID, err error) {
	err = u.UnmarshalText([]byte(input))
	return
}

// FromStringOrNil returns UUID parsed from string input.
// Same behavior as FromString, but returns a Nil UUID on error.
func FromStringOrNil(input string) UUID {
	id, err := FromString(input)
	if err != nil {
		return Nil
	}
	return id
}

// FromStringOrPanic returns UUID parsed from string input.
// If the provided string is invalid, function panics.
// This func should only be used for testing.
func FromStringOrPanic(input string) UUID {
	if id, err := FromString(input); err == nil {
		return id
	}

	panic("Provided string is not in valid uuid format")
}

// NewV4 returns random generated UUID.
func NewV4() UUID {
	return UUID{
		uuid: uuid.Must(uuid.NewV4()),
	}
}

// NewV5FromNonUUIDs is like NewV5, but accepts non-UUIDs for the name.
// It converts the name to a UUID using SHA-256 hashing.
// The output will be deterministic.
func NewV5FromNonUUIDs(ns, name string) UUID {
	nsSha256 := sha256.Sum256([]byte(ns))
	nsUUID, err := uuid.FromBytes(nsSha256[:16])
	// This should never error out since we're passing 16 bytes, as expected by UUID.
	if err != nil {
		panic(err)
	}
	return UUID{
		uuid: uuid.NewV5(nsUUID, name),
	}
}

// NewV5 returns UUID based on SHA-1 hash of namespace UUID and name.
func NewV5(ns UUID, name string) UUID {
	return UUID{
		uuid: uuid.NewV5(ns.uuid, name),
	}
}

// NewDummy returns a uuid for testing purposes
func NewDummy() UUID {
	return FromStringOrNil("aaaaaaaa-bbbb-4011-0000-111111111111")
}
