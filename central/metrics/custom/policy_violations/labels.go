package policy_violations

import (
	"slices"
	"strconv"
	"strings"

	"github.com/stackrox/rox/central/metrics/custom/tracker"
	"github.com/stackrox/rox/generated/storage"
)

var lazyLabels = []tracker.LazyLabel[*finding]{
	// Alert
	{Label: "Cluster", Getter: func(f *finding) string { return f.GetClusterName() }},
	{Label: "Namespace", Getter: func(f *finding) string { return f.GetNamespace() }},
	{Label: "Resource", Getter: func(f *finding) string { return f.GetResource().GetName() }},
	{Label: "Deployment", Getter: func(f *finding) string { return f.GetDeployment().GetName() }},
	{Label: "IsDeploymentActive", Getter: func(f *finding) string { return strconv.FormatBool(!f.GetDeployment().GetInactive()) }},
	{Label: "IsPlatformComponent", Getter: func(f *finding) string { return strconv.FormatBool(f.GetPlatformComponent()) }},
	{Label: "Policy", Getter: func(f *finding) string { return f.GetPolicy().GetName() }},
	{Label: "Categories", Getter: func(f *finding) string {
		return strings.Join(slices.Sorted(slices.Values(f.GetPolicy().GetCategories())), ",")
	}},
	{Label: "Severity", Getter: func(f *finding) string { return f.GetPolicy().GetSeverity().String() }},
	{Label: "Action", Getter: func(f *finding) string { return f.GetEnforcement().GetAction().String() }},
	{Label: "Message", Getter: func(f *finding) string { return f.GetEnforcement().GetMessage() }},
	{Label: "Stage", Getter: func(f *finding) string { return f.GetLifecycleStage().String() }},
	{Label: "State", Getter: func(f *finding) string { return f.GetState().String() }},
	{Label: "Entity", Getter: func(f *finding) string { return f.GetEntityType().String() }},
	{Label: "EntityName", Getter: getEntityName},

	// Violation
	{Label: "Type", Getter: func(f *finding) string { return f.GetType().String() }},
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
