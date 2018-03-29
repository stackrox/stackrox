package search

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/images"
)

func getProtoMatchesMap(m map[string][]string) map[string]*v1.SearchResult_Matches {
	matches := make(map[string]*v1.SearchResult_Matches)
	for k, v := range m {
		matches[k] = &v1.SearchResult_Matches{Values: v}
	}
	return matches
}

// ConvertAlert returns proto search result from an alert object and the internal search result
func ConvertAlert(alert *v1.Alert, result Result) *v1.SearchResult {
	return &v1.SearchResult{
		Category:       v1.SearchCategory_ALERTS,
		Id:             alert.GetId(),
		Name:           alert.GetPolicy().GetName(),
		FieldToMatches: getProtoMatchesMap(result.Matches),
		Score:          result.Score,
	}
}

// ConvertDeployment returns proto search result from a deployment object and the internal search result
func ConvertDeployment(deployment *v1.Deployment, result Result) *v1.SearchResult {
	return &v1.SearchResult{
		Category:       v1.SearchCategory_DEPLOYMENTS,
		Id:             deployment.GetId(),
		Name:           deployment.GetName(),
		FieldToMatches: getProtoMatchesMap(result.Matches),
		Score:          result.Score,
	}
}

// ConvertPolicy returns proto search result from a policy object and the internal search result
func ConvertPolicy(policy *v1.Policy, result Result) *v1.SearchResult {
	return &v1.SearchResult{
		Category:       v1.SearchCategory_POLICIES,
		Id:             policy.GetId(),
		Name:           policy.GetName(),
		FieldToMatches: getProtoMatchesMap(result.Matches),
		Score:          result.Score,
	}
}

// ConvertImage returns proto search result from a image object and the internal search result
func ConvertImage(image *v1.Image, result Result) *v1.SearchResult {
	return &v1.SearchResult{
		Category:       v1.SearchCategory_IMAGES,
		Id:             images.NewDigest(image.GetName().GetSha()).Digest(),
		Name:           images.Wrapper{Image: image}.String(),
		FieldToMatches: getProtoMatchesMap(result.Matches),
		Score:          result.Score,
	}
}
