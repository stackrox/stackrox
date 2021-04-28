package sac

// VerifyAuthzOK converts no access to ErrPermissionDenied.
func VerifyAuthzOK(ok bool, err error) error {
	if err != nil {
		return err
	}
	if !ok {
		return ErrPermissionDenied
	}
	return nil
}
