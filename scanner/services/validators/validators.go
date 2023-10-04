package validators

import (
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/quay/claircore/pkg/cpe"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/errox"
)

// hasIDAndCPE is the common interface offered by some proto definitions related
// to ClairCore. It's used to simplify validation.
type hasIDAndCPE interface {
	GetCpe() string
	GetId() string
}

// ValidateGetVulnerabilitiesRequest validates matcher get vulnerability
// requests, ensuring any potential internal error is captured upfront as invalid
// argument.
func ValidateGetVulnerabilitiesRequest(req *v4.GetVulnerabilitiesRequest) error {
	if req == nil {
		return errors.New("empty request")
	}
	// We only support container image resources for now.
	if !strings.HasPrefix(req.GetHashId(), "/v4/containerimage/") {
		return fmt.Errorf("invalid hash id: %q", req.GetHashId())
	}
	if err := validateContents(req.GetContents()); err != nil {
		return err
	}
	return nil
}

// ValidateContainerImageRequest validates a container image request, ensuring
// any potential internal error is captured upfront as invalid argument.
func ValidateContainerImageRequest(req *v4.CreateIndexReportRequest) error {
	if req == nil {
		return errox.InvalidArgs.New("empty request")
	}
	if !strings.HasPrefix(req.GetHashId(), "/v4/containerimage/") {
		return errox.InvalidArgs.Newf("invalid hash id: %q", req.GetHashId())
	}
	if req.GetContainerImage() == nil {
		return errox.InvalidArgs.New("invalid resource locator for container image")
	}
	// Validate container image URL.
	imgURL := req.GetContainerImage().GetUrl()
	if imgURL == "" {
		return errox.InvalidArgs.New("missing image URL")
	}
	u, err := url.Parse(imgURL)
	if err != nil {
		return errox.InvalidArgs.Newf("invalid image URL: %q", imgURL).CausedBy(err)
	}
	switch u.Scheme {
	case "http", "https":
	default:
		return errox.InvalidArgs.New("image URL does not start with http:// or https://")
	}
	imageRef := strings.TrimPrefix(imgURL, u.Scheme+"://")
	_, err = name.ParseReference(imageRef, name.StrictValidation)
	if err != nil {
		return errox.InvalidArgs.CausedBy(err)
	}
	return nil
}

func validateContents(contents *v4.Contents) error {
	if contents == nil {
		return nil
	}
	if err := validateList(contents.GetPackages(), "Contents.Packages", validatePackage); err != nil {
		return err
	}
	if err := validateList(contents.GetDistributions(), "Contents.Distributions", validateDistribution); err != nil {
		return err
	}
	if err := validateList(contents.GetRepositories(), "Contents.Repositories", validateRepository); err != nil {
		return err
	}
	for k, envs := range contents.GetEnvironments() {
		for idx, env := range envs.GetEnvironments() {
			if env == nil {
				return fmt.Errorf("Contents.Environments[%q] element #%d is empty", k, idx+1)
			}
		}
	}
	return nil
}

func validateList[T hasIDAndCPE](l []T, fieldName string, validateF func(T) error) error {
	for idx, o := range l {
		n := idx + 1
		if reflect.ValueOf(o).IsZero() {
			return fmt.Errorf("%s element #%d is empty", fieldName, n)
		}
		if o.GetId() == "" {
			return fmt.Errorf("%s element #%d: Id is empty", fieldName, n)
		}
		_, err := cpe.UnbindFS(o.GetCpe())
		if err != nil {
			return fmt.Errorf("%s element #%d (id: %q): invalid CPE: %w", fieldName, n, o.GetId(), err)
		}
		if err := validateF(o); err != nil {
			return fmt.Errorf("%s element #%d (id: %q): %w", fieldName, n, o.GetId(), err)
		}
	}
	return nil
}

func validateRepository(_ *v4.Repository) error {
	// Placeholder, currently no additional validation for distribution repository.
	return nil
}

func validateDistribution(_ *v4.Distribution) error {
	// Placeholder, currently no additional validation for distribution.
	return nil
}

func validatePackage(pkg *v4.Package) error {
	if pkg.GetSource().GetSource() != nil {
		return fmt.Errorf("package ID=%q has a source with a source", pkg.GetId())
	}
	return nil
}
