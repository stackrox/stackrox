package api

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/scandata/datastore"
	"github.com/stackrox/rox/central/scandata/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/logging"
	"google.golang.org/grpc/codes"
)

var (
	log = logging.LoggerForModule()
)

type handler struct {
	datastore datastore.DataStore
}

// NewHandler creates a new HTTP handler for scan data API
func NewHandler(ds datastore.DataStore) http.Handler {
	h := &handler{
		datastore: ds,
	}

	router := mux.NewRouter()
	router.HandleFunc("/v1/scandata/cves", h.listCVEs).Methods(http.MethodGet)
	router.HandleFunc("/v1/scandata/cves/{cveName}", h.getCVEDetail).Methods(http.MethodGet)
	router.HandleFunc("/v1/scandata/images", h.listImages).Methods(http.MethodGet)
	router.HandleFunc("/v1/scandata/images/{imageId}/findings", h.getImageFindings).Methods(http.MethodGet)
	router.HandleFunc("/v1/scandata/images/{imageId}", h.getImageDetail).Methods(http.MethodGet)
	router.HandleFunc("/v1/scandata/advisories", h.listAdvisories).Methods(http.MethodGet)
	router.HandleFunc("/v1/scandata/deployments", h.listDeployments).Methods(http.MethodGet)
	router.HandleFunc("/v1/scandata/deployments/{deploymentId}", h.getDeploymentDetail).Methods(http.MethodGet)
	router.HandleFunc("/v1/scandata/components", h.listComponents).Methods(http.MethodGet)
	router.HandleFunc("/v1/scandata/components/cves", h.getComponentCVEs).Methods(http.MethodGet)
	router.HandleFunc("/v1/scandata/components/images", h.getComponentImages).Methods(http.MethodGet)
	router.HandleFunc("/v1/scandata/components/detail", h.getComponentDetail).Methods(http.MethodGet)

	return router
}

// CVEListResponse is the response for GET /v1/scandata/cves
type CVEListResponse struct {
	CVEs       []CVEListItem `json:"cves"`
	TotalCount int           `json:"totalCount"`
}

// CVEListItem represents one CVE in the list
type CVEListItem struct {
	CVEName         string     `json:"cveName"`
	Severity        int32      `json:"severity"`
	CVSS            float32    `json:"cvss"`
	ImageCount      int        `json:"imageCount"`
	Fixable         bool       `json:"fixable"`
	FirstSeen       *time.Time `json:"firstSeen,omitzero"`
	PublishedDate   *time.Time `json:"publishedDate,omitzero"`
	EPSSProbability float32    `json:"epssProbability,omitzero"`
}

// CVEDetailResponse is the response for GET /v1/scandata/cves/{cveName}
type CVEDetailResponse struct {
	CVEName     string          `json:"cveName"`
	Severity    int32           `json:"severity"`
	CVSS        float32         `json:"cvss"`
	Description string          `json:"description,omitempty"`
	Advisories  []AdvisoryInfo  `json:"advisories"`
	Components  []ComponentInfo `json:"components"`
	Images      []ImageInfo     `json:"images"`
}

// AdvisoryInfo represents an advisory for a CVE
type AdvisoryInfo struct {
	ID          string  `json:"id"`
	Severity    int32   `json:"severity"`
	CVSS        float32 `json:"cvss"`
	SourceName  string  `json:"sourceName"`
	Description string  `json:"description,omitempty"`
	Link        string  `json:"link,omitempty"`
	FixedBy     string  `json:"fixedBy,omitempty"`
}

// ComponentInfo represents a component affected by a CVE
type ComponentInfo struct {
	Name       string `json:"name"`
	Version    string `json:"version"`
	Source     string `json:"source"`
	FixedBy    string `json:"fixedBy,omitzero"`
	ImageCount int    `json:"imageCount"`
}

// ImageComponentInfo represents a component affected by a CVE within a specific image.
type ImageComponentInfo struct {
	Name       string   `json:"name"`
	Version    string   `json:"version"`
	Source     string   `json:"source"`
	Advisories []string `json:"advisories"`
	FixedBy    string   `json:"fixedBy,omitempty"`
}

// ImageInfo represents an image affected by a CVE
type ImageInfo struct {
	ImageID        string               `json:"imageId"`
	ImageUUID      string               `json:"imageUuid,omitempty"`
	ImageName      string               `json:"imageName,omitempty"`
	ComponentCount int                  `json:"componentCount"`
	Severity       int32                `json:"severity"`
	Fixable        bool                 `json:"fixable"`
	Components     []ImageComponentInfo `json:"components"`
}

// ImageFindingsResponse is the response for GET /v1/scandata/images/{imageId}/findings
type ImageFindingsResponse struct {
	ImageID  string                 `json:"imageId"`
	Findings []FindingWithComponent `json:"findings"`
}

// FindingWithComponent represents a finding with component data
type FindingWithComponent struct {
	CVEName          string  `json:"cveName"`
	Severity         int32   `json:"severity"`
	CVSS             float32 `json:"cvss"`
	IsFixable        bool    `json:"isFixable"`
	FixedBy          string  `json:"fixedBy,omitzero"`
	ComponentName    string  `json:"componentName"`
	ComponentVersion string  `json:"componentVersion"`
	ComponentSource  string  `json:"componentSource"`
	SourceName       string  `json:"sourceName"`
}

// DeploymentListResponse is the response for GET /v1/scandata/deployments
type DeploymentListResponse struct {
	Deployments []DeploymentListItem `json:"deployments"`
	TotalCount  int                  `json:"totalCount"`
}

// DeploymentListItem represents one deployment in the list
type DeploymentListItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Cluster     string `json:"cluster"`
	Namespace   string `json:"namespace"`
	ImageCount  int    `json:"imageCount"`
	CVECount    int    `json:"cveCount"`
	TopSeverity int32  `json:"topSeverity"`
	Fixable     bool   `json:"fixable"`
}

// DeploymentDetailResponse is the response for GET /v1/scandata/deployments/{deploymentId}
type DeploymentDetailResponse struct {
	ID        string                  `json:"id"`
	Name      string                  `json:"name"`
	Cluster   string                  `json:"cluster"`
	Namespace string                  `json:"namespace"`
	Images    []DeploymentImageDetail `json:"images"`
}

// DeploymentImageDetail represents an image in a deployment with CVE data
type DeploymentImageDetail struct {
	ImageID     string `json:"imageId"`
	ImageUUID   string `json:"imageUuid,omitempty"`
	ImageName   string `json:"imageName,omitempty"`
	CVECount    int    `json:"cveCount"`
	TopSeverity int32  `json:"topSeverity"`
	Fixable     bool   `json:"fixable"`
}

// ImageDetailResponse is the response for GET /v1/scandata/images/{imageId}
type ImageDetailResponse struct {
	ImageID        string                 `json:"imageId"`
	ImageName      string                 `json:"imageName,omitempty"`
	ImageOS        string                 `json:"imageOS,omitempty"`
	ScanTime       string                 `json:"scanTime,omitempty"`
	ScannerVersion string                 `json:"scannerVersion,omitempty"`
	BundleVersion  string                 `json:"bundleVersion,omitempty"`
	DataSources    []string               `json:"dataSources"`
	Components     []ImageDetailComponent `json:"components"`
	CVESummary     ImageDetailCVESummary  `json:"cveSummary"`
}

// ImageDetailComponent represents a component with its CVEs in the image detail view.
type ImageDetailComponent struct {
	Name     string           `json:"name"`
	Version  string           `json:"version"`
	Source   string           `json:"source"`
	Arch     string           `json:"arch,omitzero"`
	Location string           `json:"location,omitempty"`
	CVEs     []ImageDetailCVE `json:"cves"`
}

// ImageDetailCVE represents a CVE within a component in the image detail view.
type ImageDetailCVE struct {
	CVEName    string   `json:"cveName"`
	Severity   int32    `json:"severity"`
	CVSS       float32  `json:"cvss"`
	FixedBy    string   `json:"fixedBy,omitempty"`
	Advisories []string `json:"advisories"`
}

// ImageDetailCVESummary is the severity breakdown for an image.
type ImageDetailCVESummary struct {
	Total     int `json:"total"`
	Critical  int `json:"critical"`
	Important int `json:"important"`
	Moderate  int `json:"moderate"`
	Low       int `json:"low"`
}

// AdvisoryListResponse is the response for GET /v1/scandata/advisories
type AdvisoryListResponse struct {
	Advisories []AdvisoryListItem `json:"advisories"`
	TotalCount int                `json:"totalCount"`
}

// AdvisoryListItem represents one advisory in the list
type AdvisoryListItem struct {
	AdvisoryID     string  `json:"advisoryId"`
	CVEName        string  `json:"cveName"`
	Severity       int32   `json:"severity"`
	CVSS           float32 `json:"cvss"`
	SourceName     string  `json:"sourceName"`
	Description    string  `json:"description"`
	FixedBy        string  `json:"fixedBy,omitempty"`
	ImageCount     int     `json:"imageCount"`
	ComponentCount int     `json:"componentCount"`
	Link           string  `json:"link"`
}

// ImageListResponse is the response for GET /v1/scandata/images
type ImageListResponse struct {
	Images     []ImageListItem `json:"images"`
	TotalCount int             `json:"totalCount"`
}

// ImageListItem represents one image in the list
type ImageListItem struct {
	ImageID        string     `json:"imageId"`
	ImageUUID      string     `json:"imageUuid,omitempty"`
	ImageName      string     `json:"imageName,omitempty"`
	ImageOS        string     `json:"imageOS,omitempty"`
	CVECount       int        `json:"cveCount"`
	ComponentCount int        `json:"componentCount"`
	TopSeverity    int32      `json:"topSeverity"`
	TopCVSS        float32    `json:"topCvss"`
	Fixable        bool       `json:"fixable"`
	ScanTime       *time.Time `json:"scanTime,omitempty"`
	CriticalCount  int        `json:"criticalCount"`
	ImportantCount int        `json:"importantCount"`
	ModerateCount  int        `json:"moderateCount"`
	LowCount       int        `json:"lowCount"`
}

// ComponentListResponse is the response for GET /v1/scandata/components
type ComponentListResponse struct {
	Components []ComponentListItem `json:"components"`
	TotalCount int                 `json:"totalCount"`
}

// ComponentListItem represents one component in the list
type ComponentListItem struct {
	Name           string  `json:"name"`
	VersionCount   int     `json:"versionCount"`
	CVECount       int     `json:"cveCount"`
	ImageCount     int     `json:"imageCount"`
	TopSeverity    int32   `json:"topSeverity"`
	TopCVSS        float32 `json:"topCvss"`
	CriticalCount  int     `json:"criticalCount"`
	ImportantCount int     `json:"importantCount"`
	ModerateCount  int     `json:"moderateCount"`
	LowCount       int     `json:"lowCount"`
}

// ComponentCVEsResponse is the response for GET /v1/scandata/components/cves
type ComponentCVEsResponse struct {
	ComponentName    string         `json:"componentName"`
	ComponentVersion string         `json:"componentVersion"`
	CVEs             []ComponentCVE `json:"cves"`
}

// ComponentCVE represents a CVE affecting a specific component version
type ComponentCVE struct {
	CVEName     string  `json:"cveName"`
	Severity    int32   `json:"severity"`
	CVSS        float32 `json:"cvss"`
	Fixable     bool    `json:"fixable"`
	FixedBy     string  `json:"fixedBy,omitempty"`
	Description string  `json:"description,omitempty"`
	ImageCount  int     `json:"imageCount"`
}

// ComponentImageInfo represents an image containing a component
type ComponentImageInfo struct {
	ImageID     string `json:"imageId"`
	ImageUUID   string `json:"imageUuid,omitempty"`
	ImageName   string `json:"imageName,omitempty"`
	Version     string `json:"version"`
	Arch        string `json:"arch,omitempty"`
	CVECount    int    `json:"cveCount"`
	TopSeverity int32  `json:"topSeverity"`
	Fixable     bool   `json:"fixable"`
}

// ComponentImagesResponse is the response for GET /v1/scandata/components/{componentName}/images
type ComponentImagesResponse struct {
	Images []ComponentImageInfo `json:"images"`
}

// ComponentDetailResponse is the response for GET /v1/scandata/components/{componentName}
type ComponentDetailResponse struct {
	Name     string                 `json:"name"`
	Versions []ComponentVersionInfo `json:"versions"`
}

// ComponentVersionInfo represents one version of a component with CVE data
type ComponentVersionInfo struct {
	Version     string  `json:"version"`
	Source      string  `json:"source"`
	Arch        string  `json:"arch,omitzero"`
	Module      string  `json:"module,omitzero"`
	CVECount    int     `json:"cveCount"`
	ImageCount  int     `json:"imageCount"`
	TopSeverity int32   `json:"topSeverity"`
	TopCVSS     float32 `json:"topCvss"`
	Fixable     bool    `json:"fixable"`
	FixedBy     string  `json:"fixedBy,omitempty"`
}

func (h *handler) listCVEs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	limitStr := cmp.Or(r.URL.Query().Get("limit"), "20")
	offsetStr := cmp.Or(r.URL.Query().Get("offset"), "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.InvalidArgument, errors.Wrap(err, "invalid limit"))
		return
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.InvalidArgument, errors.Wrap(err, "invalid offset"))
		return
	}

	// Query data
	rows, total, err := h.datastore.ListCVEs(ctx, limit, offset)
	if err != nil {
		log.Errorf("failed to list CVEs: %v", err)
		httputil.WriteGRPCStyleError(w, codes.Internal, errors.Wrap(err, "listing CVEs"))
		return
	}

	// Convert to response
	items := make([]CVEListItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, CVEListItem{
			CVEName:         row.CVEName,
			Severity:        row.Severity,
			CVSS:            row.CVSS,
			ImageCount:      row.ImageCount,
			Fixable:         row.Fixable,
			FirstSeen:       row.FirstSeen,
			PublishedDate:   row.PublishedDate,
			EPSSProbability: row.EPSSProbability,
		})
	}

	response := CVEListResponse{
		CVEs:       items,
		TotalCount: total,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Errorf("failed to encode response: %v", err)
	}
}

func (h *handler) getCVEDetail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vars := mux.Vars(r)
	cveName := vars["cveName"]
	if cveName == "" {
		httputil.WriteGRPCStyleError(w, codes.InvalidArgument, errors.New("cveName is required"))
		return
	}

	// Get all findings with component data via a single JOIN query.
	findingsWithComps, err := h.datastore.GetFindingsWithComponentsByCVE(ctx, cveName)
	if err != nil {
		log.Errorf("failed to get findings for CVE %s: %v", cveName, err)
		httputil.WriteGRPCStyleError(w, codes.Internal, errors.Wrap(err, "getting CVE findings"))
		return
	}

	if len(findingsWithComps) == 0 {
		httputil.WriteGRPCStyleError(w, codes.NotFound, errors.Errorf("CVE %s not found", cveName))
		return
	}

	response := buildCVEDetailResponse(ctx, h.datastore, cveName, findingsWithComps)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Errorf("failed to encode response: %v", err)
	}
}

// advisoryLink constructs a URL for a known advisory ID prefix.
func advisoryLink(id string) string {
	switch {
	case strings.HasPrefix(id, "GHSA-"):
		return "https://github.com/advisories/" + id
	case strings.HasPrefix(id, "GO-"):
		return "https://pkg.go.dev/vuln/" + id
	case strings.HasPrefix(id, "RHSA-"):
		return "https://access.redhat.com/errata/" + id
	default:
		return ""
	}
}

func buildCVEDetailResponse(ctx context.Context, ds datastore.DataStore, cveName string, findings []*types.FindingWithComponent) *CVEDetailResponse {
	var maxSeverity int32
	var maxCVSS float32

	advisoryMap := make(map[string]*AdvisoryInfo)

	// Group components by name|version|source so the same component across images is one entry.
	componentMap := make(map[string]*ComponentInfo)
	// Track which images we've already counted for each component.
	compImageSeen := make(map[string]map[string]bool)

	// Track per-image data including which components affect each image.
	imageMap := make(map[string]*ImageInfo)
	// Track per-image component indices so we can append advisories to existing entries.
	imageCompIdx := make(map[string]map[string]int)

	for _, fc := range findings {
		f := fc.Finding

		// Global max severity / CVSS
		if f.GetSeverity() > storage.VulnerabilitySeverity(maxSeverity) {
			maxSeverity = int32(f.GetSeverity())
		}
		if f.GetCvss() > maxCVSS {
			maxCVSS = f.GetCvss()
		}

		// Advisory (dedup by advisory ID)
		advisoryID := f.GetAdvisoryId()
		if advisoryID != "" && advisoryMap[advisoryID] == nil {
			advisoryMap[advisoryID] = &AdvisoryInfo{
				ID:          advisoryID,
				Severity:    int32(f.GetSeverity()),
				CVSS:        f.GetCvss(),
				SourceName:  f.GetSourceName(),
				Description: f.GetDescription(),
				Link:        advisoryLink(advisoryID),
				FixedBy:     f.GetFixedBy(),
			}
		}

		// Component -- group by name|version|source, not by per-image component ID.
		compSource := storage.SourceType(fc.ComponentSource).String()
		compKey := fmt.Sprintf("%s|%s|%s", fc.ComponentName, fc.ComponentVersion, compSource)
		if componentMap[compKey] == nil {
			componentMap[compKey] = &ComponentInfo{
				Name:       fc.ComponentName,
				Version:    fc.ComponentVersion,
				Source:     compSource,
				FixedBy:    f.GetFixedBy(),
				ImageCount: 0,
			}
			compImageSeen[compKey] = make(map[string]bool)
		}
		// Count each image only once per component (multiple advisories inflate the count otherwise).
		if !compImageSeen[compKey][f.GetImageId()] {
			compImageSeen[compKey][f.GetImageId()] = true
			componentMap[compKey].ImageCount++
		}

		// Image -- track per-image components for expandable rows.
		imageID := f.GetImageId()
		if imageMap[imageID] == nil {
			imageMap[imageID] = &ImageInfo{
				ImageID:        imageID,
				ComponentCount: 0,
				Severity:       int32(f.GetSeverity()),
				Fixable:        f.GetIsFixable(),
				Components:     nil,
			}
			imageCompIdx[imageID] = make(map[string]int)
		} else {
			if f.GetSeverity() > storage.VulnerabilitySeverity(imageMap[imageID].Severity) {
				imageMap[imageID].Severity = int32(f.GetSeverity())
			}
			if f.GetIsFixable() {
				imageMap[imageID].Fixable = true
			}
		}
		// Per-image components: dedup by name+version, collect advisory IDs.
		imgCompKey := fmt.Sprintf("%s|%s", fc.ComponentName, fc.ComponentVersion)
		if idx, seen := imageCompIdx[imageID][imgCompKey]; seen {
			// Component already tracked — just append advisory ID.
			imageMap[imageID].Components[idx].Advisories = append(
				imageMap[imageID].Components[idx].Advisories, f.GetAdvisoryId())
		} else {
			// New component for this image.
			imageCompIdx[imageID][imgCompKey] = len(imageMap[imageID].Components)
			imageMap[imageID].ComponentCount++
			imageMap[imageID].Components = append(imageMap[imageID].Components, ImageComponentInfo{
				Name:       fc.ComponentName,
				Version:    fc.ComponentVersion,
				Source:     compSource,
				FixedBy:    f.GetFixedBy(),
				Advisories: []string{f.GetAdvisoryId()},
			})
		}
	}

	// Convert maps to slices.
	advisories := make([]AdvisoryInfo, 0, len(advisoryMap))
	for _, adv := range advisoryMap {
		advisories = append(advisories, *adv)
	}

	// Sort advisories by severity DESC, then CVSS DESC to pick the best description.
	slices.SortFunc(advisories, func(a, b AdvisoryInfo) int {
		if a.Severity != b.Severity {
			return int(b.Severity) - int(a.Severity)
		}
		if a.CVSS != b.CVSS {
			if b.CVSS > a.CVSS {
				return 1
			}
			return -1
		}
		return 0
	})

	// Use the description from the highest-severity advisory.
	var description string
	for _, adv := range advisories {
		if adv.Description != "" {
			description = adv.Description
			break
		}
	}

	components := make([]ComponentInfo, 0, len(componentMap))
	for _, comp := range componentMap {
		components = append(components, *comp)
	}

	// Enrich images with name and UUID from images_v2.
	digests := make([]string, 0, len(imageMap))
	for digest := range imageMap {
		digests = append(digests, digest)
	}
	imageInfoMap, err := ds.GetImageInfoByDigests(ctx, digests)
	if err != nil {
		log.Warnf("failed to enrich image info: %v", err)
	}

	images := make([]ImageInfo, 0, len(imageMap))
	for _, img := range imageMap {
		if info, ok := imageInfoMap[img.ImageID]; ok {
			img.ImageUUID = info.UUID
			img.ImageName = info.FullName
		}
		images = append(images, *img)
	}

	return &CVEDetailResponse{
		CVEName:     cveName,
		Severity:    maxSeverity,
		CVSS:        maxCVSS,
		Description: description,
		Advisories:  advisories,
		Components:  components,
		Images:      images,
	}
}

func (h *handler) getImageDetail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vars := mux.Vars(r)
	imageID := vars["imageId"]
	if imageID == "" {
		httputil.WriteGRPCStyleError(w, codes.InvalidArgument, errors.New("imageId is required"))
		return
	}

	// Get scan metadata.
	scanData, err := h.datastore.GetScanDataByImageID(ctx, imageID)
	if err != nil {
		log.Errorf("failed to get scan data for image %s: %v", imageID, err)
		httputil.WriteGRPCStyleError(w, codes.Internal, errors.Wrap(err, "getting image scan data"))
		return
	}
	if scanData == nil || scanData.Scan == nil {
		httputil.WriteGRPCStyleError(w, codes.NotFound, errors.Errorf("image %s not found", imageID))
		return
	}

	// Get findings joined with component data.
	findingsWithComps, err := h.datastore.GetFindingsWithComponentsByImageID(ctx, imageID)
	if err != nil {
		log.Errorf("failed to get findings for image %s: %v", imageID, err)
		httputil.WriteGRPCStyleError(w, codes.Internal, errors.Wrap(err, "getting image findings"))
		return
	}

	// Look up image name.
	var imageName string
	imageInfoMap, err := h.datastore.GetImageInfoByDigests(ctx, []string{imageID})
	if err != nil {
		log.Warnf("failed to enrich image info for %s: %v", imageID, err)
	} else if info, ok := imageInfoMap[imageID]; ok {
		imageName = info.FullName
	}

	// Look up image OS.
	imageOS, err := h.datastore.GetImageOS(ctx, imageID)
	if err != nil {
		log.Warnf("failed to get image OS for %s: %v", imageID, err)
	}

	response := buildImageDetailResponse(imageID, imageName, imageOS, scanData, findingsWithComps)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Errorf("failed to encode response: %v", err)
	}
}

// buildImageDetailResponse groups findings by component, then by CVE within each component.
func buildImageDetailResponse(imageID, imageName, imageOS string, scanData *types.ScanData, findings []*types.FindingWithComponent) *ImageDetailResponse {
	scan := scanData.Scan

	// Collect data sources.
	dataSourceSet := make(map[string]struct{})
	for _, fc := range findings {
		src := fc.Finding.GetSourceName()
		if src != "" {
			dataSourceSet[src] = struct{}{}
		}
	}
	dataSources := make([]string, 0, len(dataSourceSet))
	for ds := range dataSourceSet {
		dataSources = append(dataSources, ds)
	}
	slices.Sort(dataSources)

	// Group findings by component (name+version+source), then by CVE within each component.
	type cveKey struct {
		cveName string
	}
	type compKey struct {
		name    string
		version string
		source  string
	}

	type cveAccum struct {
		severity   int32
		cvss       float32
		fixedBy    string
		advisories []string
	}

	compOrder := make([]compKey, 0)
	compLocation := make(map[compKey]string)
	compArch := make(map[compKey]string)
	compCVEs := make(map[compKey]map[string]*cveAccum)

	// Track unique CVE names across all components for the summary.
	allCVEs := make(map[string]int32) // cveName -> maxSeverity

	for _, fc := range findings {
		f := fc.Finding
		compSource := storage.SourceType(fc.ComponentSource).String()
		ck := compKey{name: fc.ComponentName, version: fc.ComponentVersion, source: compSource}

		if _, exists := compCVEs[ck]; !exists {
			compOrder = append(compOrder, ck)
			compCVEs[ck] = make(map[string]*cveAccum)
			compLocation[ck] = fc.ComponentLocation
			compArch[ck] = fc.ComponentArch
		}

		cveName := f.GetCveName()
		if cveName == "" {
			continue
		}

		// Track max severity for summary.
		if int32(f.GetSeverity()) > allCVEs[cveName] {
			allCVEs[cveName] = int32(f.GetSeverity())
		}

		if existing, ok := compCVEs[ck][cveName]; ok {
			// Same CVE, different advisory — append advisory ID.
			if aid := f.GetAdvisoryId(); aid != "" {
				existing.advisories = append(existing.advisories, aid)
			}
			if f.GetCvss() > existing.cvss {
				existing.cvss = f.GetCvss()
			}
			if int32(f.GetSeverity()) > existing.severity {
				existing.severity = int32(f.GetSeverity())
			}
			if existing.fixedBy == "" && f.GetFixedBy() != "" {
				existing.fixedBy = f.GetFixedBy()
			}
		} else {
			var advisories []string
			if aid := f.GetAdvisoryId(); aid != "" {
				advisories = []string{aid}
			}
			compCVEs[ck][cveName] = &cveAccum{
				severity:   int32(f.GetSeverity()),
				cvss:       f.GetCvss(),
				fixedBy:    f.GetFixedBy(),
				advisories: advisories,
			}
		}
	}

	// Build component list.
	components := make([]ImageDetailComponent, 0, len(compOrder))
	for _, ck := range compOrder {
		cves := compCVEs[ck]
		cveList := make([]ImageDetailCVE, 0, len(cves))
		for cveName, acc := range cves {
			cveList = append(cveList, ImageDetailCVE{
				CVEName:    cveName,
				Severity:   acc.severity,
				CVSS:       acc.cvss,
				FixedBy:    acc.fixedBy,
				Advisories: acc.advisories,
			})
		}
		// Sort CVEs by severity DESC, then CVSS DESC.
		slices.SortFunc(cveList, func(a, b ImageDetailCVE) int {
			if a.Severity != b.Severity {
				return int(b.Severity) - int(a.Severity)
			}
			if a.CVSS != b.CVSS {
				if b.CVSS > a.CVSS {
					return 1
				}
				return -1
			}
			return strings.Compare(a.CVEName, b.CVEName)
		})

		components = append(components, ImageDetailComponent{
			Name:     ck.name,
			Version:  ck.version,
			Source:   ck.source,
			Arch:     compArch[ck],
			Location: compLocation[ck],
			CVEs:     cveList,
		})
	}

	// Build CVE summary.
	summary := ImageDetailCVESummary{Total: len(allCVEs)}
	for _, sev := range allCVEs {
		switch sev {
		case 4:
			summary.Critical++
		case 3:
			summary.Important++
		case 2:
			summary.Moderate++
		case 1:
			summary.Low++
		}
	}

	var scanTime string
	if scan.GetScanTime() != nil {
		scanTime = scan.GetScanTime().AsTime().Format(time.RFC3339)
	}

	return &ImageDetailResponse{
		ImageID:        imageID,
		ImageName:      imageName,
		ImageOS:        imageOS,
		ScanTime:       scanTime,
		ScannerVersion: scan.GetScannerVersion(),
		BundleVersion:  scan.GetBundleVersion(),
		DataSources:    dataSources,
		Components:     components,
		CVESummary:     summary,
	}
}

func (h *handler) getImageFindings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vars := mux.Vars(r)
	imageID := vars["imageId"]
	if imageID == "" {
		httputil.WriteGRPCStyleError(w, codes.InvalidArgument, errors.New("imageId is required"))
		return
	}

	// Get scan data to access both findings and components
	scanData, err := h.datastore.GetScanDataByImageID(ctx, imageID)
	if err != nil {
		log.Errorf("failed to get scan data for image %s: %v", imageID, err)
		httputil.WriteGRPCStyleError(w, codes.Internal, errors.Wrap(err, "getting image scan data"))
		return
	}

	if scanData == nil || scanData.Scan == nil {
		httputil.WriteGRPCStyleError(w, codes.NotFound, errors.Errorf("image %s not found", imageID))
		return
	}

	// Build component map for lookup
	componentMap := make(map[string]*storage.ScanComponent)
	for _, comp := range scanData.Components {
		componentMap[comp.GetId()] = comp
	}

	// Build findings with component info
	findingsWithComp := make([]FindingWithComponent, 0, len(scanData.Findings))
	for _, f := range scanData.Findings {
		comp := componentMap[f.GetComponentId()]
		if comp == nil {
			log.Warnf("component %s not found for finding %s", f.GetComponentId(), f.GetId())
			continue
		}

		findingsWithComp = append(findingsWithComp, FindingWithComponent{
			CVEName:          f.GetCveName(),
			Severity:         int32(f.GetSeverity()),
			CVSS:             f.GetCvss(),
			IsFixable:        f.GetIsFixable(),
			FixedBy:          f.GetFixedBy(),
			ComponentName:    comp.GetName(),
			ComponentVersion: comp.GetVersion(),
			ComponentSource:  comp.GetSource().String(),
			SourceName:       f.GetSourceName(),
		})
	}

	response := ImageFindingsResponse{
		ImageID:  imageID,
		Findings: findingsWithComp,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Errorf("failed to encode response: %v", err)
	}
}

func (h *handler) listImages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	limitStr := cmp.Or(r.URL.Query().Get("limit"), "50")
	offsetStr := cmp.Or(r.URL.Query().Get("offset"), "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.InvalidArgument, errors.Wrap(err, "invalid limit"))
		return
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.InvalidArgument, errors.Wrap(err, "invalid offset"))
		return
	}

	// Query data
	rows, total, err := h.datastore.ListImages(ctx, limit, offset)
	if err != nil {
		log.Errorf("failed to list images: %v", err)
		httputil.WriteGRPCStyleError(w, codes.Internal, errors.Wrap(err, "listing images"))
		return
	}

	// Enrich with image name, UUID, and OS.
	digests := make([]string, 0, len(rows))
	for _, row := range rows {
		digests = append(digests, row.ImageID)
	}
	imageInfoMap, enrichErr := h.datastore.GetImageInfoByDigests(ctx, digests)
	if enrichErr != nil {
		log.Warnf("failed to enrich image info: %v", enrichErr)
	}

	// Convert to response
	items := make([]ImageListItem, 0, len(rows))
	for _, row := range rows {
		item := ImageListItem{
			ImageID:        row.ImageID,
			CVECount:       row.CVECount,
			ComponentCount: row.ComponentCount,
			TopSeverity:    row.TopSeverity,
			TopCVSS:        row.TopCVSS,
			Fixable:        row.Fixable,
			ScanTime:       row.ScanTime,
			CriticalCount:  row.CriticalCount,
			ImportantCount: row.ImportantCount,
			ModerateCount:  row.ModerateCount,
			LowCount:       row.LowCount,
		}
		if info, ok := imageInfoMap[row.ImageID]; ok {
			item.ImageUUID = info.UUID
			item.ImageName = info.FullName
		}
		// Get OS for each image
		imageOS, osErr := h.datastore.GetImageOS(ctx, row.ImageID)
		if osErr != nil {
			log.Warnf("failed to get image OS for %s: %v", row.ImageID, osErr)
		}
		item.ImageOS = imageOS
		items = append(items, item)
	}

	response := ImageListResponse{
		Images:     items,
		TotalCount: total,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Errorf("failed to encode response: %v", err)
	}
}

func (h *handler) listAdvisories(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	limitStr := cmp.Or(r.URL.Query().Get("limit"), "50")
	offsetStr := cmp.Or(r.URL.Query().Get("offset"), "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.InvalidArgument, errors.Wrap(err, "invalid limit"))
		return
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.InvalidArgument, errors.Wrap(err, "invalid offset"))
		return
	}

	// Query data
	rows, total, err := h.datastore.ListAdvisories(ctx, limit, offset)
	if err != nil {
		log.Errorf("failed to list advisories: %v", err)
		httputil.WriteGRPCStyleError(w, codes.Internal, errors.Wrap(err, "listing advisories"))
		return
	}

	// Convert to response
	items := make([]AdvisoryListItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, AdvisoryListItem{
			AdvisoryID:     row.AdvisoryID,
			CVEName:        row.CVEName,
			Severity:       row.Severity,
			CVSS:           row.CVSS,
			SourceName:     row.SourceName,
			Description:    row.Description,
			FixedBy:        row.FixedBy,
			ImageCount:     row.ImageCount,
			ComponentCount: row.ComponentCount,
			Link:           advisoryLink(row.AdvisoryID),
		})
	}

	response := AdvisoryListResponse{
		Advisories: items,
		TotalCount: total,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Errorf("failed to encode response: %v", err)
	}
}

func (h *handler) listDeployments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	limitStr := cmp.Or(r.URL.Query().Get("limit"), "50")
	offsetStr := cmp.Or(r.URL.Query().Get("offset"), "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.InvalidArgument, errors.Wrap(err, "invalid limit"))
		return
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.InvalidArgument, errors.Wrap(err, "invalid offset"))
		return
	}

	// Query data
	rows, total, err := h.datastore.ListDeployments(ctx, limit, offset)
	if err != nil {
		log.Errorf("failed to list deployments: %v", err)
		httputil.WriteGRPCStyleError(w, codes.Internal, errors.Wrap(err, "listing deployments"))
		return
	}

	// Convert to response
	items := make([]DeploymentListItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, DeploymentListItem{
			ID:          row.ID,
			Name:        row.Name,
			Cluster:     row.ClusterName,
			Namespace:   row.Namespace,
			ImageCount:  row.ImageCount,
			CVECount:    row.CVECount,
			TopSeverity: row.TopSeverity,
			Fixable:     row.Fixable,
		})
	}

	response := DeploymentListResponse{
		Deployments: items,
		TotalCount:  total,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Errorf("failed to encode response: %v", err)
	}
}

func (h *handler) getDeploymentDetail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vars := mux.Vars(r)
	deploymentID := vars["deploymentId"]
	if deploymentID == "" {
		httputil.WriteGRPCStyleError(w, codes.InvalidArgument, errors.New("deploymentId is required"))
		return
	}

	// Get deployment basic info
	deployment, err := h.datastore.GetDeploymentByID(ctx, deploymentID)
	if err != nil {
		log.Errorf("failed to get deployment %s: %v", deploymentID, err)
		httputil.WriteGRPCStyleError(w, codes.Internal, errors.Wrap(err, "getting deployment"))
		return
	}
	if deployment == nil {
		httputil.WriteGRPCStyleError(w, codes.NotFound, errors.Errorf("deployment %s not found", deploymentID))
		return
	}

	// Get images with CVE summary
	images, err := h.datastore.GetDeploymentImages(ctx, deploymentID)
	if err != nil {
		log.Errorf("failed to get images for deployment %s: %v", deploymentID, err)
		httputil.WriteGRPCStyleError(w, codes.Internal, errors.Wrap(err, "getting deployment images"))
		return
	}

	// Convert to response
	imageDetails := make([]DeploymentImageDetail, 0, len(images))
	for _, img := range images {
		imageDetails = append(imageDetails, DeploymentImageDetail{
			ImageID:     img.ImageID,
			ImageUUID:   img.ImageUUID,
			ImageName:   img.ImageName,
			CVECount:    img.CVECount,
			TopSeverity: img.TopSeverity,
			Fixable:     img.Fixable,
		})
	}

	response := DeploymentDetailResponse{
		ID:        deploymentID,
		Name:      deployment.Name,
		Cluster:   deployment.ClusterName,
		Namespace: deployment.Namespace,
		Images:    imageDetails,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Errorf("failed to encode response: %v", err)
	}
}

func (h *handler) listComponents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	limitStr := cmp.Or(r.URL.Query().Get("limit"), "50")
	offsetStr := cmp.Or(r.URL.Query().Get("offset"), "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.InvalidArgument, errors.Wrap(err, "invalid limit"))
		return
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.InvalidArgument, errors.Wrap(err, "invalid offset"))
		return
	}

	// Query data
	rows, total, err := h.datastore.ListComponents(ctx, limit, offset)
	if err != nil {
		log.Errorf("failed to list components: %v", err)
		httputil.WriteGRPCStyleError(w, codes.Internal, errors.Wrap(err, "listing components"))
		return
	}

	// Convert to response
	items := make([]ComponentListItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, ComponentListItem{
			Name:           row.Name,
			VersionCount:   row.VersionCount,
			CVECount:       row.CVECount,
			ImageCount:     row.ImageCount,
			TopSeverity:    row.TopSeverity,
			TopCVSS:        row.TopCVSS,
			CriticalCount:  row.CriticalCount,
			ImportantCount: row.ImportantCount,
			ModerateCount:  row.ModerateCount,
			LowCount:       row.LowCount,
		})
	}

	response := ComponentListResponse{
		Components: items,
		TotalCount: total,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Errorf("failed to encode response: %v", err)
	}
}

func (h *handler) getComponentImages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	componentName := r.URL.Query().Get("name")
	if componentName == "" {
		httputil.WriteGRPCStyleError(w, codes.InvalidArgument, errors.New("name query parameter is required"))
		return
	}

	rows, err := h.datastore.GetComponentImages(ctx, componentName)
	if err != nil {
		log.Errorf("failed to get images for component %s: %v", componentName, err)
		httputil.WriteGRPCStyleError(w, codes.Internal, errors.Wrap(err, "getting component images"))
		return
	}

	// Enrich with image name and UUID from images_v2.
	digests := make([]string, 0, len(rows))
	for _, row := range rows {
		digests = append(digests, row.ImageID)
	}
	imageInfoMap, enrichErr := h.datastore.GetImageInfoByDigests(ctx, digests)
	if enrichErr != nil {
		log.Warnf("failed to enrich image info for component %s: %v", componentName, enrichErr)
	}

	items := make([]ComponentImageInfo, 0, len(rows))
	for _, row := range rows {
		item := ComponentImageInfo{
			ImageID:     row.ImageID,
			Version:     row.Version,
			Arch:        row.Arch,
			CVECount:    row.CVECount,
			TopSeverity: row.TopSeverity,
			Fixable:     row.Fixable,
		}
		if info, ok := imageInfoMap[row.ImageID]; ok {
			item.ImageUUID = info.UUID
			item.ImageName = info.FullName
		}
		items = append(items, item)
	}

	response := ComponentImagesResponse{Images: items}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Errorf("failed to encode response: %v", err)
	}
}

func (h *handler) getComponentCVEs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	componentName := r.URL.Query().Get("name")
	if componentName == "" {
		httputil.WriteGRPCStyleError(w, codes.InvalidArgument, errors.New("name query parameter is required"))
		return
	}

	componentVersion := r.URL.Query().Get("version")
	if componentVersion == "" {
		httputil.WriteGRPCStyleError(w, codes.InvalidArgument, errors.New("version query parameter is required"))
		return
	}

	rows, err := h.datastore.GetComponentCVEs(ctx, componentName, componentVersion)
	if err != nil {
		log.Errorf("failed to get CVEs for component %s@%s: %v", componentName, componentVersion, err)
		httputil.WriteGRPCStyleError(w, codes.Internal, errors.Wrap(err, "getting component CVEs"))
		return
	}

	cves := make([]ComponentCVE, 0, len(rows))
	for _, row := range rows {
		cves = append(cves, ComponentCVE{
			CVEName:     row.CVEName,
			Severity:    row.Severity,
			CVSS:        row.CVSS,
			Fixable:     row.Fixable,
			FixedBy:     row.FixedBy,
			Description: row.Description,
			ImageCount:  row.ImageCount,
		})
	}

	response := ComponentCVEsResponse{
		ComponentName:    componentName,
		ComponentVersion: componentVersion,
		CVEs:             cves,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Errorf("failed to encode response: %v", err)
	}
}

func (h *handler) getComponentDetail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	componentName := r.URL.Query().Get("name")
	if componentName == "" {
		httputil.WriteGRPCStyleError(w, codes.InvalidArgument, errors.New("name query parameter is required"))
		return
	}

	// Get component versions
	versions, err := h.datastore.GetComponentVersions(ctx, componentName)
	if err != nil {
		log.Errorf("failed to get versions for component %s: %v", componentName, err)
		httputil.WriteGRPCStyleError(w, codes.Internal, errors.Wrap(err, "getting component versions"))
		return
	}

	if len(versions) == 0 {
		httputil.WriteGRPCStyleError(w, codes.NotFound, errors.Errorf("component %s not found", componentName))
		return
	}

	// Convert to response
	versionInfos := make([]ComponentVersionInfo, 0, len(versions))
	for _, v := range versions {
		versionInfos = append(versionInfos, ComponentVersionInfo{
			Version:     v.Version,
			Source:      v.Source,
			Arch:        v.Arch,
			Module:      v.Module,
			CVECount:    v.CVECount,
			ImageCount:  v.ImageCount,
			TopSeverity: v.TopSeverity,
			TopCVSS:     v.TopCVSS,
			Fixable:     v.Fixable,
			FixedBy:     v.FixedBy,
		})
	}

	response := ComponentDetailResponse{
		Name:     componentName,
		Versions: versionInfos,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Errorf("failed to encode response: %v", err)
	}
}

// Handler returns the HTTP handler for scan data API routes
func Handler(ds datastore.DataStore) http.Handler {
	return NewHandler(ds)
}
