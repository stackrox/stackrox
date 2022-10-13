package securedclusterservices

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stackrox/rox/pkg/k8sutil/k8sobjects"
	"github.com/stackrox/rox/pkg/maputil"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/strvals"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type customizeSuite struct {
	baseSuite
}

func TestCustomize(t *testing.T) {
	suite.Run(t, new(customizeSuite))
}

func (s *customizeSuite) TestCustomizeMetadata() {
	// Render template with no generated values, and parse the objects.
	_, rendered := s.LoadAndRender(allValuesExplicit)

	objs := s.ParseObjects(rendered)

	// Based on the present objects, build customization settings that are unique
	// per object and also exercise override and suppress logic.
	mdPrefix := uuid.NewV4()
	customizeVals := chartutil.Values{}

	expectedMDs := make(map[string]map[k8sobjects.ObjectRef]map[string]string)

	var setExprs []string
	for _, mdType := range []string{"labels", "annotations"} {
		expectedMDs[mdType] = make(map[k8sobjects.ObjectRef]map[string]string)

		globalKeyNamePrefix := fmt.Sprintf("mdTest%sGlobal", strings.Title(mdType))
		typeKeyNamePrefix := fmt.Sprintf("mdTest%sType", strings.Title(mdType))
		objKeyNamePrefix := fmt.Sprintf("mdTest%sObj", strings.Title(mdType))

		globalCustomizePrefix := mdType
		globalSig := fmt.Sprintf("%s-%s", mdPrefix, mdType)
		setExprs = append(setExprs,
			fmt.Sprintf("%s.%s=%s", globalCustomizePrefix, globalKeyNamePrefix, globalSig),
			fmt.Sprintf("%s.%sToOverrideByObj=INVALID", globalCustomizePrefix, globalKeyNamePrefix),
			fmt.Sprintf("%s.%sToDeleteByObj=INVALID", globalCustomizePrefix, globalKeyNamePrefix),
			fmt.Sprintf("%s.%sToOverrideByType=INVALID", globalCustomizePrefix, globalKeyNamePrefix),
			fmt.Sprintf("%s.%sToDeleteByType=INVALID", globalCustomizePrefix, globalKeyNamePrefix),
		)

		seenKinds := set.NewStringSet()

		for i := range objs {
			obj := objs[i]
			typeSig := fmt.Sprintf("%s-%s", globalSig, obj.GetKind())
			objSig := fmt.Sprintf("%s-%s", typeSig, obj.GetName())

			expectedMD := map[string]string{
				globalKeyNamePrefix: globalSig,
				fmt.Sprintf("%sToOverrideByType", globalKeyNamePrefix): typeSig,
				fmt.Sprintf("%sToOverrideByObj", globalKeyNamePrefix):  objSig,

				typeKeyNamePrefix: typeSig,
				fmt.Sprintf("%sToOverrideByObj", typeKeyNamePrefix): objSig,

				objKeyNamePrefix: objSig,
			}
			expectedMDs[mdType][k8sobjects.RefOf(&obj)] = expectedMD

			objKeyPrefix := fmt.Sprintf("other.%s/%s.%s", strings.ToLower(obj.GetKind()), obj.GetName(), mdType)
			setExprs = append(setExprs,
				fmt.Sprintf("%s.%s=%s", objKeyPrefix, objKeyNamePrefix, objSig),
				fmt.Sprintf("%s.%sToOverrideByObj=%s", objKeyPrefix, globalKeyNamePrefix, objSig),
				fmt.Sprintf("%s.%sToDeleteByObj=null", objKeyPrefix, globalKeyNamePrefix),
				fmt.Sprintf("%s.%sToOverrideByObj=%s", objKeyPrefix, typeKeyNamePrefix, objSig),
				fmt.Sprintf("%s.%sToDeleteByObj=null", objKeyPrefix, typeKeyNamePrefix),
			)

			if seenKinds.Add(obj.GetKind()) {
				typeKeyPrefix := fmt.Sprintf("other.%s/*.%s", strings.ToLower(obj.GetKind()), mdType)
				setExprs = append(setExprs,
					fmt.Sprintf("%s.%s=%s", typeKeyPrefix, typeKeyNamePrefix, typeSig),
					fmt.Sprintf("%s.%sToOverrideByObj=INVALID", typeKeyPrefix, typeKeyNamePrefix),
					fmt.Sprintf("%s.%sToDeleteByObj=INVALID", typeKeyPrefix, typeKeyNamePrefix),
					fmt.Sprintf("%s.%sToOverrideByType=%s", typeKeyPrefix, globalKeyNamePrefix, typeSig),
					fmt.Sprintf("%s.%sToDeleteByType=null", typeKeyPrefix, globalKeyNamePrefix),
				)
			}
		}
	}

	for _, setExpr := range setExprs {
		s.Require().NoError(strvals.ParseInto(fmt.Sprintf("customize.%s", setExpr), customizeVals), "failed to evaluate --set expression %q", setExpr)
	}

	var customizeValsStrBuilder strings.Builder
	s.Require().NoError(customizeVals.Encode(&customizeValsStrBuilder), "failed to encode customize values")

	_, rendered = s.LoadAndRender(allValuesExplicit, customizeValsStrBuilder.String())
	objs = s.ParseObjects(rendered)

	for i := range objs {
		obj := objs[i]
		for _, mdType := range []string{"labels", "annotations"} {
			objRef := k8sobjects.RefOf(&obj)
			expectedMD := maputil.ShallowClone(expectedMDs[mdType][objRef])

			actualMD, found, err := unstructured.NestedStringMap(obj.Object, "metadata", mdType)
			s.Require().NoErrorf(err, "could not retrieve %s metadata for object %v", mdType, objRef)
			s.Require().True(found, "could not retrieve %s metadata for object %v", mdType, objRef)

			for k, v := range actualMD {
				if !strings.HasPrefix(k, "mdTest") {
					continue
				}
				expectedV, ok := expectedMD[k]
				s.Truef(ok, "unexpected custom %s metadata key %s in object %v", mdType, k, objRef)
				s.Equalf(expectedV, v, "unexpected custom %s metadata value for key %s in object %v", mdType, k, objRef)
				delete(expectedMD, k)
			}

			s.Emptyf(expectedMD, "expected metadata keys for object %v not found", objRef)
		}
	}
}

func (s *customizeSuite) TestCustomizePodMetadata() {
	// Render template with no generated values, and parse the objects.
	_, rendered := s.LoadAndRender(allValuesExplicit)

	objs := s.ParseObjects(rendered)

	// Based on the present objects, build customization settings that are unique
	// per object and also exercise override and suppress logic.
	mdPrefix := uuid.NewV4()
	customizeVals := chartutil.Values{}

	expectedMDs := make(map[string]map[k8sobjects.ObjectRef]map[string]string)

	var setExprs []string
	for _, mdType := range []string{"labels", "annotations"} {
		expectedMDs[mdType] = make(map[k8sobjects.ObjectRef]map[string]string)

		globalKeyNamePrefix := fmt.Sprintf("mdTest%sGlobal", strings.Title(mdType))
		typeKeyNamePrefix := fmt.Sprintf("mdTest%sType", strings.Title(mdType))
		objKeyNamePrefix := fmt.Sprintf("mdTest%sObj", strings.Title(mdType))

		globalCustomizePrefix := fmt.Sprintf("pod%s", strings.Title(mdType))
		globalSig := fmt.Sprintf("%s-%s", mdPrefix, mdType)
		setExprs = append(setExprs,
			fmt.Sprintf("%s.%s=%s", globalCustomizePrefix, globalKeyNamePrefix, globalSig),
			fmt.Sprintf("%s.%sToOverrideByObj=INVALID", globalCustomizePrefix, globalKeyNamePrefix),
			fmt.Sprintf("%s.%sToDeleteByObj=INVALID", globalCustomizePrefix, globalKeyNamePrefix),
			fmt.Sprintf("%s.%sToOverrideByType=INVALID", globalCustomizePrefix, globalKeyNamePrefix),
			fmt.Sprintf("%s.%sToDeleteByType=INVALID", globalCustomizePrefix, globalKeyNamePrefix),
		)

		seenKinds := set.NewStringSet()

		for i := range objs {
			obj := objs[i]
			if obj.GetKind() != "Deployment" && obj.GetKind() != "DaemonSet" {
				continue
			}
			typeSig := fmt.Sprintf("%s-%s", globalSig, obj.GetKind())
			objSig := fmt.Sprintf("%s-%s", typeSig, obj.GetName())

			expectedMD := map[string]string{
				globalKeyNamePrefix: globalSig,
				fmt.Sprintf("%sToOverrideByType", globalKeyNamePrefix): typeSig,
				fmt.Sprintf("%sToOverrideByObj", globalKeyNamePrefix):  objSig,

				typeKeyNamePrefix: typeSig,
				fmt.Sprintf("%sToOverrideByObj", typeKeyNamePrefix): objSig,

				objKeyNamePrefix: objSig,
			}
			expectedMDs[mdType][k8sobjects.RefOf(&obj)] = expectedMD

			objKeyPrefix := fmt.Sprintf("other.%s/%s.pod%s", strings.ToLower(obj.GetKind()), obj.GetName(), strings.Title(mdType))
			setExprs = append(setExprs,
				fmt.Sprintf("%s.%s=%s", objKeyPrefix, objKeyNamePrefix, objSig),
				fmt.Sprintf("%s.%sToOverrideByObj=%s", objKeyPrefix, globalKeyNamePrefix, objSig),
				fmt.Sprintf("%s.%sToDeleteByObj=null", objKeyPrefix, globalKeyNamePrefix),
				fmt.Sprintf("%s.%sToOverrideByObj=%s", objKeyPrefix, typeKeyNamePrefix, objSig),
				fmt.Sprintf("%s.%sToDeleteByObj=null", objKeyPrefix, typeKeyNamePrefix),
			)

			if seenKinds.Add(obj.GetKind()) {
				typeKeyPrefix := fmt.Sprintf("other.%s/*.pod%s", strings.ToLower(obj.GetKind()), strings.Title(mdType))
				setExprs = append(setExprs,
					fmt.Sprintf("%s.%s=%s", typeKeyPrefix, typeKeyNamePrefix, typeSig),
					fmt.Sprintf("%s.%sToOverrideByObj=INVALID", typeKeyPrefix, typeKeyNamePrefix),
					fmt.Sprintf("%s.%sToDeleteByObj=INVALID", typeKeyPrefix, typeKeyNamePrefix),
					fmt.Sprintf("%s.%sToOverrideByType=%s", typeKeyPrefix, globalKeyNamePrefix, typeSig),
					fmt.Sprintf("%s.%sToDeleteByType=null", typeKeyPrefix, globalKeyNamePrefix),
				)
			}
		}
	}

	for _, setExpr := range setExprs {
		s.Require().NoError(strvals.ParseInto(fmt.Sprintf("customize.%s", setExpr), customizeVals), "failed to evaluate --set expression %q", setExpr)
	}

	var customizeValsStrBuilder strings.Builder
	s.Require().NoError(customizeVals.Encode(&customizeValsStrBuilder), "failed to encode customize values")

	_, rendered = s.LoadAndRender(allValuesExplicit, customizeValsStrBuilder.String())
	s.Require().NotEmpty(rendered)

	objs = s.ParseObjects(rendered)
	s.Require().NotEmpty(objs)

	for i := range objs {
		obj := objs[i]
		if obj.GetKind() != "Deployment" && obj.GetKind() != "DaemonSet" {
			continue
		}

		for _, mdType := range []string{"labels", "annotations"} {
			objRef := k8sobjects.RefOf(&obj)
			expectedMD := maputil.ShallowClone(expectedMDs[mdType][objRef])

			actualMD, found, err := unstructured.NestedStringMap(obj.Object, "spec", "template", "metadata", mdType)
			s.Require().NoErrorf(err, "could not retrieve %s metadata for object %v", mdType, objRef)
			s.Require().True(found, "could not retrieve %s metadata for object %v", mdType, objRef)

			for k, v := range actualMD {
				if !strings.HasPrefix(k, "mdTest") {
					continue
				}
				expectedV, ok := expectedMD[k]
				s.Truef(ok, "unexpected custom %s metadata key %s in object %v", mdType, k, objRef)
				s.Equalf(expectedV, v, "unexpected custom %s metadata value for key %s in object %v", mdType, k, objRef)
				delete(expectedMD, k)
			}

			s.Emptyf(expectedMD, "expected metadata keys for object %v not found", objRef)
		}
	}
}
