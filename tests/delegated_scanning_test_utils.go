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
	operatorv1client "github.com/openshift/client-go/operator/clientset/versioned/typed/operator/v1"
	operatorv1alpha1client "github.com/openshift/client-go/operator/clientset/versioned/typed/operator/v1alpha1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/docker/config"
	imgUtils "github.com/stackrox/rox/pkg/images/utils"
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
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// This file contains utilities to aid the execution of the delegated scanning tests.
// The current implementation is specific to the needs of delegated scanning.

const (
	// ignoreFieldManager is the name of the field manager to pass the k8s API.
	// The k8s API uses the name of a field manager to detect conflicts when
	// making changes via server side apply.
	ignoreFieldManager = "Ignore"
)

// deleScanTestImage holds a full reference to an image and provides
// methods for obtaining the string representation in various
// formats (syntax sugar).
type deleScanTestImage struct {
	cImage *storage.ContainerImage
}

func NewDeleScanTestImage(t *testing.T, imageStr string) deleScanTestImage {
	cImage, err := imgUtils.GenerateImageFromString(imageStr)
	require.NoError(t, err)

	return deleScanTestImage{cImage: cImage}
}

// ID returns the digest of the image.
func (d deleScanTestImage) ID() string {
	return d.cImage.GetId()
}

// ByID returns the image referenced by ID.
func (d deleScanTestImage) IDRef() string {
	name := d.cImage.GetName()
	return fmt.Sprintf("%s/%s@%s", name.GetRegistry(), name.GetRemote(), d.cImage.GetId())
}

// ByID returns the image referenced by Tag.
func (d deleScanTestImage) TagRef() string {
	name := d.cImage.GetName()
	return fmt.Sprintf("%s/%s:%s", name.GetRegistry(), name.GetRemote(), name.GetTag())
}

// Base returns the registry + repo only, it will NOT include the ID or tag.
func (d deleScanTestImage) Base() string {
	return fmt.Sprintf("%s/%s", d.cImage.GetName().GetRegistry(), d.cImage.GetName().GetRemote())
}

// WithReg creates a new instance with only the registry changed.
func (d deleScanTestImage) WithReg(reg string) deleScanTestImage {
	cImage := d.cImage.CloneVT()
	cImage.GetName().Registry = reg

	return deleScanTestImage{cImage: cImage}
}

// deleScanTestUtils contains helper functions to assist delegated scanning tests.
type deleScanTestUtils struct {
	restCfg *rest.Config

	apiResourceList []*metav1.APIResourceList
}

func NewDeleScanTestUtils(t *testing.T, restCfg *rest.Config, apiResourceList []*metav1.APIResourceList) *deleScanTestUtils {
	utils := &deleScanTestUtils{
		restCfg:         restCfg,
		apiResourceList: apiResourceList,
	}

	return utils
}

// apiResourceSupported will return true if the cluster has an API resources that matches
// group and name, false otherwise.
func (d *deleScanTestUtils) apiResourceSupported(groupVersion, name string) bool {
	for _, list := range d.apiResourceList {
		gv := list.GroupVersion
		if gv != groupVersion {
			continue
		}
		for _, resource := range list.APIResources {
			if resource.Name == name {
				return true
			}
		}
	}

	return false
}

// SetupMirrors will add creds to the Global Pull Secret, and create the mirroring CRs
// (ImageContentSourcePolicy, ImageDigestMirrorSet, and ImageTagMirrorSet). If changes
// were made will wait for changes to be propagated to all nodes. During testing on an
// OCP cluster with 3 master + 2 worker nodes the average time to propagate these
// changes was between 5-10 minutes.
//
// Will return 3 bools which indicate if the associated mirroring CR is supported.
// 1. ImageContentSourcePolicy
// 2. ImageDigestMirrorSet
// 3. ImageTagMirrorSet
func (d *deleScanTestUtils) SetupMirrors(t *testing.T, ctx context.Context, reg string, dce config.DockerConfigEntry) (icspSupported bool, idmsSupported bool, itmsSupported bool) {
	start := time.Now()
	defer func() {
		logf(t, "Setting up mirrors took:: %v", time.Since(start))
	}()

	// Instantiate various API clients.
	machineCfgClient, err := machineconfigurationv1client.NewForConfig(d.restCfg)
	require.NoError(t, err)

	opClient, err := operatorv1alpha1client.NewForConfig(d.restCfg)
	require.NoError(t, err)

	cfgClient, err := configv1client.NewForConfig(d.restCfg)
	require.NoError(t, err)

	// Get current status of the machine config pools, this will be used to detect
	// when changes have been fully propagated.
	poolList, err := machineCfgClient.MachineConfigPools().List(ctx, v1.ListOptions{})
	require.NoError(t, err)
	origPools := map[string]machineconfigurationv1.MachineConfigPool{}
	for _, pool := range poolList.Items {
		origPools[pool.GetName()] = pool
	}

	// Update the OCP global pull secret, which k8s needs in order to pull images from authenticated mirrors.
	updated := d.addCredToOCPGlobalPullSecret(t, ctx, reg, dce)

	// Identify which mirroring CRs are supported for creation.
	icspSupported = d.apiResourceSupported("operator.openshift.io/v1alpha1", "imagecontentsourcepolicies")
	idmsSupported = d.apiResourceSupported("config.openshift.io/v1", "imagedigestmirrorsets")
	itmsSupported = d.apiResourceSupported("config.openshift.io/v1", "imagetagmirrorsets")

	if icspSupported {
		// Create an ImageContentSourcePolicy.
		icspName := "icsp-invalid"
		logf(t, "Applying ImageContentSourcePolicy %q", icspName)
		origIcsp, _ := opClient.ImageContentSourcePolicies().Get(ctx, icspName, v1.GetOptions{})

		yamlB := d.renderTemplate(t, "testdata/delegatedscanning/mirrors/icsp.yaml.tmpl", map[string]string{"name": icspName})
		icsp := new(operatorv1alpha1.ImageContentSourcePolicy)
		d.mustApplyK8sYamlOrJson(t, ctx, opClient.RESTClient(), "", "imagecontentsourcepolicies", yamlB, icsp)

		if icsp.ResourceVersion != origIcsp.ResourceVersion {
			logf(t, "ImageContentSourcePolicy %q updated", icspName)
			updated = true
		}
	}

	if idmsSupported {
		// Create an ImageDigestMirrorSet.
		idmsName := "idms-invalid"
		logf(t, "Applying ImageDigestMirrorSet %q", idmsName)
		origIdms, _ := cfgClient.ImageDigestMirrorSets().Get(ctx, idmsName, v1.GetOptions{})

		yamlB := d.renderTemplate(t, "testdata/delegatedscanning/mirrors/idms.yaml.tmpl", map[string]string{"name": idmsName})
		idms := new(configv1.ImageDigestMirrorSet)
		d.mustApplyK8sYamlOrJson(t, ctx, cfgClient.RESTClient(), "", "imagedigestmirrorsets", yamlB, idms)

		if idms.ResourceVersion != origIdms.ResourceVersion {
			logf(t, "ImageDigestMirrorSet %q updated", idmsName)
			updated = true
		}
	}

	if itmsSupported {
		// Create an ImageTagMirrorSet.
		itmsName := "itms-invalid"
		logf(t, "Applying ImageTagMirrorSet %q", itmsName)
		origItms, _ := cfgClient.ImageTagMirrorSets().Get(ctx, itmsName, v1.GetOptions{})

		yamlB := d.renderTemplate(t, "testdata/delegatedscanning/mirrors/itms.yaml.tmpl", map[string]string{"name": itmsName})
		itms := new(configv1.ImageTagMirrorSet)
		d.mustApplyK8sYamlOrJson(t, ctx, cfgClient.RESTClient(), "", "imagetagmirrorsets", yamlB, itms)

		if itms.ResourceVersion != origItms.ResourceVersion {
			logf(t, "ImageTagMirrorSet %q updated", itmsName)
			updated = true
		}

	}

	// If no resources were updated exit early.
	if !updated {
		logf(t, "Mirroring resources unchanged")
		return
	}

	logf(t, "Mirroring resources changed, waiting for cluster nodes to be updated")
	d.waitForNodesToProcessConfigUpdates(t, ctx, machineCfgClient, origPools)
	return
}

// addCredToOCPGlobalPullSecret will append an entry to the OCP global pull secret,
// if no change needed returns false, true otherwise.
//
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

			logf(t, "No change needed to secret %q in namespace %q", name, ns)
			return false
		}
	}

	dockerConfigJSON.Auths[newReg] = newDce
	dataB, err = json.Marshal(dockerConfigJSON)
	require.NoError(t, err)

	secret.Data[key] = dataB
	_, err = k8s.CoreV1().Secrets(ns).Update(ctx, secret, v1.UpdateOptions{FieldManager: ignoreFieldManager})
	require.NoError(t, err)
	logf(t, "Updated secret %q in namespace %q", name, ns)
	return true
}

// waitForNodesToProcessConfigUpdates polls the status of the cluster machine config pools
// checking for evidence that the latest configuration changes have been processed and
// all nodes returned to a ready state.
func (d *deleScanTestUtils) waitForNodesToProcessConfigUpdates(t *testing.T, ctx context.Context, machineCfgClient *machineconfigurationv1client.MachineconfigurationV1Client, origPools map[string]machineconfigurationv1.MachineConfigPool) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	logf(t, "Waiting for changes to be propagated")
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

			logf(t, "All nodes are now updated")
			return
		}
	}
}

// logMachineConfigPoolsState logs the status of all machine config pools.
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

	logf(t, "Machine Config Pools Status: \n%s", sb.String())
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

// DeploySleeperImage creates a deployment using imageStr that will sleep forever.
func (d *deleScanTestUtils) DeploySleeperImage(t *testing.T, ctx context.Context, namespace, name, imageStr string) {
	yamlB := d.renderTemplate(t, "testdata/delegatedscanning/deploys/generic.yaml.tmpl", map[string]string{
		"name":      name,
		"namespace": namespace,
		"image":     imageStr,
	})

	clientset, err := kubernetes.NewForConfig(d.restCfg)
	require.NoError(t, err)

	d.mustApplyK8sYamlOrJson(t, ctx, clientset.AppsV1().RESTClient(), namespace, "deployments", yamlB, nil)
	logf(t, "Deployed image: %q", imageStr)
}

// BuildOCPInternalImage builds an image, pushes it to the OCP internal registry, and
// returns the image reference. Name is used as the k8s metadata.name for the
// associated build configs, image streams, and final image. FromImage is used
// as the 'FROM' instruction when building the new image, which allows for differnet
// images to be created in the OCP internal registry.
func (d *deleScanTestUtils) BuildOCPInternalImage(t *testing.T, ctx context.Context, namespace, name, fromImage string) deleScanTestImage {
	d.applyBuildConfig(t, ctx, namespace, name)
	d.applyImageStream(t, ctx, namespace, name)

	imgStrWithTag := d.buildAndPushImage(t, ctx, namespace, name, fromImage)
	digest := d.getDigestFromImageStreamTag(t, ctx, namespace, name, "latest")

	return NewDeleScanTestImage(t, fmt.Sprintf("%s@%s", imgStrWithTag, digest))
}

// applyBuildConfig creates a build config.
func (d *deleScanTestUtils) applyBuildConfig(t *testing.T, ctx context.Context, namespace, name string) {
	restCfg := d.restCfg

	logf(t, "Applying BuildConfig %q", name)
	buildV1Client, err := buildv1client.NewForConfig(restCfg)
	require.NoError(t, err)

	yamlB := d.renderTemplate(t, "testdata/delegatedscanning/build-config.yaml.tmpl", map[string]string{"name": name, "namespace": namespace})
	d.mustApplyK8sYamlOrJson(t, ctx, buildV1Client.RESTClient(), namespace, "buildconfigs", yamlB, nil)
}

// applyBuildConfig creates an image stream.
func (d *deleScanTestUtils) applyImageStream(t *testing.T, ctx context.Context, namespace, name string) {
	restCfg := d.restCfg

	logf(t, "Applying ImageStream %q", name)
	imageV1Client, err := imagev1client.NewForConfig(restCfg)
	require.NoError(t, err)

	yamlB := d.renderTemplate(t, "testdata/delegatedscanning/image-stream.yaml.tmpl", map[string]string{"name": name, "namespace": namespace})
	d.mustApplyK8sYamlOrJson(t, ctx, imageV1Client.RESTClient(), namespace, "imagestreams", yamlB, nil)
}

// getDigestFromImageStreamTag extracts an image's digest from an image stream tag.
func (d *deleScanTestUtils) getDigestFromImageStreamTag(t *testing.T, ctx context.Context, namespace, name, tag string) string {
	restCfg := d.restCfg

	nameTag := fmt.Sprintf("%s:%s", name, tag)

	logf(t, "Getting image digest from image stream tag: %v", nameTag)
	imageV1Client, err := imagev1client.NewForConfig(restCfg)
	require.NoError(t, err)

	isTag, err := imageV1Client.ImageStreamTags(namespace).Get(ctx, nameTag, v1.GetOptions{})
	require.NoError(t, err)

	digest := isTag.Image.GetObjectMeta().GetName()
	require.NotEmpty(t, digest)
	logf(t, "Digest found: %s", digest)

	return digest
}

// buildAndPushImage will build an image, wait for it to complete, and push it to the OCP internal registry.
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
	logf(t, "Starting build for %q", name)
	err = client.Post().
		Namespace(namespace).
		Resource("buildconfigs").
		Name(name).
		SubResource("instantiatebinary").
		Body(buf).
		Do(ctx).
		Into(result)
	require.NoError(t, err)

	logf(t, "Streaming build logs for %q", result.GetName())
	err = d.streamBuildLogs(ctx, client, result, namespace)
	require.NoError(t, err)

	logf(t, "Waiting for build %q to complete", result.GetName())
	err = d.waitForBuildComplete(ctx, buildV1Client.Builds(namespace), result.GetName())
	require.NoError(t, err)

	return result.Status.OutputDockerImageReference
}

// mustApplyK8sYamlOrJson mimics applyK8sYamlOrJson but fails the test when any errors are
// encountered.
func (d *deleScanTestUtils) mustApplyK8sYamlOrJson(t *testing.T, ctx context.Context, client rest.Interface, namespace, resource string, yamlOrJson []byte, result runtime.Object) {
	err := d.applyK8sYamlOrJson(t, ctx, client, namespace, resource, yamlOrJson, result)
	require.NoError(t, err)
}

// applyK8sYamlOrJson performs a server side apply using the provided client along with the
// YAML or JSON. Result is populated with the updated object.
func (d *deleScanTestUtils) applyK8sYamlOrJson(t *testing.T, ctx context.Context, client rest.Interface, namespace, resource string, yamlOrJson []byte, result runtime.Object) error {
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

	return err
}

// renderTemplate renders a go template using the data provided.
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

// DisableMCONodeDrain configures the OpenShift Machine Config Operator to not drain
// nodes when mirroring CRs are applied. This is best effort as the capability was
// backported and may only be available to some MCO z-stream releases.
//
// This should speed up mirroring tests and reduce flakes on OCP versions that would
// otherwise perform drains.
//
// The test will only fail if static testdata files are not found.
func (d *deleScanTestUtils) DisableMCONodeDrain(t *testing.T, ctx context.Context) error {
	crFound, err := d.applyMachineConfiguration(t, ctx)
	if err != nil {
		logf(t, "Error occured disabling node drain via machineconfigurations: %v", err)
	}
	if crFound && err == nil {
		// The machine configuration was successfully applied, short-circuit.
		return nil
	}

	// Otherwise apply the drain override configmap if the namespace exists.
	mcoNamespace := "openshift-machine-config-operator"
	k8s := createK8sClient(t)
	_, err = k8s.CoreV1().Namespaces().Get(ctx, mcoNamespace, v1.GetOptions{})
	if err != nil {
		return fmt.Errorf("checking if namespace %q exists: %w", mcoNamespace, err)
	}

	// The MCO namespace exists, apply the configmap.
	logf(t, "Applying image registry drain override configmap to namespace %q", mcoNamespace)

	yamlB, err := os.ReadFile("testdata/delegatedscanning/mirrors/image-registry-override-drain-configmap.yaml")
	require.NoError(t, err, "expected image-registry-override-drain-configmap.yaml not found in testdata")

	err = d.applyK8sYamlOrJson(t, ctx, k8s.CoreV1().RESTClient(), mcoNamespace, "configmaps", yamlB, nil)
	if err != nil {
		return fmt.Errorf("applying image drain configmap to namespace %q: %w", mcoNamespace, err)
	}

	return nil
}

// applyMachineConfiguration attempts to apply the machine configuration CR, will return true
// if the CR is supported.
func (d *deleScanTestUtils) applyMachineConfiguration(t *testing.T, ctx context.Context) (bool, error) {
	if d.apiResourceSupported("operator.openshift.io/v1", "machineconfigurations") {
		logf(t, "Cluster supports machineconfigurations CR, disabling node drain")

		yamlB, err := os.ReadFile("testdata/delegatedscanning/mirrors/machineconfiguration.yaml")
		require.NoError(t, err, "expected machineconfiguration.yaml not found in testdata")

		operatorClient, err := operatorv1client.NewForConfig(d.restCfg)
		if err != nil {
			return true, err
		}

		err = d.applyK8sYamlOrJson(t, ctx, operatorClient.RESTClient(), "", "machineconfigurations", yamlB, nil)
		return true, err
	}
	return false, nil
}
