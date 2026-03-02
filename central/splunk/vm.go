package splunk

import (
	"net/http"
	"time"

	"github.com/stackrox/rox/central/deployment/datastore"
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	imageV2Datastore "github.com/stackrox/rox/central/imagev2/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/jsonutil"
	"github.com/stackrox/rox/pkg/protoconv"
)

type splunkDeploymentEvent struct {
	Type        string            `json:"type"`
	Cluster     string            `json:"cluster"`
	Namespace   string            `json:"namespace"`
	Deployment  string            `json:"deployment"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	ImageDigest string            `json:"imageDigest"`
}

type splunkImageEvent struct {
	Type          string    `json:"type"`
	ImageDigest   string    `json:"imageDigest"`
	ImageRegistry string    `json:"imageRegistry"`
	ImageRepo     string    `json:"imageRepo"`
	ImageTag      string    `json:"imageTag"`
	OS            string    `json:"os"`
	CreatedTime   time.Time `json:"createdTime"`
}

type splunkCVEEvent struct {
	Type        string    `json:"type"`
	ImageDigest string    `json:"imageDigest"`
	Component   string    `json:"component"`
	Version     string    `json:"version"`
	CVE         string    `json:"cve"`
	CVSS        float32   `json:"cvss"`
	FixedBy     string    `json:"fixedBy"`
	FirstSeen   time.Time `json:"firstSeen"`
	Source      string    `json:"source"`
}

// imageFields is satisfied by both *storage.Image and *storage.ImageV2,
// providing access to the shared fields needed for Splunk events.
type imageFields interface {
	GetScan() *storage.ImageScan
	GetMetadata() *storage.ImageMetadata
	GetName() *storage.ImageName
}

// NewVulnMgmtHandler returns an http.HandlerFunc implementation that returns all the required events for the Splunk TA
func NewVulnMgmtHandler(deployments datastore.DataStore, images imageDatastore.DataStore, imagesV2 imageV2Datastore.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		arrayWriter := jsonutil.NewJSONArrayWriter(w)
		if err := arrayWriter.Init(); err != nil {
			httputil.WriteError(w, err)
			return
		}

		ids, err := deployments.GetDeploymentIDs(r.Context())
		if err != nil {
			httputil.WriteError(w, err)
			return
		}

		v2Enabled := features.FlattenImageData.Enabled()

		// imageLookup maps image SHA digest to the datastore lookup key.
		// For V1, the lookup key is the digest itself.
		// For V2, the lookup key is the UUID (IdV2).
		imageLookup := make(map[string]string)
		for _, id := range ids {
			deployment, exists, err := deployments.GetDeployment(r.Context(), id)
			if err != nil {
				httputil.WriteError(w, err)
				return
			}
			if !exists {
				continue
			}
			for _, c := range deployment.GetContainers() {
				digest := c.GetImage().GetId()
				if digest == "" {
					continue
				}
				if _, seen := imageLookup[digest]; !seen {
					lookupKey := digest
					if v2Enabled {
						lookupKey = c.GetImage().GetIdV2()
						if lookupKey == "" {
							continue
						}
					}
					imageLookup[digest] = lookupKey
				}

				err := arrayWriter.WriteObject(&splunkDeploymentEvent{
					Type:        "deployment",
					Cluster:     deployment.GetClusterName(),
					Namespace:   deployment.GetNamespace(),
					Labels:      deployment.GetLabels(),
					Annotations: deployment.GetAnnotations(),
					Deployment:  deployment.GetName(),
					ImageDigest: digest,
				})
				if err != nil {
					httputil.WriteError(w, err)
					return
				}
			}
		}

		for digest, lookupKey := range imageLookup {
			var img imageFields
			if v2Enabled {
				v2, exists, err := imagesV2.GetImage(r.Context(), lookupKey)
				if err != nil {
					httputil.WriteError(w, err)
					return
				}
				if !exists {
					continue
				}
				img = v2
			} else {
				v1, exists, err := images.GetImage(r.Context(), lookupKey)
				if err != nil {
					httputil.WriteError(w, err)
					return
				}
				if !exists {
					continue
				}
				img = v1
			}

			if err := writeImageEvents(arrayWriter, digest, img); err != nil {
				httputil.WriteError(w, err)
				return
			}
		}

		if err := arrayWriter.Finish(); err != nil {
			httputil.WriteError(w, err)
		}
	}
}

func writeImageEvents(w *jsonutil.JSONArrayWriter, digest string, img imageFields) error {
	err := w.WriteObject(&splunkImageEvent{
		Type:          "image",
		ImageDigest:   digest,
		OS:            img.GetScan().GetOperatingSystem(),
		CreatedTime:   protoconv.ConvertTimestampToTimeOrNow(img.GetMetadata().GetV1().GetCreated()),
		ImageRegistry: img.GetName().GetRegistry(),
		ImageRepo:     img.GetName().GetRemote(),
		ImageTag:      img.GetName().GetTag(),
	})
	if err != nil {
		return err
	}

	for _, c := range img.GetScan().GetComponents() {
		for _, v := range c.GetVulns() {
			err = w.WriteObject(&splunkCVEEvent{
				Type:        "cve",
				ImageDigest: digest,
				Component:   c.GetName(),
				Version:     c.GetVersion(),
				CVE:         v.GetCve(),
				CVSS:        v.GetCvss(),
				FixedBy:     v.GetFixedBy(),
				FirstSeen:   protoconv.ConvertTimestampToTimeOrNow(v.GetFirstImageOccurrence()),
				Source:      c.GetSource().String(),
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}
