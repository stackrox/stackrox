package upgradecontroller

type unexpectedError struct {
	err error
}

func (u *upgradeController) expectNoError(err error) {
	if err == nil {
		return
	}
	log.Errorf("Unexpected error in upgrade controller for cluster %s: %v", u.clusterID, err)
	panic(unexpectedError{err: err})
}
