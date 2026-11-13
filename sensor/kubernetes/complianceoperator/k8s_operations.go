package complianceoperator

import (
	"context"
	"fmt"
	"reflect"
	"slices"

	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/complianceoperator"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/k8sapi"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/set"
	kubeAPIErr "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
)

type scanSettingWriter struct {
	client       dynamic.Interface
	retrier      *retrier
	roleProvider func() []string
}

func (w *scanSettingWriter) createScanResources(ctx context.Context, ns string, request *central.ApplyComplianceScanConfigRequest_BaseScanSettings, cron string) error {
	roles := w.roleProvider()
	scanSetting, err := runtimeObjToUnstructured(convertCentralRequestToScanSetting(ns, request, cron, roles))
	if err != nil {
		return err
	}

	scanSettingBinding, err := runtimeObjToUnstructured(convertCentralRequestToScanSettingBinding(ns, request, "", nil))
	if err != nil {
		return err
	}

	err = w.retrier.callWithRetry(ctx, func(ctx context.Context) error {
		_, err = w.client.Resource(complianceoperator.ScanSetting.GroupVersionResource()).Namespace(ns).Create(ctx, scanSetting, v1.CreateOptions{})
		return errors.Wrapf(err, "Could not create namespaces/%s/scansettings/%s", ns, scanSetting.GetName())
	})
	if err != nil {
		return err
	}

	return w.retrier.callWithRetry(ctx, func(callCtx context.Context) error {
		_, err = w.client.Resource(complianceoperator.ScanSettingBinding.GroupVersionResource()).Namespace(ns).Create(callCtx, scanSettingBinding, v1.CreateOptions{})
		return errors.Wrapf(err, "Could not create namespaces/%s/scansettingbindings/%s", ns, scanSettingBinding.GetName())
	})
}

func (w *scanSettingWriter) updateScanConfig(ctx context.Context, ns string, request *central.ApplyComplianceScanConfigRequest_UpdateScheduledScan) error {
	// Retrieve the ScanSetting and ScanSettingBinding objects for update
	resSS := w.client.Resource(complianceoperator.ScanSetting.GroupVersionResource()).Namespace(ns)
	var ssObj *unstructured.Unstructured
	err := w.retrier.callWithRetry(ctx, func(callCtx context.Context) error {
		var err error
		ssObj, err = resSS.Get(callCtx, request.GetScanSettings().GetScanName(), v1.GetOptions{})
		return errors.Wrapf(err, "namespaces/%s/scansettings/%s not found.  Treating as a create", ns, request.GetScanSettings().GetScanName())
	})
	if err != nil {
		log.Warn(err)
	}

	resSSB := w.client.Resource(complianceoperator.ScanSettingBinding.GroupVersionResource()).Namespace(ns)
	var ssbObj *unstructured.Unstructured
	err = w.retrier.callWithRetry(ctx, func(callCtx context.Context) error {
		var err error
		ssbObj, err = resSSB.Get(callCtx, request.GetScanSettings().GetScanName(), v1.GetOptions{})
		return errors.Wrapf(err, "namespaces/%s/scansettingsbindings/%s not found.  Treating as a create", ns, request.GetScanSettings().GetScanName())
	})
	if err != nil {
		log.Warn(err)
	}

	if ssObj == nil && ssbObj == nil {
		// This is an add instead
		return w.createScanResources(ctx, ns, request.GetScanSettings(), request.GetCron())
	}

	// Invalid case because scan setting is created first, so we should not have a situation where
	// we have a scan setting binding and not a scan setting.  This probably isn't even possible.
	if ssObj == nil {
		return errors.New("scan setting binding exists without a corresponding scan setting")
	}

	roles := w.roleProvider()
	updatedScanSetting, err := updateScanSettingFromUpdateRequest(ssObj, request, roles)
	if err != nil {
		return err
	}

	var updatedScanSettingBinding *unstructured.Unstructured
	// It is possible that we successfully created the scan setting but had issues on the binding.
	if ssbObj != nil {
		updatedScanSettingBinding, err = updateScanSettingBindingFromUpdateRequest(ssbObj, request, nil)
		if err != nil {
			return err
		}
	} else {
		updatedScanSettingBinding, err = runtimeObjToUnstructured(convertCentralRequestToScanSettingBinding(ns, request.GetScanSettings(), "", nil))
		if err != nil {
			return err
		}
	}

	err = w.retrier.callWithRetryWithOnConflictCallback(ctx,
		func(callCtx context.Context) error {
			if updatedScanSetting == nil {
				return retry.MakeRetryable(errors.Errorf("updated ScanSetting %q is 'nil'", request.GetScanSettings().GetScanName()))
			}
			_, err := resSS.Update(callCtx, updatedScanSetting, v1.UpdateOptions{})
			return errors.Wrapf(err, "Could not update namespaces/%s/scansettings/%s", ns, updatedScanSetting.GetName())
		},
		func(callCtx context.Context) error {
			ssObj, err = resSS.Get(callCtx, request.GetScanSettings().GetScanName(), v1.GetOptions{})
			if err != nil {
				return errors.Wrapf(err, "unable to get namespaces/%s/scansettings/%s", ns, request.GetScanSettings().GetScanName())
			}
			updatedScanSetting, err = updateScanSettingFromUpdateRequest(ssObj, request, roles)
			return err
		})
	if err != nil {
		return err
	}

	// Process SSB as an update
	if ssbObj != nil {
		return w.retrier.callWithRetryWithOnConflictCallback(ctx,
			func(callCtx context.Context) error {
				if updatedScanSettingBinding == nil {
					return retry.MakeRetryable(errors.Errorf("updated ScanSettingBinding %q is 'nil'", request.GetScanSettings().GetScanName()))
				}
				_, err = resSSB.Update(callCtx, updatedScanSettingBinding, v1.UpdateOptions{})
				return errors.Wrapf(err, "Could not update namespaces/%s/scansettingbindings/%s", ns, updatedScanSettingBinding.GetName())
			},
			func(callCtx context.Context) error {
				ssbObj, err = resSSB.Get(callCtx, request.GetScanSettings().GetScanName(), v1.GetOptions{})
				if err != nil {
					return errors.Wrapf(err, "unable to get namespaces/%s/scansettingbindings/%s", ns, request.GetScanSettings().GetScanName())
				}
				updatedScanSettingBinding, err = updateScanSettingBindingFromUpdateRequest(ssbObj, request, nil)
				return err
			})
	}

	return w.retrier.callWithRetry(ctx, func(callCtx context.Context) error {
		_, err = w.client.Resource(complianceoperator.ScanSettingBinding.GroupVersionResource()).Namespace(ns).Create(callCtx, updatedScanSettingBinding, v1.CreateOptions{})
		return errors.Wrapf(err, "Could not create namespaces/%s/scansettingbindings/%s", ns, updatedScanSettingBinding.GetName())
	})
}

func (w *scanSettingWriter) rerunScan(ctx context.Context, ns string, scanName string) error {
	// Conventionally, compliance scan can be rerun by applying an
	// annotation to ComplianceScan CR. The ComplianceScan CRs can be
	// found from the ComplianceSuite CR. Note that the ComplianceSuite
	// created from a scan configuration have the same name as scan
	// configuration.
	resI := w.client.Resource(complianceoperator.ComplianceSuite.GroupVersionResource()).Namespace(ns)
	var obj *unstructured.Unstructured
	err := w.retrier.callWithRetry(ctx, func(callCtx context.Context) error {
		var err error
		obj, err = resI.Get(callCtx, scanName, v1.GetOptions{})
		return errors.Wrapf(err, "namespaces/%s/compliancesuites/%s not found", ns, scanName)
	})
	if err != nil || obj == nil {
		return err
	}

	var complianceSuite v1alpha1.ComplianceSuite
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &complianceSuite); err != nil {
		return errors.Wrap(err, "Could not convert unstructured to compliance suite")
	}

	// Apply annotation to indicate compliance scan be rerun.
	for _, scan := range complianceSuite.Spec.Scans {
		if err := w.reRunSingleScan(ctx, scan, ns); err != nil {
			return err
		}
	}
	return nil
}

func (w *scanSettingWriter) reRunSingleScan(ctx context.Context, scan v1alpha1.ComplianceScanSpecWrapper, ns string) error {
	var obj *unstructured.Unstructured
	resI := w.client.Resource(complianceoperator.ComplianceScan.GroupVersionResource()).Namespace(ns)
	// Get the scan and apply annotation to indicate compliance scan be rerun.
	complianceScanName, scanObj, err := getScanForReRun(func() (*unstructured.Unstructured, error) {
		err := w.retrier.callWithRetry(ctx, func(callCtx context.Context) error {
			var err error
			obj, err = resI.Get(callCtx, scan.Name, v1.GetOptions{})
			return errors.Wrapf(err, "namespaces/%s/compliancescans/%s not found", ns, scan.Name)
		})
		return obj, err
	}, scan.Name)
	if err != nil {
		return err
	}
	return w.retrier.callWithRetryWithOnConflictCallback(ctx,
		func(callCtx context.Context) error {
			log.Infof("Rerunning compliance scan %s", complianceScanName)
			if scanObj == nil {
				return retry.MakeRetryable(errors.Errorf("Scan %q is 'nil'", complianceScanName))
			}
			_, err = resI.Update(callCtx, scanObj, v1.UpdateOptions{})
			return errors.Wrapf(err, "Could not update namespaces/%s/compliancescans/%s", ns, complianceScanName)
		},
		func(callCtx context.Context) error {
			complianceScanName, scanObj, err = getScanForReRun(func() (*unstructured.Unstructured, error) {
				return resI.Get(callCtx, complianceScanName, v1.GetOptions{})
			}, scan.Name)
			return err
		})
}

func getScanForReRun(getFunc func() (*unstructured.Unstructured, error), scanName string) (string, *unstructured.Unstructured, error) {
	obj, err := getFunc()
	if err != nil || obj == nil {
		return scanName, nil, err
	}
	var complianceScan v1alpha1.ComplianceScan
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &complianceScan); err != nil {
		return scanName, nil, errors.Wrap(err, "Could not convert unstructured to compliance scan")
	}

	if complianceScan.GetAnnotations() == nil {
		complianceScan.Annotations = make(map[string]string)
	}
	complianceScan.Annotations[rescanAnnotation] = ""

	obj, err = runtimeObjToUnstructured(&complianceScan)
	if err != nil {
		return complianceScan.Name, nil, err
	}
	return complianceScan.Name, obj, nil
}

func (w *scanSettingWriter) changeScanSchedule(ctx context.Context, ns string, config scanScheduleConfiguration) error {
	obj, err := getScanSettingForUpdate(func() (*unstructured.Unstructured, error) {
		resI := w.client.Resource(complianceoperator.ScanSetting.GroupVersionResource()).Namespace(ns)
		var obj *unstructured.Unstructured
		err := w.retrier.callWithRetry(ctx, func(callCtx context.Context) error {
			var err error
			obj, err = resI.Get(callCtx, config.ScanName, v1.GetOptions{})
			return errors.Wrapf(err, "namespaces/%s/scansettings/%s not found", ns, config.ScanName)
		})
		return obj, err
	}, config)
	if err != nil {
		return err
	}
	resI := w.client.Resource(complianceoperator.ScanSetting.GroupVersionResource()).Namespace(ns)
	return w.retrier.callWithRetryWithOnConflictCallback(ctx,
		func(callCtx context.Context) error {
			if obj == nil {
				return retry.MakeRetryable(errors.Errorf("ScanSetting %q is 'nil'", config.ScanName))
			}
			_, err := resI.Update(callCtx, obj, v1.UpdateOptions{})
			return errors.Wrapf(err, "Could not update namespaces/%s/scansettings/%s", ns, config.ScanName)
		},
		func(callCtx context.Context) error {
			obj, err = getScanSettingForUpdate(func() (*unstructured.Unstructured, error) {
				return resI.Get(callCtx, config.ScanName, v1.GetOptions{})
			}, config)
			if err != nil {
				return errors.Wrapf(err, "unable to retrieve ScanSetting %s", config.ScanName)
			}
			return nil
		})
}

func getScanSettingForUpdate(getFunc func() (*unstructured.Unstructured, error), config scanScheduleConfiguration) (*unstructured.Unstructured, error) {
	obj, err := getFunc()
	if err != nil || obj == nil {
		return nil, err
	}

	var scanSetting v1alpha1.ScanSetting
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &scanSetting); err != nil {
		return nil, errors.Wrap(err, "Could not convert unstructured to scan setting")
	}

	// Check if scanSetting has the "Suspend" field
	if _, ok := reflect.TypeFor[v1alpha1.ScanSetting]().FieldByName("Suspend"); ok {
		scanSetting.Suspend = *config.Suspend
	} else {
		// Handle the case where the field doesn't exist (older CRD)
		return nil, errors.New("suspending a scan is not supported on this version of the compliance operator")
	}

	obj, err = runtimeObjToUnstructured(&scanSetting)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func (w *scanSettingWriter) deleteScanConfig(ctx context.Context, ns string, name string) error {
	deletePolicy := v1.DeletePropagationForeground
	scanSettingBindingResourceI := w.client.Resource(complianceoperator.ScanSettingBinding.GroupVersionResource()).Namespace(ns)
	err := w.retrier.callWithRetry(ctx, func(callCtx context.Context) error {
		return scanSettingBindingResourceI.Delete(callCtx, name, v1.DeleteOptions{PropagationPolicy: &deletePolicy})
	})
	if err != nil && !kubeAPIErr.IsNotFound(err) {
		return errors.Wrapf(err, "Could not delete namespaces/%s/scansettingbindings/%s", ns, name)
	}

	scanSettingResourceI := w.client.Resource(complianceoperator.ScanSetting.GroupVersionResource()).Namespace(ns)
	err = w.retrier.callWithRetry(ctx, func(callCtx context.Context) error {
		return scanSettingResourceI.Delete(callCtx, name, v1.DeleteOptions{PropagationPolicy: &deletePolicy})
	})
	if err != nil && !kubeAPIErr.IsNotFound(err) {
		return errors.Wrapf(err, "Could not delete namespaces/%s/scansettings/%s", ns, name)
	}
	return nil
}

func (w *scanSettingWriter) syncScanConfigs(ctx context.Context, ns string, request *central.SyncComplianceScanConfigRequest) error {
	// List all ScanSettings in the cluster with the `stackrox` label.
	scanSettingsInCluster, err := w.getResourcesInCluster(ctx, complianceoperator.ScanSetting)
	if err != nil {
		return err
	}
	// List all ScanSettingBindings in the cluster with the `stackrox` label.
	scanSettingBindingsInCluster, err := w.getResourcesInCluster(ctx, complianceoperator.ScanSettingBinding)
	if err != nil {
		return err
	}

	// Compare with the ScanConfig in the request.
	var errList errorhelpers.ErrorList
	inCentralSet := set.NewStringSet()
	for _, scanCfg := range request.GetScanConfigs() {
		if err := validateUpdateScheduledScanConfigRequest(scanCfg.GetUpdateScan()); err != nil {
			errList.AddError(err)
			continue
		}
		inCentralSet.Add(generateScanIndex(ns, scanCfg.GetUpdateScan().GetScanSettings().GetScanName()))
		// Reconcile ScanSetting
		if err := w.reconcileCreateOrUpdateResource(
			ctx, ns,
			scanCfg.GetUpdateScan(),
			scanSettingsInCluster,
			updateScanSettingFromUpdateRequest,
			convertCentralRequestToScanSetting,
			complianceoperator.ScanSetting,
		); err != nil {
			errList.AddError(err)
		}
		// Reconcile ScanSettingBinding
		if err := w.reconcileCreateOrUpdateResource(
			ctx, ns,
			scanCfg.GetUpdateScan(),
			scanSettingBindingsInCluster,
			updateScanSettingBindingFromUpdateRequest,
			convertCentralRequestToScanSettingBinding,
			complianceoperator.ScanSettingBinding,
		); err != nil {
			errList.AddError(err)
		}
	}
	// Delete ScanSettingBindings that are no longer in Central
	if err := w.reconcileDeleteResource(ctx, inCentralSet, scanSettingBindingsInCluster, complianceoperator.ScanSettingBinding); err != nil {
		errList.AddError(err)
	}
	// Delete ScanSettings that are no longer in Central
	if err := w.reconcileDeleteResource(ctx, inCentralSet, scanSettingsInCluster, complianceoperator.ScanSetting); err != nil {
		errList.AddError(err)
	}
	return errList.ToError()
}

func (w *scanSettingWriter) getResourcesInCluster(ctx context.Context, api k8sapi.APIResource) (map[string]unstructured.Unstructured, error) {
	resourceInterface := w.client.Resource(api.GroupVersionResource())
	var resourcesInCluster *unstructured.UnstructuredList
	err := w.retrier.callWithRetry(ctx, func(callCtx context.Context) error {
		var err error
		resourcesInCluster, err = resourceInterface.List(callCtx, v1.ListOptions{LabelSelector: labels.SelectorFromSet(stackroxLabels).String()})
		return errors.Wrap(err, "listing resources in cluster")
	})
	if err != nil {
		return nil, err
	}
	resourcesInClusterMap := make(map[string]unstructured.Unstructured)
	for _, resource := range resourcesInCluster.Items {
		log.Debugf("%s in the cluster: %s", api.Kind, resource.GetName())
		resourcesInClusterMap[generateScanIndex(resource.GetNamespace(), resource.GetName())] = resource
	}
	return resourcesInClusterMap, nil
}

func (w *scanSettingWriter) reconcileCreateOrUpdateResource(
	ctx context.Context,
	namespace string,
	req *central.ApplyComplianceScanConfigRequest_UpdateScheduledScan,
	inClusterResources map[string]unstructured.Unstructured,
	updateFn updateFunction,
	convertFn convertFunction,
	api k8sapi.APIResource,
) error {
	namespaceNameIndex := generateScanIndex(namespace, req.GetScanSettings().GetScanName())
	if resource, isInCluster := inClusterResources[namespaceNameIndex]; isInCluster {
		// Update Resource
		log.Debugf("Update %s %s", api.Kind, req.GetScanSettings().GetScanName())
		delete(inClusterResources, namespaceNameIndex)
		roles := w.roleProvider()
		updatedResource, err := updateFn(&resource, req, roles)
		if err != nil {
			return err
		}
		return w.retrier.callWithRetryWithOnConflictCallback(ctx,
			func(callCtx context.Context) error {
				if updatedResource == nil {
					return retry.MakeRetryable(errors.Errorf("updated %s %q is 'nil'", api.GroupVersionResource(), resource.GetName()))
				}
				_, err := w.client.Resource(api.GroupVersionResource()).Namespace(namespace).Update(callCtx, updatedResource, v1.UpdateOptions{})
				return errors.Wrapf(err, "updating namespace %q", namespace)
			},
			func(callCtx context.Context) error {
				currentResource, err := w.client.Resource(api.GroupVersionResource()).Namespace(namespace).Get(callCtx, resource.GetName(), v1.GetOptions{})
				if err != nil {
					return errors.Wrapf(err, "unable to get namespaces/%s/scans/%s", namespace, resource.GetName())
				}
				updatedResource, err = updateFn(currentResource, req, roles)
				return err
			})
	}

	// The Resource is in Central but not in the cluster
	log.Debugf("Create %s %s", api.Kind, req.GetScanSettings().GetScanName())
	roles := w.roleProvider()
	resource, err := runtimeObjToUnstructured(convertFn(namespace, req.GetScanSettings(), req.GetCron(), roles))
	if err != nil {
		return err
	}
	return w.retrier.callWithRetry(ctx, func(callCtx context.Context) error {
		_, err := w.client.Resource(api.GroupVersionResource()).Namespace(namespace).Create(callCtx, resource, v1.CreateOptions{})
		return errors.Wrapf(err, "Could not create namespaces/%s/%s/%s", namespace, api.Name, resource.GetName())
	})
}

func (w *scanSettingWriter) reconcileDeleteResource(ctx context.Context, inCentral set.StringSet, inClusterResources map[string]unstructured.Unstructured, api k8sapi.APIResource) error {
	var errList errorhelpers.ErrorList
	deletePolicy := v1.DeletePropagationForeground
	// Delete Resources that are no longer in Central
	for index, resource := range inClusterResources {
		if !inCentral.Contains(index) {
			// The Resource is in the cluster but not in Central
			log.Debugf("Delete %s %s", api.Kind, resource.GetName())
			cli := w.client.Resource(api.GroupVersionResource()).Namespace(resource.GetNamespace())
			err := w.retrier.callWithRetry(ctx, func(callCtx context.Context) error {
				return cli.Delete(callCtx, resource.GetName(), v1.DeleteOptions{PropagationPolicy: &deletePolicy})
			})
			if err != nil && !kubeAPIErr.IsNotFound(err) {
				errList.AddError(err)
			}
		}
	}
	return errList.ToError()
}

func (w *scanSettingWriter) updateAllScanSettingRoles(ctx context.Context, ns string, roles []string) error {
	scanSettings, err := w.getResourcesInCluster(ctx, complianceoperator.ScanSetting)
	if err != nil {
		return err
	}
	var errList errorhelpers.ErrorList
	for _, ss := range scanSettings {
		if err := w.updateScanSettingRoles(ctx, ns, &ss, roles); err != nil {
			errList.AddError(errors.Wrapf(err, "updating roles for scan setting %s", ss.GetName()))
		}
	}
	return errList.ToError()
}

func (w *scanSettingWriter) updateScanSettingRoles(ctx context.Context, ns string, obj *unstructured.Unstructured, roles []string) error {
	var scanSetting v1alpha1.ScanSetting
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &scanSetting); err != nil {
		return errors.Wrap(err, "converting unstructured to scan setting")
	}

	if slices.Equal(slices.Sorted(slices.Values(scanSetting.Roles)), slices.Sorted(slices.Values(roles))) {
		return nil
	}

	scanSetting.Roles = roles
	updated, err := runtimeObjToUnstructured(&scanSetting)
	if err != nil {
		return err
	}

	resI := w.client.Resource(complianceoperator.ScanSetting.GroupVersionResource()).Namespace(ns)
	return w.retrier.callWithRetryWithOnConflictCallback(ctx,
		func(callCtx context.Context) error {
			_, err := resI.Update(callCtx, updated, v1.UpdateOptions{})
			return errors.Wrapf(err, "updating scan setting %s roles", obj.GetName())
		},
		func(callCtx context.Context) error {
			current, err := resI.Get(callCtx, obj.GetName(), v1.GetOptions{})
			if err != nil {
				return err
			}
			var currentSS v1alpha1.ScanSetting
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(current.Object, &currentSS); err != nil {
				return err
			}
			currentSS.Roles = roles
			updated, err = runtimeObjToUnstructured(&currentSS)
			return err
		})
}

func generateScanIndex(namespace string, scanName string) string {
	return fmt.Sprintf("%s-%s", namespace, scanName)
}
