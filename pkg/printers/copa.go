package printers

import (
	"encoding/json"
	"io"
	"strings"

	copa "github.com/project-copacetic/copacetic/pkg/types/v1alpha1"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/gjson"
	"github.com/stackrox/rox/pkg/set"
	"golang.org/x/exp/maps"
)

type CopaPrinter struct {
	jsonPathExpressions map[string]string
}

func NewCopaPrinter(jsonPathExpressions map[string]string) *CopaPrinter {
	return &CopaPrinter{jsonPathExpressions}
}

func (c *CopaPrinter) Print(object any, w io.Writer) error {
	updatePacakges, err := copaUpdateFromJSONObject(object, c.jsonPathExpressions)
	if err != nil {
		return err
	}

	manifest := copa.UpdateManifest{
		APIVersion: copa.APIVersion,
		Metadata: copa.Metadata{
			OS: copa.OS{
				Type:    "",
				Version: "",
			},
			Config: copa.Config{
				Arch: "",
			},
		},
		Updates: updatePacakges,
	}

	marshal, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	_, err = w.Write(marshal)
	if err != nil {
		return err
	}
	return nil
}

const (
	CopaNameJSONPathExpressionKey             = "CopaNameJSONPathExpressionKey"
	CopaInstalledVersionJSONPathExpressionKey = "CopaInstalledVersionJSONPathExpressionKey"
	CopaFixedVersionJSONPathExpressionKey     = "CopaFixedVersionJSONPathExpressionKey"
	CopaVulnerabilityIdJSONPathExpressionKey  = "CopaVulnerabilityIdJSONPathExpressionKey"
)

var copaRequiredKeys = []string{
	CopaNameJSONPathExpressionKey,
	CopaInstalledVersionJSONPathExpressionKey,
	CopaFixedVersionJSONPathExpressionKey,
	CopaVulnerabilityIdJSONPathExpressionKey,
}

func copaUpdateFromJSONObject(jsonObject any, pathExpressions map[string]string) (copa.UpdatePackages, error) {
	pathExpr := set.NewStringSet(maps.Keys(pathExpressions)...)
	for _, copaRequiredKey := range copaRequiredKeys {
		if !pathExpr.Contains(copaRequiredKey) {
			return nil, errox.InvalidArgs.Newf("not all required JSON path expressions given, ensure JSON "+
				"path expression are given for: [%s]", strings.Join(copaRequiredKeys, ","))
		}
	}

	sliceMapper, err := gjson.NewSliceMapper(jsonObject, pathExpressions)
	if err != nil {
		return nil, err
	}
	data := sliceMapper.CreateSlices()
	numberOfValues := len(data[CopaNameJSONPathExpressionKey])

	copaUpdatePacakges := make([]copa.UpdatePackage, 0, numberOfValues)
	for i := 0; i < numberOfValues; i++ {
		entry := copa.UpdatePackage{
			Name:             data[CopaNameJSONPathExpressionKey][i],
			InstalledVersion: data[CopaInstalledVersionJSONPathExpressionKey][i],
			FixedVersion:     data[CopaFixedVersionJSONPathExpressionKey][i],
			VulnerabilityID:  data[CopaVulnerabilityIdJSONPathExpressionKey][i],
		}
		if entry.FixedVersion == "-" {
			continue
		}
		copaUpdatePacakges = append(copaUpdatePacakges, entry)
	}

	return copaUpdatePacakges, nil
}
