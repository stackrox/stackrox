package google

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stackrox/rox/pkg/urlfmt"
	"golang.org/x/oauth2/google"
	"google.golang.org/genproto/googleapis/devtools/containeranalysis/v1alpha1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
)

const (
	cloudPlatformScope        = "https://www.googleapis.com/auth/cloud-platform"
	containerAnalysisEndpoint = "containeranalysis.googleapis.com:443"
	requestTimeout            = 10 * time.Second

	maxComponentResults = 1000 // Need all the components so vulns can be attributed
	maxVulnResults      = 200

	typeString = "google"
)

var (
	log = logging.LoggerForModule()
)

// Creator provides the type an scanners.Creator to add to the scanners Registry.
func Creator() (string, func(integration *v1.ImageIntegration) (types.ImageScanner, error)) {
	return typeString, func(integration *v1.ImageIntegration) (types.ImageScanner, error) {
		scan, err := newScanner(integration)
		return scan, err
	}
}

type googleScanner struct {
	client           containeranalysis.ContainerAnalysisClient
	project          string
	registry         string
	protoIntegration *v1.ImageIntegration
}

func validate(google *v1.GoogleConfig) error {
	errorList := errorhelpers.NewErrorList("Google Validation")
	if google.GetEndpoint() == "" {
		errorList.AddString("Endpoint must be specified for Google Container Analysis (e.g. gcr.io, us.gcr.io, eu.gcr.io)")
	}
	if google.GetServiceAccount() == "" {
		errorList.AddString("Service account must be specified for Google Container Analysis")
	}
	if google.GetProject() == "" {
		errorList.AddString("Project must be specified for Google Container Analysis")
	}
	return errorList.ToError()
}

func newScanner(integration *v1.ImageIntegration) (*googleScanner, error) {
	googleConfig, ok := integration.IntegrationConfig.(*v1.ImageIntegration_Google)
	if !ok {
		return nil, fmt.Errorf("Google Container Analysis configuration required")
	}
	config := googleConfig.Google
	if err := validate(config); err != nil {
		return nil, err
	}

	url, err := urlfmt.FormatURL(config.GetEndpoint(), urlfmt.HTTPS, urlfmt.NoTrailingSlash)
	if err != nil {
		return nil, err
	}
	registry := urlfmt.GetServerFromURL(url)
	conn, err := getGRPCConnection(config.GetServiceAccount())
	if err != nil {
		return nil, err
	}
	scanner := &googleScanner{
		client: containeranalysis.NewContainerAnalysisClient(conn),

		project:          config.GetProject(),
		registry:         registry,
		protoIntegration: integration,
	}
	return scanner, nil
}

func getGRPCConnection(serviceAccount string) (*grpc.ClientConn, error) {
	ctx, cancel := grpcContext()
	defer cancel()
	tokenSource, err := getTokenSource(ctx, serviceAccount)
	if err != nil {
		return nil, err
	}
	creds := oauth.TokenSource{TokenSource: tokenSource.TokenSource}
	conn, err := grpc.Dial(containerAnalysisEndpoint,
		grpc.WithTransportCredentials(credentials.NewTLS(nil)),
		grpc.WithPerRPCCredentials(
			oauth.TokenSource{
				TokenSource: creds.TokenSource,
			},
		),
	)
	return conn, err
}

func getTokenSource(ctx context.Context, serviceAccount string) (*google.DefaultCredentials, error) {
	serviceAccountBytes := []byte(serviceAccount)
	cfg, err := google.JWTConfigFromJSON(serviceAccountBytes, cloudPlatformScope)
	if err != nil {
		return nil, fmt.Errorf("google.JWTConfigFromJSON: %v", err)
	}
	// jwt.Config does not expose the project ID, so re-unmarshal to get it.
	var pid struct {
		ProjectID string `json:"project_id"`
	}
	if err := json.Unmarshal(serviceAccountBytes, &pid); err != nil {
		return nil, err
	}
	return &google.DefaultCredentials{
		ProjectID:   pid.ProjectID,
		TokenSource: cfg.TokenSource(ctx),
		JSON:        serviceAccountBytes,
	}, nil
}

func grpcContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), requestTimeout)
}

// Test initiates a test of the Google Scanner which verifies that we have the proper scan permissions
func (g *googleScanner) Test() error {
	ctx, cancel := grpcContext()
	defer cancel()
	_, err := g.client.ListNotes(ctx, &containeranalysis.ListNotesRequest{
		Parent:   "projects/" + g.project,
		PageSize: 1,
	})
	return err
}

func getResourceURL(image *v1.Image) string {
	return fmt.Sprintf("https://%s/%s@%s", image.GetName().GetRegistry(), image.GetName().GetRemote(), image.GetId())
}

func generalizeName(name string) string {
	if idx := strings.Index(name, "-"); idx != -1 {
		return name[:idx]
	}
	return name
}

// getComponents returns a map of cpeURIs -> map of generalized component names to components
// the cpeURIs are for determining which summary is correct for the distribution
// The names are generalized because google doesn't do direct correlation so pkg mysql should match mysql-5.5
func (g *googleScanner) getComponents(image *v1.Image) (map[string]map[string]*v1.ImageScanComponent, error) {
	cpeToComponentMap := make(map[string]map[string]*v1.ImageScanComponent)
	filter := fmt.Sprintf(`kind="PACKAGE_MANAGER" AND resourceUrl="%s"`, getResourceURL(image))
	occurenceReq := &containeranalysis.ListOccurrencesRequest{
		Parent:   "projects/" + g.project,
		PageSize: maxComponentResults,
		Filter:   filter,
	}
	ctx, cancel := grpcContext()
	defer cancel()
	resp, err := g.client.ListOccurrences(ctx, occurenceReq)
	if err != nil {
		return nil, err
	}
	log.Infof("Found %d components for image %s", len(resp.GetOccurrences()), image.GetName().GetFullName())
	for _, occurrence := range resp.GetOccurrences() {
		cpeURI, component := g.convertComponentFromPackageManagerOccurrence(occurrence)

		if _, ok := cpeToComponentMap[cpeURI]; !ok {
			cpeToComponentMap[cpeURI] = make(map[string]*v1.ImageScanComponent)
		}
		cpeToComponentMap[cpeURI][generalizeName(component.GetName())] = component
	}
	return cpeToComponentMap, nil
}

// this function matches against package substrings in an attempt to limit the number of vulns that cannot be correlated to matches
// this function is expensive
func vulnSubstringMatch(componentMap map[string]*v1.ImageScanComponent, pkg string) (*v1.ImageScanComponent, bool) {
	for k, comp := range componentMap {
		if strings.Contains(k, pkg) {
			return comp, true
		}
	}
	return nil, false
}

func (g *googleScanner) getVulnsForImage(image *v1.Image) ([]*containeranalysis.Occurrence, error) {
	filter := fmt.Sprintf(`kind="PACKAGE_VULNERABILITY" AND resourceUrl="%s"`, getResourceURL(image))
	occurenceReq := &containeranalysis.ListOccurrencesRequest{
		Parent:   "projects/" + g.project,
		PageSize: maxVulnResults,
		Filter:   filter,
	}
	ctx, cancel := grpcContext()
	defer cancel()
	resp, err := g.client.ListOccurrences(ctx, occurenceReq)
	if err != nil {
		return nil, fmt.Errorf("could not list occurences: %s", err)
	}
	log.Infof("Found %d vulnerabilities for image %s", len(resp.GetOccurrences()), image.GetName().GetFullName())
	return resp.GetOccurrences(), nil
}

func (g *googleScanner) getOccurrenceNote(name string) (*containeranalysis.Note, error) {
	ctx, cancel := grpcContext()
	defer cancel()
	return g.client.GetOccurrenceNote(ctx, &containeranalysis.GetOccurrenceNoteRequest{Name: name})
}

// addVulnsToComponents takes in the cpeToComponentMap and uses it to correlate its vulns to the components
func (g *googleScanner) addVulnsToComponents(cpeToComponentMap map[string]map[string]*v1.ImageScanComponent, image *v1.Image) error {
	// This retrieves all the vulnerabilities for the image through a request to the API
	pkgVulnOccurences, err := g.getVulnsForImage(image)
	if err != nil {
		return fmt.Errorf("failed to get vulns for image: %s", err)
	}
	// For every package based vulnerability, get the occurrence note, which gives more info about the vuln
	for _, occurrence := range pkgVulnOccurences {
		note, err := g.getOccurrenceNote(occurrence.GetName())
		if err != nil {
			log.Errorf("unable to get occurrence note '%s': %s", occurrence.GetName(), err)
			continue
		}

		// Join the vuln to the component by using the vulns cpeURI -> to look up the affected components
		g.joinVulnToComponent(cpeToComponentMap, occurrence, note)
	}
	return nil
}

func (g *googleScanner) joinVulnToComponent(cpeToComponentMap map[string]map[string]*v1.ImageScanComponent, occurrence *containeranalysis.Occurrence, note *containeranalysis.Note) {
	cpeURI, pkg, vuln := g.convertVulnerabilityFromPackageVulnerabilityOccurrence(occurrence, note)
	_, ok := cpeToComponentMap[cpeURI]
	if !ok {
		cpeToComponentMap[cpeURI] = make(map[string]*v1.ImageScanComponent)
	}
	componentNameToComponents := cpeToComponentMap[cpeURI]

	component, ok := componentNameToComponents[generalizeName(pkg)]
	if !ok {
		if matchedComponent, ok := vulnSubstringMatch(componentNameToComponents, pkg); ok {
			componentNameToComponents[pkg] = matchedComponent
		} else {
			componentNameToComponents[pkg] = &v1.ImageScanComponent{
				Name: pkg,
			}
		}
		component = componentNameToComponents[pkg]
	}
	component.Vulns = append(component.Vulns, vuln)
}

// GetLastScan retrieves the most recent scan
func (g *googleScanner) GetLastScan(image *v1.Image) (*v1.ImageScan, error) {
	log.Infof("Retrieving scans for image %s", image.GetName().GetFullName())
	cpeToComponentMap, err := g.getComponents(image)
	if err != nil {
		return nil, fmt.Errorf("failed to get components: %s", err)
	}
	if len(cpeToComponentMap) == 0 {
		return nil, fmt.Errorf("No components were found in image '%s'", image.GetName().GetFullName())
	}
	if err := g.addVulnsToComponents(cpeToComponentMap, image); err != nil {
		return nil, fmt.Errorf("failed to add vulns to components: %s", err)
	}
	var components []*v1.ImageScanComponent
	for _, v := range cpeToComponentMap {
		for _, component := range v {
			components = append(components, component)
		}
	}
	return &v1.ImageScan{
		Components: components,
	}, nil
}

// Match decides if the image is contained within this scanner
func (g *googleScanner) Match(image *v1.Image) bool {
	return image.GetName().GetRegistry() == g.registry && strings.HasPrefix(image.GetName().GetRemote(), g.project)
}

func (g *googleScanner) Global() bool {
	return len(g.protoIntegration.GetClusters()) == 0
}

func (g *googleScanner) Type() string {
	return typeString
}
