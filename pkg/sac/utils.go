package sac

// VerifyAuthzOK converts no access to ErrResourceAccessDenied.
func VerifyAuthzOK(ok bool, err error) error {
	if err != nil {
		return err
	}
	if !ok {
		return ErrResourceAccessDenied
	}
	return nil
}
