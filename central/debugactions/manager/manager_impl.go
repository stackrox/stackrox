package manager

import v2 "github.com/stackrox/rox/generated/api/v2"

type managerImpl struct {
}

func (m managerImpl) RegisterAction(action *v2.DebugAction) error {
	//TODO implement me
	panic("implement me")
}

func (m managerImpl) GetActionStatus(identifier string) (*v2.ActionStatus, error) {
	//TODO implement me
	panic("implement me")
}

func (m managerImpl) DeleteAction(identifier string) error {
	//TODO implement me
	panic("implement me")
}

func (m managerImpl) ProceedOldest(identifier string) error {
	//TODO implement me
	panic("implement me")
}

func (m managerImpl) ProceedAll(identifier string) error {
	//TODO implement me
	panic("implement me")
}

func (m managerImpl) Start() {
	//TODO implement me
	panic("implement me")
}

func (m managerImpl) Stop() {
	//TODO implement me
	panic("implement me")
}
