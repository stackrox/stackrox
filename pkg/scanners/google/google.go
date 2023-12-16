package google

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"

	containeranalysis "cloud.google.com/go/containeranalysis/apiv1beta1"
	gogoTypes "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stackrox/rox/pkg/urlfmt"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/genproto/googleapis/devtools/containeranalysis/v1beta1/common"
	"google.golang.org/genproto/googleapis/devtools/containeranalysis/v1beta1/grafeas"
	_package "google.golang.org/genproto/googleapis/devtools/containeranalysis/v1beta1/package"
	"google.golang.org/grpc"
)

const (
	requestTimeout = 10 * time.Second

	maxOccurrenceResults = 1000
)

var (
	log = logging.LoggerForModule()
)

// Creator provides the type an scanners.Creator to add to the scanners Registry.
func Creator() (string, func(integration *storage.ImageIntegration) (types.Scanner, error)) {
	return types.Google, func(integration *storage.ImageIntegration) (types.Scanner, error) {
		scan, err := newScanner(integration)
		return scan, err
	}
}

type googleScanner struct {
	types.ScanSemaphore
	betaClient *containeranalysis.GrafeasV1Beta1Client

	project          string
	registry         string
	protoIntegration *storage.ImageIntegration
}

func validate(google *storage.GoogleConfig) error {
	errorList := errorhelpers.NewErrorList("Google Validation")
	if google.GetEndpoint() == "" {
		errorList.AddString("Endpoint must be specified for Google Container Analysis (e.g. gcr.io, us.gcr.io, eu.gcr.io)")
	}
	if google.GetServiceAccount() == "" {
		errorList.AddString("Service account must be specified for Google Container Analysis")
	}
	if google.GetProject() == "" {
		errorList.AddString("ProjectID must be specified for Google Container Analysis")
	}
	return errorList.ToError()
}

func newScanner(integration *storage.ImageIntegration) (*googleScanner, error) {
	googleConfig, ok := integration.IntegrationConfig.(*storage.ImageIntegration_Google)
	if !ok {
		return nil, errors.New("Google Container Analysis configuration required")
	}
	config := googleConfig.Google
	if err := validate(config); err != nil {
		return nil, err
	}

	url := urlfmt.FormatURL(config.GetEndpoint(), urlfmt.HTTPS, urlfmt.NoTrailingSlash)

	proxySupport := grpc.WithContextDialer(proxy.AwareDialContext)
	registry := urlfmt.GetServerFromURL(url)
	betaClient, err := containeranalysis.NewGrafeasV1Beta1Client(context.Background(), option.WithGRPCDialOption(proxySupport), option.WithCredentialsJSON([]byte(config.GetServiceAccount())))
	if err != nil {
		return nil, err
	}

	scanner := &googleScanner{
		betaClient:       betaClient,
		project:          config.GetProject(),
		registry:         registry,
		protoIntegration: integration,

		ScanSemaphore: types.NewDefaultSemaphore(),
	}
	return scanner, nil
}

func grpcContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), requestTimeout)
}

// Test initiates a test of the Google Scanner which verifies that we have the proper scan permissions
func (c *googleScanner) Test() error {
	ctx, cancel := grpcContext()
	defer cancel()
	it := c.betaClient.ListNotes(ctx, &grafeas.ListNotesRequest{
		Parent:   "projects/" + c.project,
		PageSize: 1,
	})
	if _, err := it.Next(); err != nil && err != iterator.Done {
		return err
	}
	return nil
}

func (c *googleScanner) getOccurrencesForImage(image *storage.Image) ([]*grafeas.Occurrence, []*grafeas.Occurrence, error) {
	filterStr := `resourceUrl="` + getResourceURL(image) + `"`
	project := "projects/" + c.project

	req := &grafeas.ListOccurrencesRequest{
		Parent:   project,
		Filter:   filterStr,
		PageSize: maxOccurrenceResults,
	}
	ctx, cancel := grpcContext()
	defer cancel()

	it := c.betaClient.ListOccurrences(ctx, req)
	var componentOccurrences []*grafeas.Occurrence
	var vulnOccurrences []*grafeas.Occurrence
	for {
		occ, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, nil, err
		}
		switch occ.GetKind() {
		case common.NoteKind_PACKAGE:
			componentOccurrences = append(componentOccurrences, occ)
		case common.NoteKind_VULNERABILITY:
			vulnOccurrences = append(vulnOccurrences, occ)
		}
	}
	return componentOccurrences, vulnOccurrences, nil
}

func getResourceURL(image *storage.Image) string {
	return fmt.Sprintf("https://%s/%s@%s", image.GetName().GetRegistry(), image.GetName().GetRemote(), utils.GetSHA(image))
}

func trimDigits(r rune) bool {
	if unicode.IsDigit(r) || r == '.' {
		return true
	}
	return false
}

func generalizeName(name string) string {
	name = strings.TrimPrefix(name, "lib")
	name = strings.TrimRightFunc(name, trimDigits)
	return name
}

// this function matches against package substrings in an attempt to limit the number of vulns that cannot be correlated to matches
// this function is expensive
func bestEffortMatch(componentsToVulns map[packageAndVersion][]*grafeas.Occurrence, pkg packageAndVersion) (packageAndVersion, bool) {
	for pv := range componentsToVulns {
		if strings.Contains(generalizeName(pv.name), generalizeName(pkg.name)) && pv.version == pkg.version {
			return pv, true
		}
	}
	return pkg, false
}

type packageAndVersion struct {
	name    string
	version string
}

func getPackageAndVersion(installation *_package.Installation) packageAndVersion {
	pv := packageAndVersion{
		name: installation.GetName(),
	}
	if len(installation.GetLocation()) > 0 {
		pv.version = installation.GetLocation()[0].GetVersion().GetName()
	}
	return pv
}

func (c *googleScanner) processComponent(pv packageAndVersion, vulns []*grafeas.Occurrence, convertChan chan *storage.EmbeddedImageScanComponent) {
	component := c.convertComponentFromPackageAndVersion(pv)
	component.Vulns = c.convertVulnsFromOccurrences(vulns)
	convertChan <- component
}

// GetScan retrieves the most recent scan
func (c *googleScanner) GetScan(image *storage.Image) (*storage.ImageScan, error) {
	log.Infof("Retrieving scans for image %s", image.GetName().GetFullName())

	componentOccurrences, vulnOccurrences, err := c.getOccurrencesForImage(image)
	if err != nil {
		return nil, err
	}

	componentsToVulns := make(map[packageAndVersion][]*grafeas.Occurrence)
	for _, c := range componentOccurrences {
		pv := getPackageAndVersion(c.GetInstallation().GetInstallation())
		componentsToVulns[pv] = []*grafeas.Occurrence{}
	}

	// Match vulnerabilities with the componentOccurrences
	for _, v := range vulnOccurrences {
		vulnerability := v.GetVulnerability()
		if len(vulnerability.GetPackageIssue()) == 0 {
			continue
		}
		packageIssue := vulnerability.GetPackageIssue()[0]
		pv := packageAndVersion{
			name:    packageIssue.AffectedLocation.Package,
			version: packageIssue.AffectedLocation.GetVersion().GetName(),
		}

		if _, ok := componentsToVulns[pv]; ok {
			componentsToVulns[pv] = append(componentsToVulns[pv], v)
		} else if matchedPV, ok := bestEffortMatch(componentsToVulns, pv); ok {
			componentsToVulns[matchedPV] = append(componentsToVulns[matchedPV], v)
		} else {
			// We don't want to miss vulnerabilities so add them as a separate component
			componentsToVulns[pv] = []*grafeas.Occurrence{v}
		}
	}

	// Parallelize this as it makes a bunch of calls to the API
	convertChan := make(chan *storage.EmbeddedImageScanComponent)
	components := make([]*storage.EmbeddedImageScanComponent, 0, len(componentsToVulns))
	for pv, occurrences := range componentsToVulns {
		go c.processComponent(pv, occurrences, convertChan)
	}
	for range componentsToVulns {
		components = append(components, <-convertChan)
	}
	// Google can't give the data via layers at this time
	return &storage.ImageScan{
		ScanTime:        gogoTypes.TimestampNow(),
		Components:      components,
		OperatingSystem: "unknown",
		Notes: []storage.ImageScan_Note{
			storage.ImageScan_OS_UNAVAILABLE,
		},
	}, nil
}

// Match decides if the image is contained within this scanner
func (c *googleScanner) Match(image *storage.ImageName) bool {
	return image.GetRegistry() == c.registry && strings.HasPrefix(image.GetRemote(), c.project)
}

func (c *googleScanner) Type() string {
	return types.Google
}

func (c *googleScanner) Name() string {
	return c.protoIntegration.GetName()
}

func (c *googleScanner) GetVulnDefinitionsInfo() (*v1.VulnDefinitionsInfo, error) {
	return nil, nil
}
