package sensor

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/listeners"
	"bitbucket.org/stack-rox/apollo/pkg/sources"
)

// EventWrapToEvent returns a function that, when given a DeploymentEventWrap,  updates it's images with
// any meta data or scan data we can find for it in the registry.
func eventWrapToEvent(imageIntegrationPoller *sources.Client) func(ev *listeners.DeploymentEventWrap) (*v1.DeploymentEvent, error) {
	return func(ev *listeners.DeploymentEventWrap) (*v1.DeploymentEvent, error) {
		// TODO(cgorman) can reuse code from central to implement this
		for _, c := range ev.GetDeployment().GetContainers() {
			img := c.GetImage()
			for _, integration := range imageIntegrationPoller.Integrations() {
				registry := integration.Registry
				if registry != nil && registry.Match(img) {
					meta, err := registry.Metadata(img)
					if err != nil {
						log.Warnf("Couldn't get metadata for %v: %s", img, err)
					}
					img.Metadata = meta
				}
			}
			for _, integration := range imageIntegrationPoller.Integrations() {
				scanner := integration.Scanner
				if scanner != nil && scanner.Match(img) {
					scan, err := scanner.GetLastScan(img)
					if err != nil {
						log.Warnf("Couldn't get metadata for %v: %s", img, err)
					}
					img.Scan = scan
				}
			}
		}

		return ev.DeploymentEvent, nil
	}
}
