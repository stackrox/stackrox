package api

import (
	"cmp"
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/scandata/datastore"
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
	CVEName    string              `json:"cveName"`
	Severity   int32               `json:"severity"`
	CVSS       float32             `json:"cvss"`
	Advisories []AdvisoryInfo      `json:"advisories"`
	Components []ComponentInfo     `json:"components"`
	Images     []ImageInfo         `json:"images"`
}

// AdvisoryInfo represents an advisory for a CVE
type AdvisoryInfo struct {
	ID         string  `json:"id"`
	Severity   int32   `json:"severity"`
	CVSS       float32 `json:"cvss"`
	SourceName string  `json:"sourceName"`
}

// ComponentInfo represents a component affected by a CVE
type ComponentInfo struct {
	Name       string `json:"name"`
	Version    string `json:"version"`
	Source     string `json:"source"`
	FixedBy    string `json:"fixedBy,omitzero"`
	ImageCount int    `json:"imageCount"`
}

// ImageInfo represents an image affected by a CVE
type ImageInfo struct {
	ImageID        string `json:"imageId"`
	ComponentCount int    `json:"componentCount"`
	Severity       int32  `json:"severity"`
	Fixable        bool   `json:"fixable"`
}

// ImageFindingsResponse is the response for GET /v1/scandata/images/{imageId}/findings
type ImageFindingsResponse struct {
	ImageID  string                `json:"imageId"`
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

	// Get all findings for this CVE
	findings, err := h.datastore.GetFindingsByCVE(ctx, cveName)
	if err != nil {
		log.Errorf("failed to get findings for CVE %s: %v", cveName, err)
		httputil.WriteGRPCStyleError(w, codes.Internal, errors.Wrap(err, "getting CVE findings"))
		return
	}

	if len(findings) == 0 {
		httputil.WriteGRPCStyleError(w, codes.NotFound, errors.Errorf("CVE %s not found", cveName))
		return
	}

	// Build response by aggregating findings
	response := buildCVEDetailResponse(ctx, h.datastore, cveName, findings)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Errorf("failed to encode response: %v", err)
	}
}

func buildCVEDetailResponse(ctx context.Context, ds datastore.DataStore, cveName string, findings []*storage.ScanFinding) *CVEDetailResponse {
	// Track max severity and CVSS
	var maxSeverity int32
	var maxCVSS float32

	// Track unique advisories
	advisoryMap := make(map[string]*AdvisoryInfo)

	// Track unique components by component ID
	componentMap := make(map[string]*ComponentInfo)

	// Track unique images
	imageMap := make(map[string]*ImageInfo)

	for _, f := range findings {
		// Update max severity and CVSS
		if f.GetSeverity() > storage.VulnerabilitySeverity(maxSeverity) {
			maxSeverity = int32(f.GetSeverity())
		}
		if f.GetCvss() > maxCVSS {
			maxCVSS = f.GetCvss()
		}

		// Track advisory
		advisoryID := f.GetAdvisoryId()
		if advisoryID != "" && advisoryMap[advisoryID] == nil {
			advisoryMap[advisoryID] = &AdvisoryInfo{
				ID:         advisoryID,
				Severity:   int32(f.GetSeverity()),
				CVSS:       f.GetCvss(),
				SourceName: f.GetSourceName(),
			}
		}

		// Track component (need to fetch component details)
		compID := f.GetComponentId()
		if componentMap[compID] == nil {
			componentMap[compID] = &ComponentInfo{
				Name:       "", // Will be populated below
				Version:    "", // Will be populated below
				Source:     "", // Will be populated below
				FixedBy:    f.GetFixedBy(),
				ImageCount: 0,
			}
		}
		componentMap[compID].ImageCount++

		// Track image
		imageID := f.GetImageId()
		if imageMap[imageID] == nil {
			imageMap[imageID] = &ImageInfo{
				ImageID:        imageID,
				ComponentCount: 0,
				Severity:       int32(f.GetSeverity()),
				Fixable:        f.GetIsFixable(),
			}
		} else {
			// Update severity to max
			if f.GetSeverity() > storage.VulnerabilitySeverity(imageMap[imageID].Severity) {
				imageMap[imageID].Severity = int32(f.GetSeverity())
			}
			// Update fixable if any finding is fixable
			if f.GetIsFixable() {
				imageMap[imageID].Fixable = true
			}
		}
		imageMap[imageID].ComponentCount++
	}

	// Populate component names/versions (simple approach: query each image's scan data)
	// For a prototype, we'll do this inefficiently. Production would use a JOIN query.
	for compID, compInfo := range componentMap {
		// Find any finding with this component ID
		for _, f := range findings {
			if f.GetComponentId() == compID {
				// Get the scan data to find component details
				scanData, err := ds.GetScanDataByImageID(ctx, f.GetImageId())
				if err != nil {
					log.Warnf("failed to get scan data for image %s: %v", f.GetImageId(), err)
					continue
				}

				// Find the component
				for _, comp := range scanData.Components {
					if comp.GetId() == compID {
						compInfo.Name = comp.GetName()
						compInfo.Version = comp.GetVersion()
						compInfo.Source = comp.GetSource().String()
						break
					}
				}

				if compInfo.Name != "" {
					break
				}
			}
		}
	}

	// Convert maps to slices
	advisories := make([]AdvisoryInfo, 0, len(advisoryMap))
	for _, adv := range advisoryMap {
		advisories = append(advisories, *adv)
	}

	components := make([]ComponentInfo, 0, len(componentMap))
	for _, comp := range componentMap {
		components = append(components, *comp)
	}

	images := make([]ImageInfo, 0, len(imageMap))
	for _, img := range imageMap {
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

// Handler returns the HTTP handler for scan data API routes
func Handler(ds datastore.DataStore) http.Handler {
	return NewHandler(ds)
}
