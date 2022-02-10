package util

import (
	"bufio"
	"bytes"
	"io"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/stringutils"
	"helm.sh/helm/v3/pkg/chart"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	kindRegex = regexp.MustCompile(`^[A-Z][a-zA-z0-9]+$`)
)

// ExtractApproximateGVKsFromChart aims to extract the approximate set of GroupVersionKinds contained in the template
// manifests, regardless of chart configuration.
// This function only returns an approximate set. Based on the usage of templates in the code, it can return both an
// over-approximation (returning GVKs that can never occur in the rendering output) as well as an under-approximation
// (not returning GVKs that can occur in the rendering output).
func ExtractApproximateGVKsFromChart(ch *chart.Chart) ([]schema.GroupVersionKind, error) {
	allGVKs := make(map[schema.GroupVersionKind]struct{})
	for _, tpl := range ch.Templates {
		lname := strings.ToLower(tpl.Name)
		if !strings.HasSuffix(lname, ".yaml") && !strings.HasSuffix(lname, ".yml") {
			continue
		}
		if err := extractGVKsFromTemplate(tpl.Data, allGVKs); err != nil {
			return nil, err
		}
	}
	gvksSorted := make([]schema.GroupVersionKind, 0, len(allGVKs))
	for gvk := range allGVKs {
		gvksSorted = append(gvksSorted, gvk)
	}
	sort.Slice(gvksSorted, func(i, j int) bool {
		return gvksSorted[i].String() < gvksSorted[j].String()
	})
	return gvksSorted, nil
}

func extractGVKsFromTemplate(tplContents []byte, gvkSetOut map[schema.GroupVersionKind]struct{}) error {
	// Add a newline + document separator at end of input, to avoid replicating the end-of-document logic in the below
	// loop.
	scanner := bufio.NewScanner(io.MultiReader(bytes.NewReader(tplContents), strings.NewReader("\n---")))

	gvs := make(map[schema.GroupVersion]struct{})
	kinds := set.NewStringSet()
	for scanner.Scan() {
		line := strings.TrimRightFunc(stringutils.GetUpTo(scanner.Text(), "#"), unicode.IsSpace) // remove comments and trim spaces
		if strings.Contains(line, "{{") {                                                        // ignore lines with templating
			continue
		}
		if line == "---" { // document separator
			for gv := range gvs {
				for kind := range kinds {
					gvk := schema.GroupVersionKind{Group: gv.Group, Version: gv.Version, Kind: kind}
					gvkSetOut[gvk] = struct{}{}
				}
			}
			gvs = make(map[schema.GroupVersion]struct{})
			kinds = set.NewStringSet()
			continue
		}

		if stringutils.ConsumePrefix(&line, "apiVersion:") {
			if gv, err := schema.ParseGroupVersion(strings.TrimSpace(line)); err == nil {
				gvs[gv] = struct{}{}
			}
		} else if stringutils.ConsumePrefix(&line, "kind:") {
			kind := strings.TrimSpace(line)
			if kindRegex.MatchString(kind) {
				kinds.Add(kind)
			}
		}
	}
	return scanner.Err()
}
