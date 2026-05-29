package api

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
	router.HandleFunc("/v1/scandata/images/{imageId}/findings", h.getImageFindings).Methods(http.MethodGet)
	router.HandleFunc("/v1/scandata/deployments", h.listDeployments).Methods(http.MethodGet)
	router.HandleFunc("/v1/scandata/deployments/{deploymentId}", h.getDeploymentDetail).Methods(http.MethodGet)

	return router
}

// CVEListResponse is the response for GET /v1/scandata/cves
type CVEListResponse struct {
	CVEs       []CVEListItem `json:"cves"`
	TotalCount int           `json:"totalCount"`
}

// CVEListItem represents one CVE in the list
type CVEListItem struct {
	CVEName    string     `json:"cveName"`
	Severity   int32      `json:"severity"`
	CVSS       float32    `json:"cvss"`
	ImageCount int        `json:"imageCount"`
	Fixable    bool       `json:"fixable"`
	FirstSeen  *time.Time `json:"firstSeen,omitzero"`
}

// CVEDetailResponse is the response for GET /v1/scandata/cves/{cveName}
type CVEDetailResponse struct {
	CVEName    string          `json:"cveName"`
	Severity   int32           `json:"severity"`
	CVSS       float32         `json:"cvss"`
	Advisories []AdvisoryInfo  `json:"advisories"`
	Components []ComponentInfo `json:"components"`
	Images     []ImageInfo     `json:"images"`
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
			CVEName:    row.CVEName,
			Severity:   row.Severity,
			CVSS:       row.CVSS,
			ImageCount: row.ImageCount,
			Fixable:    row.Fixable,
			FirstSeen:  row.FirstSeen,
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
		CVEName:    cveName,
		Severity:   maxSeverity,
		CVSS:       maxCVSS,
		Advisories: advisories,
		Components: components,
		Images:     images,
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

// Handler returns the HTTP handler for scan data API routes
func Handler(ds datastore.DataStore) http.Handler {
	return NewHandler(ds)
}
