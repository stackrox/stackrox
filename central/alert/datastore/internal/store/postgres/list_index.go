package postgres

import "github.com/stackrox/rox/generated/storage"

func (b *indexerImpl) AddListAlert(listalert *storage.ListAlert) error {
	return nil
}

func (b *indexerImpl) AddListAlerts(listalerts []*storage.ListAlert) error {
	return nil
}

func (b *indexerImpl) DeleteListAlert(id string) error {
	return nil
}

func (b *indexerImpl) DeleteListAlerts(ids []string) error {
	return nil
}
