//go:build test_e2e

package tests

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"os"
	"strings"
	"testing"
	"text/tabwriter"
	"time"

	buildv1 "github.com/openshift/api/build/v1"
	configv1 "github.com/openshift/api/config/v1"
	machineconfigurationv1 "github.com/openshift/api/machineconfiguration/v1"
	operatorv1alpha1 "github.com/openshift/api/operator/v1alpha1"
	buildv1client "github.com/openshift/client-go/build/clientset/versioned/typed/build/v1"
	configv1client "github.com/openshift/client-go/config/clientset/versioned/typed/config/v1"
	imagev1client "github.com/openshift/client-go/image/clientset/versioned/typed/image/v1"
	machineconfigurationv1client "github.com/openshift/client-go/machineconfiguration/clientset/versioned/typed/machineconfiguration/v1"
	operatorv1alpha1client "github.com/openshift/client-go/operator/clientset/versioned/typed/operator/v1alpha1"
	"github.com/stackrox/rox/pkg/docker/config"
	pkgTar "github.com/stackrox/rox/pkg/tar"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	k8sTypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/rest"
	//TODO: Cleanup these imports so using camel case
)

// This file contains utility functions to aid the execution of the delegated scanning tests.
// The current implementation is specific to the needs of delegated scanning.

const (
	// ignoreFieldManager is the name of the field manager to pass the k8s API.
	// The k8s API uses the name of a field manager to detect conflicts when
	// making changes via the server side apply mechanism.
	ignoreFieldManager = "Ignore"
)

// deleScanTestUtils contains helper functions to assist delegated scanning tests.
// These functions were added to the struct to avoid naming conflicts amongst all
// the other e2e tests.
type deleScanTestUtils struct {
	restCfg *rest.Config
}

// SetupMirrors will add creds to the Global Pull Secret, and create the mirroring CRs
// (ImageContentSourcePolicy, ImageDigestMirrorSet, and ImageTagMirrorSet). If changes
// were made will wait for changes to be propagated to all nodes.
// During testing on a 3 master + 2 worker node OCP cluster the average time it took
// to propagate these changes was between 5-10 minutes.
func (d *deleScanTestUtils) SetupMirrors(t *testing.T, ctx context.Context, reg string, dce config.DockerConfigEntry) {
	start := time.Now()
	defer func() {
		t.Logf("Setting up mirrors took:: %v", time.Since(start))
	}()

	// Instantiate various API clients.
	opClient, err := operatorv1alpha1client.NewForConfig(d.restCfg)
	require.NoError(t, err)

	cfgClient, err := configv1client.NewForConfig(d.restCfg)
	require.NoError(t, err)

	machineCfgClient, err := machineconfigurationv1client.NewForConfig(d.restCfg)
	require.NoError(t, err)

	// Get current status of current machine config pools, this will be used to detect
	// when changes have been fully propagated.
	poolList, err := machineCfgClient.MachineConfigPools().List(ctx, v1.ListOptions{})
	require.NoError(t, err)
	origPools := map[string]machineconfigurationv1.MachineConfigPool{}
	for _, pool := range poolList.Items {
		origPools[pool.GetName()] = pool
	}

	// updated will be set to true if any resources have changed, indicating tests will need to wait
	// for the cluster nodes to be updated.
	var updated bool

	// Update the OCP global pull secret which k8s needs in order to pull images from authenticated mirrors.
	updated = d.addCredToOCPGlobalPullSecret(t, ctx, reg, dce)

	// Create an ImageContentSourcePolicy
	icspName := "icsp-invalid"
	t.Logf("Applying ImageContentSourcePolicy %q", icspName)
	origIcsp, _ := opClient.ImageContentSourcePolicies().Get(ctx, icspName, v1.GetOptions{})

	yamlB := d.renderTemplate(t, "testdata/delegatedscanning/mirrors/icsp.yaml.tmpl", map[string]string{"name": icspName})
	icsp := new(operatorv1alpha1.ImageContentSourcePolicy)
	d.applyK8sYamlOrJson(t, ctx, opClient.RESTClient(), "", "imagecontentsourcepolicies", yamlB, icsp)

	if icsp.ResourceVersion != origIcsp.ResourceVersion {
		t.Logf("ImageContentSourcePolicy %q updated", icspName)
		updated = true
	}

	// Create an ImageDigestMirrorSet
	idmsName := "idms-invalid"
	t.Logf("Applying ImageDigestMirrorSet %q", idmsName)
	origIdms, _ := cfgClient.ImageDigestMirrorSets().Get(ctx, idmsName, v1.GetOptions{})

	yamlB = d.renderTemplate(t, "testdata/delegatedscanning/mirrors/idms.yaml.tmpl", map[string]string{"name": idmsName})
	idms := new(configv1.ImageDigestMirrorSet)
	d.applyK8sYamlOrJson(t, ctx, cfgClient.RESTClient(), "", "imagedigestmirrorsets", yamlB, idms)

	if idms.ResourceVersion != origIdms.ResourceVersion {
		t.Logf("ImageDigestMirrorSet %q updated", idmsName)
		updated = true
	}

	// Create an ImageTagMirrorSet
	itmsName := "itms-invalid"
	t.Logf("Applying ImageTagMirrorSet %q", itmsName)
	origItms, _ := cfgClient.ImageTagMirrorSets().Get(ctx, itmsName, v1.GetOptions{})

	yamlB = d.renderTemplate(t, "testdata/delegatedscanning/mirrors/itms.yaml.tmpl", map[string]string{"name": itmsName})
	itms := new(configv1.ImageTagMirrorSet)
	d.applyK8sYamlOrJson(t, ctx, cfgClient.RESTClient(), "", "imagetagmirrorsets", yamlB, itms)

	if itms.ResourceVersion != origItms.ResourceVersion {
		t.Logf("ImageTagMirrorSet %q updated", itmsName)
		updated = true
	}

	// If no resources were updated exit early.
	if !updated {
		t.Logf("Mirroring resources unchanged")
		return
	}

	t.Logf("Mirroring resources changed, waiting for cluster nodes to be updated")
	d.waitForNodesToProcessConfigUpdates(t, ctx, machineCfgClient, origPools)
}

// addCredToOCPGlobalPullSecret will append an entry to the OCP global pull secret,
// if no change needed returns false, true otherwise.
// Changes to the OCP global pull secret must be propagated to each node before
// usage. This method DOES NOT wait for the propagation to complete.
func (d *deleScanTestUtils) addCredToOCPGlobalPullSecret(t *testing.T, ctx context.Context, newReg string, newDce config.DockerConfigEntry) bool {
	k8s := createK8sClient(t)
	ns := "openshift-config"
	name := "pull-secret"

	secret, err := k8s.CoreV1().Secrets(ns).Get(ctx, name, v1.GetOptions{})
	require.NoError(t, err)

	key := corev1.DockerConfigJsonKey
	dataB, ok := secret.Data[key]
	require.True(t, ok, "expected secret %s to contain key %q", name, key)

	var dockerConfigJSON config.DockerConfigJSON
	require.NoError(t, json.Unmarshal(dataB, &dockerConfigJSON))

	for reg, dce := range dockerConfigJSON.Auths {
		if reg == newReg &&
			dce.Username == newDce.Username &&
			dce.Password == newDce.Password &&
			dce.Email == newDce.Email {
			// No update needed.
			t.Logf("No change needed to secret %q in namespace %q", name, ns)
			return false
		}
	}

	dockerConfigJSON.Auths[newReg] = newDce
	dataB, err = json.Marshal(dockerConfigJSON)
	require.NoError(t, err)

	secret.Data[key] = dataB
	_, err = k8s.CoreV1().Secrets(ns).Update(ctx, secret, v1.UpdateOptions{FieldManager: ignoreFieldManager})
	require.NoError(t, err)
	t.Logf("Updated secret %q in namespace %q", name, ns)
	return true
}

// waitForNodesToProcessConfigUpdates polls the status of the cluster machine config pools
// checking for evidence that the latest configuration changes have been processed and
// all nodes returned to a ready state.
func (d *deleScanTestUtils) waitForNodesToProcessConfigUpdates(t *testing.T, ctx context.Context, machineCfgClient *machineconfigurationv1client.MachineconfigurationV1Client, origPools map[string]machineconfigurationv1.MachineConfigPool) {
	ticker := time.NewTicker(15 * time.Second)
	t.Logf("Waiting for changes to be propagated")
	for {
		select {
		case <-ctx.Done():
			require.NoError(t, ctx.Err())
		case <-ticker.C:
			poolList, err := machineCfgClient.MachineConfigPools().List(ctx, v1.ListOptions{})
			require.NoError(t, err)

			d.logMachineConfigPoolsState(t, poolList, origPools)
			if !d.machineConfigPoolsReady(poolList, origPools) {
				continue
			}

			t.Logf("All nodes are now updated")
			return
		}
	}
}

// logMachineConfigPoolsState will log the status of all machine config pools for troubleshooting purposes.
func (d *deleScanTestUtils) logMachineConfigPoolsState(t *testing.T, poolList *machineconfigurationv1.MachineConfigPoolList, origPools map[string]machineconfigurationv1.MachineConfigPool) {
	w := new(tabwriter.Writer)

	sb := &strings.Builder{}
	w.Init(sb, 0, 0, 1, ' ', tabwriter.AlignRight)
	fmt.Fprintln(w, "name\torigGen\tnewGen\tstatusGen\ttotal\tready\tupdated\tdegraded\t")
	for _, pool := range poolList.Items {
		name := pool.GetName()
		fmt.Fprintf(w, "%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t\n",
			name,
			origPools[name].Generation,
			pool.GetGeneration(),
			pool.Status.ObservedGeneration,
			pool.Status.MachineCount,
			pool.Status.ReadyMachineCount,
			pool.Status.UpdatedMachineCount,
			pool.Status.DegradedMachineCount,
		)
	}
	utils.IgnoreError(w.Flush)

	t.Logf("Machine Config Pools Status: \n%s", sb.String())
}

// machineConfigPoolsReady verifies that each machine config pool has fully processed
// the updated configurations.
func (d *deleScanTestUtils) machineConfigPoolsReady(poolList *machineconfigurationv1.MachineConfigPoolList, origPools map[string]machineconfigurationv1.MachineConfigPool) bool {
	for _, pool := range poolList.Items {
		if pool.Generation <= origPools[pool.GetName()].Generation {
			// This pool has not yet started processing the updated configuration.
			return false
		}

		if pool.Generation != pool.Status.ObservedGeneration {
			// The pool indicates a configuration change has been made, but the propagation
			// status is not yet ready.
			return false
		}

		if pool.Status.MachineCount != pool.Status.ReadyMachineCount {
			// The changes have not yet been propagated to every node.
			return false
		}
	}

	return true
}

// BuildOCPInternalImage builds an image, pushes it to the OCP internal registry, and
// returns the image reference. Name is used as the k8s object name for the
// associated build configs, image streams, and final image. FromImage is used
// as the 'FROM' instruction when building the new image, which allows for differnet
// images to be created in the internal OCP image registry.
func (d *deleScanTestUtils) BuildOCPInternalImage(t *testing.T, ctx context.Context, namespace, name, fromImage string) string {
	d.applyBuildConfig(t, ctx, namespace, name)
	d.applyImageStream(t, ctx, namespace, name)
	return d.buildAndPushImage(t, ctx, namespace, name, fromImage)
}

func (d *deleScanTestUtils) applyBuildConfig(t *testing.T, ctx context.Context, namespace, name string) {
	restCfg := d.restCfg

	t.Logf("Applying BuildConfig %q", name)
	buildV1Client, err := buildv1client.NewForConfig(restCfg)
	require.NoError(t, err)

	yamlB := d.renderTemplate(t, "testdata/delegatedscanning/build-config.yaml.tmpl", map[string]string{"name": name, "namespace": namespace})
	d.applyK8sYamlOrJson(t, ctx, buildV1Client.RESTClient(), namespace, "buildconfigs", yamlB, nil)
}

func (d *deleScanTestUtils) applyImageStream(t *testing.T, ctx context.Context, namespace, name string) {
	restCfg := d.restCfg

	t.Logf("Applying ImageStream %q", name)
	imageV1Client, err := imagev1client.NewForConfig(restCfg)
	require.NoError(t, err)

	yamlB := d.renderTemplate(t, "testdata/delegatedscanning/image-stream.yaml.tmpl", map[string]string{"name": name, "namespace": namespace})
	d.applyK8sYamlOrJson(t, ctx, imageV1Client.RESTClient(), namespace, "imagestreams", yamlB, nil)
}

func (d *deleScanTestUtils) buildAndPushImage(t *testing.T, ctx context.Context, namespace, name, fromImage string) string {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	file := "testdata/delegatedscanning/Dockerfile.tmpl"
	dockerFileB := d.renderTemplate(t, file, map[string]string{"image": fromImage})

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
		Namespace(namespace).
		Resource("buildconfigs").
		Name(name).
		SubResource("instantiatebinary").
		Body(buf).
		Do(ctx).
		Into(result)
	require.NoError(t, err)

	t.Logf("Streaming build logs for %q", result.GetName())
	err = d.streamBuildLogs(ctx, client, result, namespace)
	require.NoError(t, err)

	t.Logf("Waiting for build %q to complete", result.GetName())
	err = d.waitForBuildComplete(ctx, buildV1Client.Builds(namespace), result.GetName())
	require.NoError(t, err)

	return result.Status.OutputDockerImageReference
}

// applyK8sYamlOrJson performs a server side apply using the provided client along with the
// YAML or JSON. The updated object is returned.
func (d *deleScanTestUtils) applyK8sYamlOrJson(t *testing.T, ctx context.Context, client rest.Interface, namespace, resource string, yamlOrJson []byte, result runtime.Object) {
	partialObj := &v1.PartialObjectMetadata{}
	reader := bytes.NewReader(yamlOrJson)
	require.NoError(t, yaml.NewYAMLOrJSONDecoder(reader, 1024).Decode(partialObj))
	name := partialObj.GetObjectMeta().GetName()

	force := true
	options := metav1.PatchOptions{Force: &force, FieldManager: ignoreFieldManager}
	err := client.Patch(k8sTypes.ApplyPatchType).
		Namespace(namespace).
		Resource(resource).
		Name(name).
		Body(yamlOrJson).
		VersionedParams(&options, metav1.ParameterCodec).
		Do(ctx).
		Into(result)

	require.NoError(t, err)
}

func (d *deleScanTestUtils) renderTemplate(t *testing.T, templateFile string, data map[string]string) []byte {
	tmpl, err := template.ParseFiles(templateFile)
	require.NoError(t, err)
	fileB := new(bytes.Buffer)
	err = tmpl.Execute(fileB, data)
	require.NoError(t, err)

	return fileB.Bytes()
}

// streamBuildLogs was inspired by https://github.com/openshift/oc/blob/67813c212f6625919fa42524a27c399be653a51f/pkg/cli/startbuild/startbuild.go#L507,
// it mimics the log following behavior of `oc start-build --follow=true`.
func (d *deleScanTestUtils) streamBuildLogs(ctx context.Context, buildRestClient rest.Interface, build *buildv1.Build, namespace string) error {
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
func (d *deleScanTestUtils) waitForBuildComplete(ctx context.Context, c buildv1client.BuildInterface, name string) error {
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
