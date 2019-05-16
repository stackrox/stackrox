package client

import (
	"fmt"

	"github.com/docker/docker/api/types/versions"
)

// errConnectionFailed implements an error returned when connection failed.
type errConnectionFailed struct {
	host string
}

// Error returns a string representation of an errConnectionFailed
func (err errConnectionFailed) Error() string {
	if err.host == "" {
		return "Cannot connect to the Docker daemon. Is the docker daemon running on this host?"
	}
	return fmt.Sprintf("Cannot connect to the Docker daemon at %s. Is the docker daemon running?", err.host)
}

// ErrorConnectionFailed returns an error with host in the error message when connection to docker daemon failed.
func ErrorConnectionFailed(host string) error {
	return errConnectionFailed{host: host}
}

type notFound interface {
	error
	NotFound() bool // Is the error a NotFound error
}

// IsErrNotFound returns true if the error is caused with an
// object (image, container, network, volume, …) is not found in the docker host.
func IsErrNotFound(err error) bool {
	te, ok := err.(notFound)
	return ok && te.NotFound()
}

// imageNotFoundError implements an error returned when an image is not in the docker host.
type imageNotFoundError struct {
	imageID string
}

// NotFound indicates that this error type is of NotFound
func (e imageNotFoundError) NotFound() bool {
	return true
}

// Error returns a string representation of an imageNotFoundError
func (e imageNotFoundError) Error() string {
	return fmt.Sprintf("Error: No such image: %s", e.imageID)
}

// IsErrImageNotFound returns true if the error is caused
// when an image is not found in the docker host.
func IsErrImageNotFound(err error) bool {
	return IsErrNotFound(err)
}

// containerNotFoundError implements an error returned when a container is not in the docker host.
type containerNotFoundError struct {
	containerID string
}

// NotFound indicates that this error type is of NotFound
func (e containerNotFoundError) NotFound() bool {
	return true
}

// Error returns a string representation of a containerNotFoundError
func (e containerNotFoundError) Error() string {
	return fmt.Sprintf("Error: No such container: %s", e.containerID)
}

// IsErrContainerNotFound returns true if the error is caused
// when a container is not found in the docker host.
func IsErrContainerNotFound(err error) bool {
	return IsErrNotFound(err)
}

// networkNotFoundError implements an error returned when a network is not in the docker host.
type networkNotFoundError struct {
	networkID string
}

// NotFound indicates that this error type is of NotFound
func (e networkNotFoundError) NotFound() bool {
	return true
}

// Error returns a string representation of a networkNotFoundError
func (e networkNotFoundError) Error() string {
	return fmt.Sprintf("Error: No such network: %s", e.networkID)
}

// NewVersionError returns an error if the APIVersion required
// if less than the current supported version
func (cli *Client) NewVersionError(APIrequired, feature string) error {
	if cli.version != "" && versions.LessThan(cli.version, APIrequired) {
		return fmt.Errorf("%q requires API version %s, but the Docker daemon API version is %s", feature, APIrequired, cli.version)
	}
	return nil
}
