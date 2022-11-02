package client

import (
	"bytes"
	"fmt"

	sdkhandler "github.com/operator-framework/operator-lib/handler"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/postrender"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/operator-framework/helm-operator-plugins/internal/sdk/controllerutil"
	"github.com/operator-framework/helm-operator-plugins/pkg/manifestutil"
)

// WithInstallPostRenderer sets the post-renderer to use for the install.
// It overrides any post-renderer that may already be configured or set
// as a default.
func WithInstallPostRenderer(pr postrender.PostRenderer) InstallOption {
	return func(i *action.Install) error {
		i.PostRenderer = pr
		return nil
	}
}

// AppendInstallPostRenderer appends a post-renderer to the existing chain
// of post-renderers configured for the install. This function should be used
// instead of WithInstallPostRenderer if you want to inherit the default set
// of post-renderers configured by an ActionClientGetter.
func AppendInstallPostRenderer(pr postrender.PostRenderer) InstallOption {
	return func(a *action.Install) error {
		a.PostRenderer = appendPostRenderer(a.PostRenderer, pr)
		return nil
	}
}

// WithUpgradePostRenderer sets the post-renderer to use for the upgrade.
// It overrides any post-renderer that may already be configured or set
// as a default.
func WithUpgradePostRenderer(pr postrender.PostRenderer) UpgradeOption {
	return func(a *action.Upgrade) error {
		a.PostRenderer = pr
		return nil
	}
}

// AppendUpgradePostRenderer appends a post-renderer to the existing chain
// of post-renderers configured for the upgrade. This function should be used
// instead of WithUpgradePostRenderer if you want to inherit the default set
// of post-renderers configured by an ActionClientGetter.
func AppendUpgradePostRenderer(pr postrender.PostRenderer) UpgradeOption {
	return func(a *action.Upgrade) error {
		a.PostRenderer = appendPostRenderer(a.PostRenderer, pr)
		return nil
	}
}

func appendPostRenderer(pr postrender.PostRenderer, extra postrender.PostRenderer) postrender.PostRenderer {
	if pr == nil {
		return extra
	}
	if cpr, ok := pr.(chainedPostRenderer); ok {
		cpr = append(cpr, extra)
		return cpr
	}
	return chainedPostRenderer{pr, extra}
}

// PostRendererFunc defines a function signature that implements helm's PostRenderer interface.
type PostRendererFunc func(buffer *bytes.Buffer) (*bytes.Buffer, error)

// Run runs the post-renderer function.
func (f PostRendererFunc) Run(buffer *bytes.Buffer) (*bytes.Buffer, error) {
	return f(buffer)
}

// DefaultPostRendererFunc returns a post-renderer that applies owner references to compatible objects
// in a helm release manifest. This is the default post-renderer used by ActionClients created with
// NewActionClientGetter.
var DefaultPostRendererFunc = func(rm meta.RESTMapper, kubeClient kube.Interface, owner client.Object) postrender.PostRenderer {
	return &ownerPostRenderer{rm, kubeClient, owner}
}

type chainedPostRenderer []postrender.PostRenderer

func (prs chainedPostRenderer) Run(in *bytes.Buffer) (*bytes.Buffer, error) {
	var (
		out = &bytes.Buffer{}
		err error
	)
	out.Write(in.Bytes())
	for i, pr := range prs {
		out, err = pr.Run(out)
		if err != nil {
			return nil, fmt.Errorf("postrenderer[%d] (%T) failed: %v", i, pr, err)
		}
	}
	return out, nil
}

type ownerPostRenderer struct {
	rm         meta.RESTMapper
	kubeClient kube.Interface
	owner      client.Object
}

func (pr *ownerPostRenderer) Run(in *bytes.Buffer) (*bytes.Buffer, error) {
	resourceList, err := pr.kubeClient.Build(in, false)
	if err != nil {
		return nil, err
	}
	out := bytes.Buffer{}

	err = resourceList.Visit(func(r *resource.Info, err error) error {
		if err != nil {
			return err
		}
		objMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(r.Object)
		if err != nil {
			return err
		}
		u := &unstructured.Unstructured{Object: objMap}
		useOwnerRef, err := controllerutil.SupportsOwnerReference(pr.rm, pr.owner, u)
		if err != nil {
			return err
		}
		if useOwnerRef && !manifestutil.HasResourcePolicyKeep(u.GetAnnotations()) {
			ownerRef := metav1.NewControllerRef(pr.owner, pr.owner.GetObjectKind().GroupVersionKind())
			ownerRefs := append(u.GetOwnerReferences(), *ownerRef)
			u.SetOwnerReferences(ownerRefs)
		} else {
			if err := sdkhandler.SetOwnerAnnotations(pr.owner, u); err != nil {
				return err
			}
		}
		outData, err := yaml.Marshal(u.Object)
		if err != nil {
			return err
		}
		if _, err := out.WriteString("---\n" + string(outData)); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}
