package splunk

import (
	"net/http"
	"time"

	"github.com/stackrox/rox/central/deployment/datastore"
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/jsonutil"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/set"
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

// NewVulnMgmtHandler returns an http.HandlerFunc implementation that returns all the required events for the Splunk TA
func NewVulnMgmtHandler(deployments datastore.DataStore, images imageDatastore.DataStore) http.HandlerFunc {
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

		imageSet := set.NewStringSet()
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
				if c.GetImage().GetId() == "" {
					continue
				}
				imageSet.Add(c.GetImage().GetId())

				err := arrayWriter.WriteObject(&splunkDeploymentEvent{
					Type:        "deployment",
					Cluster:     deployment.GetClusterName(),
					Namespace:   deployment.GetNamespace(),
					Labels:      deployment.GetLabels(),
					Annotations: deployment.GetAnnotations(),
					Deployment:  deployment.GetName(),
					ImageDigest: c.GetImage().GetId(),
				})
				if err != nil {
					httputil.WriteError(w, err)
					return
				}
			}
		}
		for id := range imageSet {
			image, exists, err := images.GetImage(r.Context(), id)
			if err != nil {
				httputil.WriteError(w, err)
				return
			}
			if !exists {
				continue
			}

			err = arrayWriter.WriteObject(&splunkImageEvent{
				Type:          "image",
				ImageDigest:   id,
				OS:            image.GetScan().GetOperatingSystem(),
				CreatedTime:   protoconv.ConvertTimestampToTimeOrNow(image.GetMetadata().GetV1().GetCreated()),
				ImageRegistry: image.GetName().GetRegistry(),
				ImageRepo:     image.GetName().GetRemote(),
				ImageTag:      image.GetName().GetTag(),
			})
			if err != nil {
				httputil.WriteError(w, err)
				return
			}

			for _, c := range image.GetScan().GetComponents() {
				for _, v := range c.GetVulns() {
					err = arrayWriter.WriteObject(&splunkCVEEvent{
						Type:        "cve",
						ImageDigest: id,
						Component:   c.GetName(),
						Version:     c.GetVersion(),
						CVE:         v.GetCve(),
						CVSS:        v.GetCvss(),
						FixedBy:     v.GetFixedBy(),
						FirstSeen:   protoconv.ConvertTimestampToTimeOrNow(v.GetFirstImageOccurrence()),
						Source:      c.GetSource().String(),
					})
					if err != nil {
						httputil.WriteError(w, err)
						return
					}
				}
			}
		}
		if err := arrayWriter.Finish(); err != nil {
			httputil.WriteError(w, err)
		}
	}
}
