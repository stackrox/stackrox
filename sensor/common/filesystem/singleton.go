package filesystem

import "github.com/stackrox/rox/generated/storage"

func New() Service {
	return &serviceImpl{
		queue:  make(chan *storage.FileActivity),
		name:   "it's me!",
		writer: nil,
	}
}
