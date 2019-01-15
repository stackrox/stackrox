package admissioncontroller

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/util/json"
)

const deploymentReview = `
{
   "kind":"AdmissionReview",
   "apiVersion":"admission.k8s.io/v1beta1",
   "request":{
      "uid":"67add3d2-0249-11e9-9fa7-42010a8a01aa",
      "kind":{
         "group":"extensions",
         "version":"v1beta1",
         "kind":"Deployment"
      },
      "resource":{
         "group":"extensions",
         "version":"v1beta1",
         "resource":"deployments"
      },
      "namespace":"default",
      "operation":"CREATE",
      "userInfo":{
         "username":"cgorman@stackrox.com",
         "groups":[
            "system:authenticated"
         ],
         "extra":{
            "user-assertion.cloud.google.com":[
               "AM6SrXi9jx9RjUGUb/+4ASdGbS5PNFqDUQLiWFTs9EHC7gKjfdTK8ZcM5CUZZU+f8400IV//WMBdp7l9Ww5Iu82aHNHmjU7TyXVgcM9Ij6PzVTeVkjEarrGf2cP0mq0N0UcXcCNodATw35g1Pj6s90pbUD7+26tBU+3ImrblsmMohx72s4Bn5blXE+/mt+zeSdd7fZgKqc3Z5ZERLC6zIQh2mCrJ7mGWSNtXfWdG"
            ]
         }
      },
      "object":{
         "metadata":{
            "name":"nginx4",
            "namespace":"default",
            "uid":"67adcc10-0249-11e9-9fa7-42010a8a01aa",
            "generation":1,
            "creationTimestamp":"2018-12-17T22:16:30Z",
            "labels":{
               "run":"nginx4"
            }
         },
         "spec":{
            "replicas":1,
            "selector":{
               "matchLabels":{
                  "run":"nginx4"
               }
            },
            "template":{
               "metadata":{
                  "creationTimestamp":null,
                  "labels":{
                     "run":"nginx4"
                  }
               },
               "spec":{
                  "containers":[
                     {
                        "name":"nginx4",
                        "image":"nginx",
                        "resources":{

                        },
                        "terminationMessagePath":"/dev/termination-log",
                        "terminationMessagePolicy":"File",
                        "imagePullPolicy":"Always"
                     }
                  ],
                  "restartPolicy":"Always",
                  "terminationGracePeriodSeconds":30,
                  "dnsPolicy":"ClusterFirst",
                  "securityContext":{

                  },
                  "schedulerName":"default-scheduler"
               }
            },
            "strategy":{
               "type":"RollingUpdate",
               "rollingUpdate":{
                  "maxUnavailable":1,
                  "maxSurge":1
               }
            }
         },
         "status":{

         }
      },
      "oldObject":null
   }
}
`

func TestDeploymentReview(t *testing.T) {
	var admissionReview v1beta1.AdmissionReview
	err := json.Unmarshal([]byte(deploymentReview), &admissionReview)
	require.NoError(t, err)

	d, err := parseIntoDeployment(&admissionReview)
	require.NoError(t, err)
	require.NotNil(t, d)
}
