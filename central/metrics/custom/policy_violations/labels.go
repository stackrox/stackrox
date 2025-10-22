package policy_violations

import (
	"slices"
	"strconv"
	"strings"

	"github.com/stackrox/rox/central/metrics/custom/tracker"
	"github.com/stackrox/rox/generated/storage"
)

var lazyLabels = tracker.LazyLabelGetters[*finding]{
	// Alert
	"Cluster":             func(f *finding) string { return f.GetClusterName() },
	"Namespace":           func(f *finding) string { return f.GetNamespace() },
	"Resource":            func(f *finding) string { return f.GetResource().GetName() },
	"Deployment":          func(f *finding) string { return f.GetDeployment().GetName() },
	"IsDeploymentActive":  func(f *finding) string { return strconv.FormatBool(!f.GetDeployment().GetInactive()) },
	"IsPlatformComponent": func(f *finding) string { return strconv.FormatBool(f.GetPlatformComponent()) },
	"Policy":              func(f *finding) string { return f.GetPolicy().GetName() },
	"Categories": func(f *finding) string {
		return strings.Join(slices.Sorted(slices.Values(f.GetPolicy().GetCategories())), ",")
	},
	"Severity":   func(f *finding) string { return f.GetPolicy().GetSeverity().String() },
	"Action":     func(f *finding) string { return f.GetEnforcement().GetAction().String() },
	"Message":    func(f *finding) string { return f.GetEnforcement().GetMessage() },
	"Stage":      func(f *finding) string { return f.GetLifecycleStage().String() },
	"State":      func(f *finding) string { return f.GetState().String() },
	"Entity":     func(f *finding) string { return f.GetEntityType().String() },
	"EntityName": getEntityName,

	// Violation
	"Type": func(f *finding) string { return f.GetType().String() },
}

type finding struct {
	*storage.Alert
	*storage.Alert_Violation
}

func getEntityName(f *finding) string {
	switch e := f.GetEntity().(type) {
	case *storage.Alert_Deployment_:
		return e.Deployment.GetName()
	case *storage.Alert_Image:
		return e.Image.GetName().GetFullName()
	case *storage.Alert_Resource_:
		return e.Resource.GetName()
	}
	return ""
}
