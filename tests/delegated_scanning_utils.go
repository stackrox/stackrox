//go:build test_e2e

package tests

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io"
	"os"
	"testing"
	"time"

	buildv1 "github.com/openshift/api/build/v1"
	buildv1client "github.com/openshift/client-go/build/clientset/versioned/typed/build/v1"
	imagev1client "github.com/openshift/client-go/image/clientset/versioned/typed/image/v1"
	pkgTar "github.com/stackrox/rox/pkg/tar"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	k8sTypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
)

// Contains utility functions to aid the execution of the delegated scanning tests.
// These may be moved to a more common util in the future when/if additional tests
// could benefit. The current implementation is specific to the needs of delegated
// scanning.

// deleScanOCPTestImageBuilder contains helper functions for adding images to the
// OCP internal registry.
type deleScanOCPTestImageBuilder struct {
	t         *testing.T
	ctx       context.Context
	namespace string
	restCfg   *rest.Config
}

// BuildOCPInternalImage builds an image, pushes it to the OCP internal registry, and
// returns the image reference.  Name is the unique name that will be used for the
// associated build configs, image streams, and final image.  From image is used
// in the 'FROM' instruction when building the new image.
func (d *deleScanOCPTestImageBuilder) BuildOCPInternalImage(name, fromImage string) string {
	d.applyBuildConfig(name)
	d.applyImageStream(name)
	return d.buildAndPushImage(name, fromImage)
}

func (d *deleScanOCPTestImageBuilder) applyBuildConfig(name string) {
	t := d.t
	restCfg := d.restCfg
	ctx := d.ctx

	t.Logf("Applying buildconfig %q", name)
	buildV1Client, err := buildv1client.NewForConfig(restCfg)
	require.NoError(t, err)
	d.applyK8sYaml(ctx, buildV1Client.RESTClient(), "buildconfigs", name, "testdata/delegatedscanning/build-config.yaml.tmpl")
}

func (d *deleScanOCPTestImageBuilder) applyImageStream(name string) {
	t := d.t
	restCfg := d.restCfg
	ctx := d.ctx

	t.Logf("Applying imagestream %q", name)
	imageV1Client, err := imagev1client.NewForConfig(restCfg)
	require.NoError(t, err)
	d.applyK8sYaml(ctx, imageV1Client.RESTClient(), "imagestreams", name, "testdata/delegatedscanning/image-stream.yaml.tmpl")
}

func (d *deleScanOCPTestImageBuilder) buildAndPushImage(name, fromImage string) string {
	t := d.t
	ns := d.namespace
	ctx := d.ctx

	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	file := "testdata/delegatedscanning/Dockerfile.tmpl"
	dockerFileB := d.renderTemplate(file, map[string]string{"image": fromImage})

	err := tw.WriteHeader(&tar.Header{
		Name:    "Dockerfile",
		ModTime: time.Now(),
		Mode:    0644,
		Size:    int64(len(dockerFileB)),
	})
	require.NoError(t, err)
	_, err = tw.Write(dockerFileB)
	require.NoError(t, err)

	dir := "testdata/delegatedscanning/image-build"
	require.NoError(t, pkgTar.FromPath(dir, tw))
	utils.IgnoreError(tw.Close)

	buildV1Client, err := buildv1client.NewForConfig(d.restCfg)
	require.NoError(t, err)
	client := buildV1Client.RESTClient()

	result := &buildv1.Build{}
	t.Logf("Starting build for %q", name)
	err = client.Post().
		Namespace(ns).
		Resource("buildconfigs").
		Name(name).
		SubResource("instantiatebinary").
		Body(buf).
		Do(ctx).
		Into(result)

	t.Logf("Streaming build logs for %q", result.GetName())
	err = d.streamBuildLogs(client, result, ns)
	require.NoError(t, err)

	t.Logf("Waiting for build %q to complete", result.GetName())
	err = d.waitForBuildComplete(buildV1Client.Builds(ns), result.GetName())
	require.NoError(t, err)

	return result.Status.OutputDockerImageReference
}

func (d *deleScanOCPTestImageBuilder) applyK8sYaml(ctx context.Context, client rest.Interface, resource, name, file string) {
	t := d.t
	ns := d.namespace

	yamlB := d.renderTemplate(file, map[string]string{"name": name, "namespace": ns})

	force := true
	options := metav1.PatchOptions{Force: &force, FieldManager: "Ignore"}
	r := client.Patch(k8sTypes.ApplyPatchType).
		Namespace(ns).
		Resource(resource).
		Name(name).
		Body(yamlB).
		VersionedParams(&options, metav1.ParameterCodec).
		Do(ctx)
	require.NoError(t, r.Error())
}

func (d *deleScanOCPTestImageBuilder) renderTemplate(templateFile string, data map[string]string) []byte {
	t := d.t

	tmpl, err := template.ParseFiles(templateFile)
	require.NoError(t, err)
	fileB := new(bytes.Buffer)
	err = tmpl.Execute(fileB, data)
	require.NoError(t, err)

	return fileB.Bytes()
}

// streamBuildLogs was inspired by https://github.com/openshift/oc/blob/67813c212f6625919fa42524a27c399be653a51f/pkg/cli/startbuild/startbuild.go#L507,
// it mimics the log following behavior of `oc start-build --follow=true`.
func (d *deleScanOCPTestImageBuilder) streamBuildLogs(buildRestClient rest.Interface, build *buildv1.Build, namespace string) error {
	ctx := d.ctx

	opts := buildv1.BuildLogOptions{
		Follow: true,
		NoWait: false,
	}
	scheme := runtime.NewScheme()
	err := buildv1.AddToScheme(scheme)
	if err != nil {
		return fmt.Errorf("adding build objects to scheme: %w", err)
	}
	for {
		rd, err := buildRestClient.Get().
			Namespace(namespace).
			Resource("builds").
			Name(build.GetName()).
			SubResource("log").
			VersionedParams(&opts, runtime.NewParameterCodec(scheme)).Stream(ctx)
		if err != nil {
			fmt.Printf("unable to stream the build logs: %v\n", err)
			time.Sleep(1 * time.Second)
			continue
		}
		defer utils.IgnoreError(rd.Close)

		if _, err := io.Copy(os.Stdout, rd); err != nil {
			fmt.Printf("unable to stream the build logs: %v\n", err)
		}
		break
	}
	return err
}

// waitForBuildComplete was inspired by https://github.com/openshift/oc/blob/67813c212f6625919fa42524a27c399be653a51f/pkg/cli/startbuild/startbuild.go#L1067,
// it mimics `oc start-build --wait` by waiting for a build to complete allowing for synchronous build execution.
func (d *deleScanOCPTestImageBuilder) waitForBuildComplete(c buildv1client.BuildInterface, name string) error {
	ctx := d.ctx

	isOK := func(b *buildv1.Build) bool {
		return b.Status.Phase == buildv1.BuildPhaseComplete
	}
	isFailed := func(b *buildv1.Build) bool {
		return b.Status.Phase == buildv1.BuildPhaseFailed ||
			b.Status.Phase == buildv1.BuildPhaseCancelled ||
			b.Status.Phase == buildv1.BuildPhaseError
	}

	for {
		list, err := c.List(ctx, metav1.ListOptions{FieldSelector: fields.Set{"metadata.name": name}.AsSelector().String()})
		if err != nil {
			return err
		}
		for i := range list.Items {
			if name == list.Items[i].Name && isOK(&list.Items[i]) {
				return nil
			}
			if name != list.Items[i].Name || isFailed(&list.Items[i]) {
				return fmt.Errorf("the build %s/%s status is %q", list.Items[i].Namespace, list.Items[i].Name, list.Items[i].Status.Phase)
			}
		}

		rv := list.ResourceVersion
		w, err := c.Watch(ctx, metav1.ListOptions{FieldSelector: fields.Set{"metadata.name": name}.AsSelector().String(), ResourceVersion: rv})
		if err != nil {
			return err
		}
		defer w.Stop()

		for {
			val, ok := <-w.ResultChan()
			if !ok {
				break
			}
			if e, ok := val.Object.(*buildv1.Build); ok {
				if name == e.Name && isOK(e) {
					return nil
				}
				if name != e.Name || isFailed(e) {
					return fmt.Errorf("the build %s/%s status is %q", e.Namespace, name, e.Status.Phase)
				}
			}
		}
	}
}
