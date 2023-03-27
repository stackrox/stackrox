package postgres

import "github.com/stackrox/rox/generated/storage"

func (b *indexerImpl) AddListAlert(_ *storage.ListAlert) error {
	return nil
}

func (b *indexerImpl) AddListAlerts(_ []*storage.ListAlert) error {
	return nil
}

func (b *indexerImpl) DeleteListAlert(_ string) error {
	return nil
}

func (b *indexerImpl) DeleteListAlerts(_ []string) error {
	return nil
}
