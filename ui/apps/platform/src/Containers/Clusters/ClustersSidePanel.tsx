import React, { useEffect, useState } from 'react';
import PropTypes from 'prop-types';
import { Alert, Button } from '@patternfly/react-core';
import cloneDeep from 'lodash/cloneDeep';
import get from 'lodash/get';
import set from 'lodash/set';

import CloseButton from 'Components/CloseButton';
import { PanelNew, PanelBody, PanelHead, PanelHeadEnd, PanelTitle } from 'Components/Panel';
import SidePanelAnimatedArea from 'Components/animations/SidePanelAnimatedArea';
import { useTheme } from 'Containers/ThemeProvider';
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
import useAnalytics, { CLUSTER_CREATED } from 'hooks/useAnalytics';

import ClusterEditForm from './ClusterEditForm';
import ClusterDeployment from './ClusterDeployment';
import DownloadHelmValues from './DownloadHelmValues';
import {
    clusterDetailPollingInterval,
    newClusterDefault,
    centralEnvDefault,
} from './cluster.helpers';
import { CentralEnv } from './clusterTypes'; // augmented with successfullyFetched

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

function ClustersSidePanel({ selectedClusterId, setSelectedClusterId }) {
    const { hasReadWriteAccess } = usePermissions();
    const hasWriteAccessForCluster = hasReadWriteAccess('Cluster');

    const metadata = useMetadata();
    const { analyticsTrack } = useAnalytics();

    const defaultCluster = cloneDeep(newClusterDefault) as unknown as Cluster;

    const { isDarkMode } = useTheme();
    const [selectedCluster, setSelectedCluster] = useState<Cluster>(defaultCluster);
    const [clusterRetentionInfo, setClusterRetentionInfo] =
        useState<DecommissionedClusterRetentionInfo>(null);
    const [centralEnv, setCentralEnv] = useState<CentralEnv>(centralEnvDefault);
    const [wizardStep, setWizardStep] = useState<WizardStep>('FORM');
    const [loadingCounter, setLoadingCounter] = useState(0);
    const [messageState, setMessageState] = useState<MessageState | null>(null);
    const [isBlocked, setIsBlocked] = useState(false);
    const [pollingCount, setPollingCount] = useState(0);
    const [pollingDelay, setPollingDelay] = useState<number | null>(null);
    const [submissionError, setSubmissionError] = useState<SubmissionError | null>(null);
    const [isDownloadingBundle, setIsDownloadingBundle] = useState(false);
    const [createUpgraderSA, setCreateUpgraderSA] = useState(true);

    function unselectCluster() {
        setSubmissionError(null);
        setSelectedClusterId('');
        setSelectedCluster(defaultCluster);
        setMessageState(null);
        setIsBlocked(false);
        setWizardStep('FORM');
        setPollingDelay(null);
    }

    function managerType(cluster: Partial<Cluster> | null): ClusterManagerType {
        return cluster?.helmConfig && cluster.managedBy === 'MANAGER_TYPE_UNKNOWN'
            ? 'MANAGER_TYPE_HELM_CHART'
            : cluster?.managedBy ?? 'MANAGER_TYPE_UNKNOWN';
    }

    useEffect(
        () => {
            const clusterIdToRetrieve = selectedClusterId;

            setLoadingCounter((prev) => prev + 1);
            getClusterDefaults()
                .then((clusterDefaults) => {
                    const {
                        mainImageRepository: mainImage,
                        collectorImageRepository: collectorImage,
                        kernelSupportAvailable,
                    } = clusterDefaults;

                    // TODO add catch block to include error message as a property of the object.
                    setCentralEnv({
                        kernelSupportAvailable,
                        successfullyFetched: true,
                    });

                    if (clusterIdToRetrieve === 'new') {
                        const updatedCluster = {
                            ...selectedCluster,
                            mainImage,
                            collectorImage,
                            slimCollector: kernelSupportAvailable,
                        };
                        setSelectedCluster(updatedCluster);
                    }
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
                        // eslint-disable-next-line no-param-reassign
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
        [selectedClusterId, pollingCount]
    );

    // use a custom hook to set up polling, thanks Dan Abramov and Rob Stark
    useInterval(() => {
        setPollingCount(pollingCount + 1);
    }, pollingDelay);

    /**
     * @param   {Event}  event  native JS Event object from an onChange event in an input
     *
     * @return  {nothing}       Side effect: change the corresponding property in selectedCluster
     */
    function onChange(event) {
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
            unselectCluster();
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

    /**
     * rendering section
     */
    if (!selectedClusterId) {
        return null;
    }

    const selectedClusterName = (selectedCluster && selectedCluster.name) || '';

    // @TODO: improve error handling when adding support for new clusters
    const isForm = wizardStep === 'FORM';

    const panelButtons =
        !hasWriteAccessForCluster || isBlocked ? (
            <div />
        ) : (
            <Button
                variant={isForm ? 'secondary' : 'primary'}
                isSmall
                className="pf-u-mr-md"
                onClick={onNext}
                disabled={isForm && Object.keys(validate(selectedCluster)).length !== 0}
            >
                {isForm ? 'Next' : 'Finish'}
            </Button>
        );

    return (
        <SidePanelAnimatedArea isDarkMode={isDarkMode} isOpen={!!selectedClusterId}>
            <PanelNew testid="clusters-side-panel">
                <PanelHead>
                    <PanelTitle testid="clusters-side-panel-header" text={selectedClusterName} />
                    <PanelHeadEnd>
                        {panelButtons}
                        <CloseButton
                            onClose={unselectCluster}
                            className="border-base-400 border-l"
                        />
                    </PanelHeadEnd>
                </PanelHead>
                <PanelBody>
                    {!!messageState && (
                        <div className="m-4">
                            <Alert
                                variant={messageState.variant}
                                isInline
                                title={messageState.title}
                            >
                                {messageState.text}
                            </Alert>
                        </div>
                    )}
                    {submissionError && (
                        <div className="w-full">
                            <div className="mb-4 mx-4">
                                <Alert type="danger" isInline title={submissionError.title}>
                                    {submissionError.text}
                                </Alert>
                            </div>
                        </div>
                    )}
                    {!isBlocked && wizardStep === 'FORM' && (
                        <ClusterEditForm
                            centralEnv={centralEnv}
                            centralVersion={metadata.version}
                            clusterRetentionInfo={clusterRetentionInfo}
                            selectedCluster={selectedCluster}
                            managerType={managerType(selectedCluster)}
                            handleChange={onChange}
                            handleChangeLabels={handleChangeLabels}
                            isLoading={loadingCounter > 0}
                        />
                    )}
                    {!isBlocked && wizardStep === 'DEPLOYMENT' && (
                        <div className="flex flex-col md:flex-row p-4">
                            <ClusterDeployment
                                editing={!!selectedCluster}
                                createUpgraderSA={createUpgraderSA}
                                toggleSA={toggleSA}
                                onFileDownload={onDownload}
                                isDownloadingBundle={isDownloadingBundle}
                                clusterCheckedIn={!!selectedCluster?.healthStatus?.lastContact}
                                managerType={managerType(selectedCluster)}
                            />
                            {!!selectedCluster?.id && (
                                <DownloadHelmValues
                                    clusterId={selectedCluster.id}
                                    description={
                                        selectedCluster?.helmConfig
                                            ? 'Download the required YAML to update your Helm values.'
                                            : 'To start managing this cluster with a Helm chart, you can download the cluster’s current configuration values in Helm format.'
                                    }
                                />
                            )}
                        </div>
                    )}
                </PanelBody>
            </PanelNew>
        </SidePanelAnimatedArea>
    );
}

ClustersSidePanel.propTypes = {
    setSelectedClusterId: PropTypes.func.isRequired,
    selectedClusterId: PropTypes.string,
};

ClustersSidePanel.defaultProps = {
    selectedClusterId: '',
};

export default ClustersSidePanel;
