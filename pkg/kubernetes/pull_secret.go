package kubernetes

import (
	"bytes"
	"html/template"

	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/urlfmt"
)

const (
	secretTemplate = `

kubectl get namespace {{.NamespaceVar}} > /dev/null || kubectl create namespace {{.NamespaceVar}}

if ! kubectl get secret/{{.ImagePullSecretVar}} -n {{.NamespaceVar}} > /dev/null; then
  if [ -z "${REGISTRY_USERNAME}" ]; then
    echo -n "Username for {{.RegistryVar}}: "
    read REGISTRY_USERNAME
    echo
  fi
  if [ -z "${REGISTRY_PASSWORD}" ]; then
    echo -n "Password for {{.RegistryVar}}: "
    read -s REGISTRY_PASSWORD
    echo
  fi

  kubectl create secret docker-registry \
    "{{.ImagePullSecretVar}}" --namespace "{{.NamespaceVar}}" \
    --docker-server={{.RegistryVar}} \
    --docker-username="${REGISTRY_USERNAME}" \
    --docker-password="${REGISTRY_PASSWORD}" \
    --docker-email="support@stackrox.com"

	echo
fi
`
)

var scriptTemplate *template.Template

func init() {
	scriptTemplate = template.Must(template.New("kubernetes").Parse(secretTemplate))
}

// GetCreateSecretTemplate returns the generic script to generate the ImagePullSecret from registry auth
func GetCreateSecretTemplate(namespaceVar, registryVar, imagePullSecretVar string) string {
	var fields = map[string]string{
		"NamespaceVar":       namespaceVar,
		"RegistryVar":        registryVar,
		"ImagePullSecretVar": imagePullSecretVar,
	}

	var b []byte
	buf := bytes.NewBuffer(b)
	err := scriptTemplate.Execute(buf, fields)
	if err != nil {
		return ""
	}
	return buf.String()
}

// GetResolvedRegistry returns the registry endpoint from the image definition
func GetResolvedRegistry(image string) (string, error) {
	parsedImage := utils.GenerateImageFromString(image)
	return urlfmt.FormatURL(parsedImage.GetName().GetRegistry(), urlfmt.HTTPS, urlfmt.NoTrailingSlash)
}
