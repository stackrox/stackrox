import React, { ReactElement, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom-v5-compat';
import {
    Alert,
    Breadcrumb,
    BreadcrumbItem,
    Bullseye,
    Button,
    Divider,
    Flex,
    FlexItem,
    PageSection,
    Spinner,
    Title,
} from '@patternfly/react-core';
import cloneDeep from 'lodash/cloneDeep';
import get from 'lodash/get';
import set from 'lodash/set';

import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import PageTitle from 'Components/PageTitle';
import useAnalytics, { CLUSTER_CREATED } from 'hooks/useAnalytics';
import useFeatureFlags from 'hooks/useFeatureFlags';
import useInterval from 'hooks/useInterval';
import useMetadata from 'hooks/useMetadata';
import usePermissions from 'hooks/usePermissions';
import {
    fetchClusterWithRetentionInformation,
    saveCluster,
    downloadClusterYaml,
    getClusterDefaults,
} from 'services/ClustersService';
import { Cluster, ClusterManagerType } from 'types/cluster.proto';
import { DecommissionedClusterRetentionInfo } from 'types/clusterService.proto';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { clustersBasePath } from 'routePaths';

import ClusterLabelsConfigurationStatusSummary from './ClusterLabelsConfigurationStatusSummary';
import ClusterEditFormLegacy from './ClusterEditFormLegacy';
import ClusterDeployment from './ClusterDeployment';
import DownloadHelmValues from './DownloadHelmValues';
import { clusterDetailPollingInterval, newClusterDefault } from './cluster.helpers';

const requiredKeys = ['name', 'type', 'mainImage', 'centralApiEndpoint'];

const validate = (values) => {
    const errors = {};

    requiredKeys.forEach((key) => {
        if (values[key].length === 0) {
            errors[key] = 'This field is required';
        }
    });

    return errors;
};

type WizardStep = 'FORM' | 'DEPLOYMENT';

type SubmissionError = {
    title: string;
    text: string;
};

type MessageState = {
    variant: 'warning' | 'danger';
    title: string;
    text: string;
};

export type ClusterPageProps = {
    clusterId: string;
};

function ClusterPage({ clusterId }: ClusterPageProps): ReactElement {
    const navigate = useNavigate();
    const { hasReadWriteAccess } = usePermissions();
    const hasWriteAccessForCluster = hasReadWriteAccess('Cluster');
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isAdmissionControllerConfigEnabled = !isFeatureFlagEnabled(
        'ROX_ADMISSION_CONTROLLER_CONFIG'
    );

    const metadata = useMetadata();
    const { analyticsTrack } = useAnalytics();

    const defaultCluster = cloneDeep(newClusterDefault) as unknown as Cluster;

    const [selectedCluster, setSelectedCluster] = useState<Cluster>(defaultCluster);
    const [clusterRetentionInfo, setClusterRetentionInfo] =
        useState<DecommissionedClusterRetentionInfo>(null);
    const [wizardStep, setWizardStep] = useState<WizardStep>('FORM');
    const [loadingCounter, setLoadingCounter] = useState(0);
    const [messageState, setMessageState] = useState<MessageState | null>(null);
    const [isBlocked, setIsBlocked] = useState(false);
    const [pollingCount, setPollingCount] = useState(0);
    const [pollingDelay, setPollingDelay] = useState<number | null>(null);
    const [submissionError, setSubmissionError] = useState<SubmissionError | null>(null);
    const [isDownloadingBundle, setIsDownloadingBundle] = useState(false);
    const [createUpgraderSA, setCreateUpgraderSA] = useState(true);

    function managerType(cluster: Partial<Cluster> | null): ClusterManagerType {
        return cluster?.helmConfig && cluster.managedBy === 'MANAGER_TYPE_UNKNOWN'
            ? 'MANAGER_TYPE_HELM_CHART'
            : (cluster?.managedBy ?? 'MANAGER_TYPE_UNKNOWN');
    }

    useEffect(
        () => {
            const clusterIdToRetrieve = clusterId;

            setLoadingCounter((prev) => prev + 1);
            getClusterDefaults()
                .then((clusterDefaults) => {
                    const {
                        mainImageRepository: mainImage,
                        collectorImageRepository: collectorImage,
                    } = clusterDefaults;

                    if (clusterIdToRetrieve === 'new') {
                        const updatedCluster = {
                            ...selectedCluster,
                            mainImage,
                            collectorImage,
                        };
                        setSelectedCluster(updatedCluster);
                    }
                })
                .catch(() => {
                    // TODO investigate how error affects ClusterLabelsConfigurationStatusSummary
                })
                .finally(() => {
                    setLoadingCounter((prev) => prev - 1);
                });

            if (clusterIdToRetrieve && clusterIdToRetrieve !== 'new') {
                setLoadingCounter((prev) => prev + 1);
                setMessageState(null);
                setIsBlocked(false);
                // don't want to cache or memoize, because we always want the latest real-time data
                fetchClusterWithRetentionInformation(clusterIdToRetrieve)
                    .then((clusterResponse) => {
                        const { cluster } = clusterResponse;
                        // cluster.managedBy = 'MANAGER_TYPE_MANUAL';
                        // TODO: refactor to use useReducer effect
                        setSelectedCluster(cluster);
                        setClusterRetentionInfo(clusterResponse.clusterRetentionInfo);

                        // stop polling after contact is established
                        if (selectedCluster?.healthStatus?.lastContact) {
                            setPollingDelay(null);
                        }

                        if (wizardStep === 'FORM') {
                            switch (managerType(cluster)) {
                                case 'MANAGER_TYPE_HELM_CHART':
                                    setMessageState({
                                        variant: 'warning',
                                        title: 'Helm-managed cluster',
                                        text: 'This is a Helm-managed cluster. The settings of Helm-managed clusters cannot be changed here, please ask your DevOps team to change the settings by updating the Helm values.',
                                    });
                                    break;
                                case 'MANAGER_TYPE_KUBERNETES_OPERATOR':
                                    setMessageState({
                                        variant: 'warning',
                                        title: 'Operator-managed cluster',
                                        text: 'This is an operator-managed cluster. The settings of operator-managed clusters cannot be changed here and must instead be changed by updating its SecuredCluster custom resource (CR).',
                                    });
                                    break;
                                default:
                                    break;
                            }
                        }
                    })
                    .catch((error) => {
                        setMessageState({
                            variant: 'danger',
                            title: 'error retrieving the configuration for the cluster',
                            text: getAxiosErrorMessage(error),
                        });
                        setIsBlocked(true);
                    })
                    .finally(() => {
                        setLoadingCounter((prev) => prev - 1);
                    });
                // TODO: When rolling out this feature the user should be informed somehow
                // in case this property could not be retrieved.
                // The default slimCollectorMode (false) is sufficient for now.
            }
        },
        // lint rule "exhaustive-deps" wants to add selectedCluster to change-detection
        // but we don't want to fetch while we're editing, so disabled that rule here
        // eslint-disable-next-line react-hooks/exhaustive-deps
        [clusterId, pollingCount]
    );

    // use a custom hook to set up polling, thanks Dan Abramov and Rob Stark
    useInterval(() => {
        setPollingCount(pollingCount + 1);
    }, pollingDelay);

    function onChange(path: string, value: boolean | number | string) {
        // path can be a dot path to property like: tolerationsConfig.disabled
        setSelectedCluster((oldClusterSettings) => {
            if (get(oldClusterSettings, path) === undefined) {
                return oldClusterSettings;
            }

            const newClusterSettings = cloneDeep(oldClusterSettings);
            set(newClusterSettings, path, value);
            return newClusterSettings;
        });
    }

    function onChangeAdmissionControllerEnforcementBehavior(value: boolean) {
        setSelectedCluster((oldClusterSettings) => {
            const newClusterSettings = cloneDeep(oldClusterSettings);
            // Special case because one selection for both values, starting in 4.9 release.
            set(newClusterSettings, 'dynamicConfig.admissionControllerConfig.enabled', value);
            set(
                newClusterSettings,
                'dynamicConfig.admissionControllerConfig.enforceOnUpdates',
                value
            );
            return newClusterSettings;
        });
    }

    /**
     * @param   {Event}  event  native JS Event object from an onChange event in an input
     *
     * @return  {nothing}       Side effect: change the corresponding property in selectedCluster
     */
    function onChangeLegacy(event) {
        // Functional update computes new state from old state to solve data race:
        // `admissionControllerEvents: false` overwritten by `type: "OPENSHIFT_CLUSTER"`
        // See guardedClusterTypeChange
        setSelectedCluster((oldClusterSettings) => {
            // event.target.name can be a dot path to property like:
            // dynamicConfig.admissionControllerConfig.timeoutSeconds
            if (get(oldClusterSettings, event.target.name) === undefined) {
                return oldClusterSettings;
            }

            const newClusterSettings = { ...oldClusterSettings };
            const newValue =
                event.target.type === 'checkbox' ? event.target.checked : event.target.value;
            set(newClusterSettings, event.target.name, newValue);
            return newClusterSettings;
        });
    }

    /*
     * Adapt preceding code for labels whose value is not from an input element.
     */
    function handleChangeLabels(labels) {
        setSelectedCluster((oldClusterSettings) => {
            return { ...oldClusterSettings, labels };
        });
    }

    function onNext() {
        if (wizardStep === 'FORM') {
            setMessageState(null);
            setSubmissionError(null);
            saveCluster(selectedCluster)
                .then((clusterResponse) => {
                    analyticsTrack(CLUSTER_CREATED);
                    const newId = clusterResponse.cluster.id;
                    const clusterWithId = { ...selectedCluster, id: newId };
                    setSelectedCluster(clusterWithId);
                    setClusterRetentionInfo(clusterResponse.clusterRetentionInfo);

                    setWizardStep('DEPLOYMENT');

                    if (!selectedCluster?.healthStatus?.lastContact) {
                        setPollingDelay(clusterDetailPollingInterval);
                    }
                })
                .catch((error) => {
                    setSubmissionError({
                        title: 'Unable to save changes',
                        text: getAxiosErrorMessage(error),
                    });
                });
        } else {
            navigate(clustersBasePath);
        }
    }

    function toggleSA() {
        setCreateUpgraderSA(!createUpgraderSA);
    }

    function onDownload() {
        setSubmissionError(null);
        setIsDownloadingBundle(true);
        if (selectedCluster?.id) {
            downloadClusterYaml(selectedCluster.id, createUpgraderSA)
                .catch((error) => {
                    setSubmissionError({
                        title: 'Unable to download file',
                        text: getAxiosErrorMessage(error),
                    });
                })
                .finally(() => {
                    setIsDownloadingBundle(false);
                });
        }
    }

    const selectedClusterName =
        selectedCluster?.name || 'Create secured cluster with legacy installation method';

    // @TODO: improve error handling when adding support for new clusters
    const isForm = wizardStep === 'FORM';

    return (
        <>
            <PageTitle title="Cluster" />
            <PageSection variant="light">
                <Breadcrumb>
                    <BreadcrumbItemLink to={clustersBasePath}>Clusters</BreadcrumbItemLink>
                    <BreadcrumbItem isActive>{selectedClusterName}</BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />
            <PageSection variant="light">
                <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsMd' }}>
                    <Flex
                        direction={{ default: 'row' }}
                        justifyContent={{ default: 'justifyContentSpaceBetween' }}
                    >
                        <Title headingLevel="h1">{selectedClusterName}</Title>
                        {hasWriteAccessForCluster && !isBlocked && (
                            <Button
                                variant={isForm ? 'secondary' : 'primary'}
                                onClick={onNext}
                                disabled={
                                    isForm && Object.keys(validate(selectedCluster)).length !== 0
                                }
                            >
                                {isForm ? 'Next' : 'Finish'}
                            </Button>
                        )}
                    </Flex>
                    {!!messageState && (
                        <Alert
                            variant={messageState.variant}
                            isInline
                            title={messageState.title}
                            component="p"
                        >
                            {messageState.text}
                        </Alert>
                    )}
                    {submissionError && (
                        <Alert
                            variant="danger"
                            isInline
                            title={submissionError.title}
                            component="p"
                        >
                            {submissionError.text}
                        </Alert>
                    )}
                    {!isBlocked &&
                        wizardStep === 'FORM' &&
                        (loadingCounter > 0 ? (
                            <Bullseye>
                                <Spinner />
                            </Bullseye>
                        ) : isAdmissionControllerConfigEnabled ? (
                            <ClusterLabelsConfigurationStatusSummary
                                centralVersion={metadata.version}
                                clusterRetentionInfo={clusterRetentionInfo}
                                selectedCluster={selectedCluster}
                                managerType={managerType(selectedCluster)}
                                handleChange={onChange}
                                handleChangeAdmissionControllerEnforcementBehavior={
                                    onChangeAdmissionControllerEnforcementBehavior
                                }
                                handleChangeLabels={handleChangeLabels}
                            />
                        ) : (
                            <ClusterEditFormLegacy
                                centralVersion={metadata.version}
                                clusterRetentionInfo={clusterRetentionInfo}
                                selectedCluster={selectedCluster}
                                managerType={managerType(selectedCluster)}
                                handleChange={onChangeLegacy}
                                handleChangeLabels={handleChangeLabels}
                            />
                        ))}
                    {!isBlocked && wizardStep === 'DEPLOYMENT' && (
                        <Flex
                            direction={{ default: 'column', lg: 'row' }}
                            flexWrap={{ default: 'nowrap' }}
                        >
                            <FlexItem flex={{ default: 'flex_1' }}>
                                <ClusterDeployment
                                    editing={!!selectedCluster}
                                    createUpgraderSA={createUpgraderSA}
                                    toggleSA={toggleSA}
                                    onFileDownload={onDownload}
                                    isDownloadingBundle={isDownloadingBundle}
                                    clusterCheckedIn={!!selectedCluster?.healthStatus?.lastContact}
                                    managerType={managerType(selectedCluster)}
                                />
                            </FlexItem>
                            {!!selectedCluster?.id && (
                                <FlexItem flex={{ default: 'flex_1' }}>
                                    <DownloadHelmValues
                                        clusterId={selectedCluster.id}
                                        description={
                                            selectedCluster?.helmConfig
                                                ? 'Download the required YAML to update your Helm values.'
                                                : 'To start managing this cluster with a Helm chart, you can download the cluster’s current configuration values in Helm format.'
                                        }
                                    />
                                </FlexItem>
                            )}
                        </Flex>
                    )}
                </Flex>
            </PageSection>
        </>
    );
}

export default ClusterPage;
