package helmcharts

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/image"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/grpc/routes"
	pkgCharts "github.com/stackrox/rox/pkg/helm/charts"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/zip"
	"google.golang.org/grpc"
)

var (
	chartNameRegex = regexp.MustCompile(`^/([^/]+)\.zip$`)

	charts = map[string]image.ChartPrefix{
		"secured-cluster-services": image.SecuredClusterServicesChartPrefix,
	}
)

type service struct{}

func (*service) RegisterServiceServer(*grpc.Server) {}
func (*service) RegisterServiceHandler(context.Context, *runtime.ServeMux, *grpc.ClientConn) error {
	return nil
}

func (s *service) CustomRoutes() []routes.CustomRoute {
	return []routes.CustomRoute{
		{
			Route:         "/api/extensions/helm-charts/",
			Authorizer:    user.Authenticated(),
			ServerHandler: http.StripPrefix("/api/extensions/helm-charts", http.HandlerFunc(s.serveChart)),
			Compression:   false,
		},
	}
}

func (s *service) serveChart(w http.ResponseWriter, req *http.Request) {
	helmImage := image.GetDefaultImage()
	if req.Method != http.MethodGet {
		http.Error(w, fmt.Sprintf("method %q not allowed", req.Method), http.StatusMethodNotAllowed)
		return
	}

	m := chartNameRegex.FindStringSubmatch(req.URL.Path)
	if len(m) != 2 {
		http.Error(w, fmt.Sprintf("unknown path %q", req.URL.Path), http.StatusNotFound)
		return
	}

	chartName := m[1]
	chartPathPrefix := charts[chartName]
	if chartPathPrefix == "" {
		http.Error(w, fmt.Sprintf("unknown chart %q", chartName), http.StatusNotFound)
		return
	}

	flavor := defaults.GetImageFlavorFromEnv()

	// Render template files.
	renderedChartFiles, err := helmImage.LoadAndInstantiateChartTemplate(chartPathPrefix, pkgCharts.GetMetaValuesForFlavor(flavor))
	if err != nil {
		http.Error(w, errors.Wrapf(err, "loading and instantiating %s helmtpl", chartName).Error(), http.StatusInternalServerError)
		return
	}

	wrapper := zip.NewWrapper()
	for _, f := range renderedChartFiles {
		wrapper.AddFiles(zip.NewFile(f.Name, f.Data, 0))
	}

	zipBytes, err := wrapper.Zip()
	if err != nil {
		http.Error(w, errors.Wrap(err, "getting ZIP data").Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, req.URL.Path[1:]))
	w.Header().Set("Content-Length", strconv.Itoa(len(zipBytes)))

	_, _ = w.Write(zipBytes)
}
