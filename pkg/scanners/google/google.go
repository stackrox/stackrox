package google

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/images"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"bitbucket.org/stack-rox/apollo/pkg/scanners"
	"golang.org/x/oauth2/google"
	"google.golang.org/genproto/googleapis/devtools/containeranalysis/v1alpha1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/oauth"
)

const (
	cloudPlatformScope        = "https://www.googleapis.com/auth/cloud-platform"
	containerAnalysisEndpoint = "containeranalysis.googleapis.com:443"
	requestTimeout            = 10 * time.Second

	maxComponentResults = 1000 // Need all the components so vulns can be attributed
	maxVulnResults      = 200
)

var (
	log = logging.LoggerForModule()
)

type googleScanner struct {
	client           containeranalysis.ContainerAnalysisClient
	project          string
	protoIntegration *v1.ImageIntegration
}

func newScanner(integration *v1.ImageIntegration) (*googleScanner, error) {
	project, ok := integration.GetConfig()["project"]
	if !ok {
		return nil, errors.New("'project' parameter must be defined for Google Container Analysis")
	}
	serviceAccount, ok := integration.GetConfig()["serviceAccount"]
	if !ok {
		return nil, errors.New("'service-account' parameter must be defined for Google Container Analysis")
	}
	conn, err := getGRPCConnection(serviceAccount)
	if err != nil {
		return nil, err
	}
	scanner := &googleScanner{
		client: containeranalysis.NewContainerAnalysisClient(conn),

		project:          project,
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
	creds := oauth.TokenSource{tokenSource.TokenSource}
	conn, err := grpc.Dial(containerAnalysisEndpoint,
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
func (c *googleScanner) Test() error {
	ctx, cancel := grpcContext()
	defer cancel()
	_, err := c.client.ListNotes(ctx, &containeranalysis.ListNotesRequest{
		Parent:   "projects/" + c.project,
		PageSize: 1,
	})
	return err
}

func getResourceURL(image *v1.Image) string {
	return fmt.Sprintf("https://%s/%s@%s", image.GetName().GetRegistry(), image.GetName().GetRemote(), images.NewDigest(image.GetName().GetSha()).Digest())
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
func (c *googleScanner) getComponents(image *v1.Image) (map[string]map[string]*v1.ImageScanComponents, error) {
	cpeToComponentMap := make(map[string]map[string]*v1.ImageScanComponents)
	filter := fmt.Sprintf(`kind="PACKAGE_MANAGER" AND resourceUrl="%s"`, getResourceURL(image))
	occurenceReq := &containeranalysis.ListOccurrencesRequest{
		Parent:   "projects/" + c.project,
		PageSize: maxComponentResults,
		Filter:   filter,
	}
	ctx, cancel := grpcContext()
	defer cancel()
	resp, err := c.client.ListOccurrences(ctx, occurenceReq)
	if err != nil {
		return nil, err
	}
	log.Infof("Found %d components for image %s", len(resp.GetOccurrences()), image.GetName().GetFullName())
	for _, occurrence := range resp.GetOccurrences() {
		cpeURI, component := c.convertComponent(occurrence)
		componentMap, ok := cpeToComponentMap[cpeURI]
		if !ok {
			componentMap = make(map[string]*v1.ImageScanComponents)
			cpeToComponentMap[cpeURI] = componentMap
		}
		componentMap[generalizeName(component.GetName())] = component
	}
	return cpeToComponentMap, nil
}

// this function matches against package substrings in an attempt to limit the number of vulns that cannot be correlated to matches
// this function is expensive
func vulnSubstringMatch(componentMap map[string]*v1.ImageScanComponents, pkg string, vulnerability *v1.Vulnerability) bool {
	for k, comp := range componentMap {
		if strings.Contains(k, pkg) {
			comp.Vulns = append(comp.Vulns, vulnerability)
			return true
		}
	}
	return false
}

// getVulns takes in the cpeToComponentMap and uses it to correlate its vulns to the components
func (c *googleScanner) getVulns(cpeToComponentMap map[string]map[string]*v1.ImageScanComponents, image *v1.Image) error {
	filter := fmt.Sprintf(`kind="PACKAGE_VULNERABILITY" AND resourceUrl="%s"`, getResourceURL(image))
	occurenceReq := &containeranalysis.ListOccurrencesRequest{
		Parent:   "projects/" + c.project,
		PageSize: maxVulnResults,
		Filter:   filter,
	}
	ctx, cancel := grpcContext()
	defer cancel()
	resp, err := c.client.ListOccurrences(ctx, occurenceReq)
	if err != nil {
		return err
	}
	log.Infof("Found %d vulnerabilies for image %s", len(resp.GetOccurrences()), image.GetName().GetFullName())
	for _, occurrence := range resp.GetOccurrences() {
		ctx, cancel = grpcContext()
		defer cancel()
		note, err := c.client.GetOccurrenceNote(ctx, &containeranalysis.GetOccurrenceNoteRequest{Name: occurrence.GetName()})
		if err != nil {
			return err
		}
		cpeURI, pkg, vuln := c.convertVulnerability(occurrence, note)
		componentMap, ok := cpeToComponentMap[cpeURI]
		if !ok {
			log.Errorf("CPE URI '%v' was not associated to a set of components", cpeURI)
		} else {
			component, ok := componentMap[generalizeName(pkg)]
			if !ok {
				if found := vulnSubstringMatch(componentMap, pkg, vuln); !found {
					log.Errorf("Could not associate vuln '%v' with package '%v' because it was not found", vuln.GetCve(), pkg)
				}
			} else {
				component.Vulns = append(component.Vulns, vuln)
			}
		}
	}
	return nil
}

// GetLastScan retrieves the most recent scan
func (c *googleScanner) GetLastScan(image *v1.Image) (*v1.ImageScan, error) {
	log.Infof("Retrieving scans for image %s", image.GetName().GetFullName())
	cpeToComponentMap, err := c.getComponents(image)
	if err != nil {
		return nil, err
	}
	if err := c.getVulns(cpeToComponentMap, image); err != nil {
		return nil, err
	}
	var components []*v1.ImageScanComponents
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
func (c *googleScanner) Match(image *v1.Image) bool {
	return strings.Contains(image.GetName().GetRegistry(), "gcr.io")
}

func (c *googleScanner) Global() bool {
	return len(c.protoIntegration.GetClusters()) == 0
}

func init() {
	scanners.Registry["google"] = func(integration *v1.ImageIntegration) (scanners.ImageScanner, error) {
		scan, err := newScanner(integration)
		return scan, err
	}
}
