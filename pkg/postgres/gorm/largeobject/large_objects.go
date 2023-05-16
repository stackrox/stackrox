package largeobject

import (
	"errors"
	"io"

	"gorm.io/gorm"
)

// LargeObjects is a structure used to access the large objects API. It is only valid within the transaction where it
// was created.
//
// For more details see: http://www.postgresql.org/docs/current/static/largeobjects.html
type LargeObjects struct {
	tx *gorm.DB
}

type Mode int32

const (
	ModeWrite Mode = 0x20000
	ModeRead  Mode = 0x40000
)

// Create creates a new large object with an unused OID assigned
func (o *LargeObjects) Create() (uint32, error) {
	o.tx = o.tx.Raw("SELECT lo_create(?)", 0)
	if err := o.tx.Error; err != nil {
		return 0, err
	}
	var oid uint32
	o.tx = o.tx.Scan(&oid)
	return oid, o.tx.Error
}

// Open opens an existing large object with the given mode. ctx will also be used for all operations on the opened large
// object.
func (o *LargeObjects) Open(oid uint32, mode Mode) (*LargeObject, error) {
	var fd int32
	o.tx = o.tx.Raw("select lo_open(?, ?)", oid, mode).Scan(&fd)
	if err := o.tx.Error; err != nil {
		return nil, err
	}
	return &LargeObject{fd: fd, tx: o.tx}, nil
}

// Unlink removes a large object from the database.
func (o *LargeObjects) Unlink(oid uint32) error {
	var count int32
	o.tx = o.tx.Raw("select lo_unlink(?)", oid).Scan(&count)
	if err := o.tx.Error; err != nil {
		return err
	}
	if count != 1 {
		return errors.New("failed to remove large object")
	}
	return nil
}

func (o *LargeObjects) Upsert(oid uint32, r io.Reader) error {
	obj, err := o.Open(oid, ModeWrite)
	if err != nil {
		return err
	} /*
		err = obj.Truncate(1)
		if err != nil {
			return err
		}
		if _, err = obj.Seek(0, io.SeekStart); err != nil {
			return err
		}*/
	_, err = io.Copy(obj, r)

	return obj.Close()
}

func (o *LargeObjects) Get(oid uint32, w io.Writer) error {
	obj, err := o.Open(oid, ModeRead)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, obj)
	return err
}

// A LargeObject is a large object stored on the server. It is only valid within the transaction that it was initialized
// in. It uses the context it was initialized with for all operations. It implements these interfaces:
//
//	io.Writer
//	io.Reader
//	io.Seeker
//	io.Closer
type LargeObject struct {
	tx *gorm.DB
	fd int32
}

// Write writes p to the large object and returns the number of bytes written and an error if not all of p was written.
func (o *LargeObject) Write(p []byte) (int, error) {
	var n int
	o.tx = o.tx.Raw("select lowrite($1, $2)", o.fd, p)
	if err := o.tx.Error; err != nil {
		return n, err
	}
	if err := o.tx.Scan(&n).Error; err != nil {
		return n, err
	}
	if err := o.tx.Error; err != nil {
		return n, err
	}

	if n < 0 {
		return 0, errors.New("failed to write to large object")
	}

	return n, nil
}

// Read reads up to len(p) bytes into p returning the number of bytes read.
func (o *LargeObject) Read(p []byte) (n int, err error) {
	var res []byte = make([]byte, 0, len(p))
	o.tx = o.tx.Raw("select loread($1, $2)", o.fd, len(p))
	if err = o.tx.Error; err != nil {
		return 0, err
	}
	if err = o.tx.Row().Scan(&res); err != nil {
		return 0, err
	}
	copy(p, res)
	if err = o.tx.Error; err != nil {
		return len(res), err
	}

	if len(res) < len(p) {
		err = io.EOF
	}
	return len(res), err
}

// Seek moves the current location pointer to the new location specified by offset.
func (o *LargeObject) Seek(offset int64, whence int) (int64, error) {
	var n int64
	o.tx = o.tx.Raw("select lo_lseek64($1, $2, $3)", o.fd, offset, whence).Scan(&n)
	return n, o.tx.Error
}

// Tell returns the current read or write location of the large object descriptor.
func (o *LargeObject) Tell() (int64, error) {
	var n int64
	o.tx = o.tx.Raw("select lo_tell64($1)", o.fd).Scan(&n)
	return n, o.tx.Error
}

// Truncate the large object to size.
func (o *LargeObject) Truncate(size int64) (err error) {
	o.tx = o.tx.Raw("select lo_truncate64(?, ?)", o.fd, size)
	return o.tx.Error
}

// Close the large object descriptor.
func (o *LargeObject) Close() error {
	o.tx = o.tx.Raw("select lo_close(?)", o.fd)
	return o.tx.Error
}
