import React, { useEffect, useState } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { selectors } from 'reducers';
import { createStructuredSelector } from 'reselect';
import { ArrowRight, Check } from 'react-feather';
import cloneDeep from 'lodash/cloneDeep';
import get from 'lodash/get';
import set from 'lodash/set';

import Message from 'Components/Message';
import Panel from 'Components/Panel';
import PanelButton from 'Components/PanelButton';
import SidePanelAnimatedDiv from 'Components/animations/SidePanelAnimatedDiv';
import { useTheme } from 'Containers/ThemeProvider';
import useInterval from 'hooks/useInterval';
import {
    getClusterById,
    saveCluster,
    downloadClusterYaml,
    fetchKernelSupportAvailable,
} from 'services/ClustersService';

import ClusterEditForm from './ClusterEditForm';
import ClusterDeployment from './ClusterDeployment';
import {
    clusterDetailPollingInterval,
    newClusterDefault,
    wizardSteps,
    centralEnvDefault,
} from './cluster.helpers';

function fetchCentralEnv() {
    return fetchKernelSupportAvailable().then((kernelSupportAvailable) => {
        return {
            kernelSupportAvailable,
            successfullyFetched: true,
        };
    });
}

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

function ClustersSidePanel({ metadata, selectedClusterId, setSelectedClusterId }) {
    const defaultCluster = cloneDeep(newClusterDefault);
    const envAwareClusterDefault = {
        ...defaultCluster,
        mainImage: metadata.releaseBuild ? 'stackrox.io/main' : 'stackrox/main',
        collectorImage: metadata.releaseBuild
            ? 'collector.stackrox.io/collector'
            : 'stackrox/collector',
    };

    const { isDarkMode } = useTheme();
    const [selectedCluster, setSelectedCluster] = useState(envAwareClusterDefault);
    const [centralEnv, setCentralEnv] = useState(centralEnvDefault);
    const [wizardStep, setWizardStep] = useState(wizardSteps.FORM);
    const [loadingCounter, setLoadingCounter] = useState(0);
    const [messageState, setMessageState] = useState(null);
    const [isBlocked, setIsBlocked] = useState(false);
    const [pollingCount, setPollingCount] = useState(0);
    const [pollingDelay, setPollingDelay] = useState(null);
    const [submissionError, setSubmissionError] = useState('');

    const [createUpgraderSA, setCreateUpgraderSA] = useState(true);

    function unselectCluster() {
        setSubmissionError('');
        setSelectedClusterId('');
        setSelectedCluster(envAwareClusterDefault);
        setMessageState(null);
        setIsBlocked(false);
        setWizardStep(wizardSteps.FORM);
        setPollingDelay(null);
    }

    useEffect(
        () => {
            const clusterIdToRetrieve = selectedClusterId;

            setLoadingCounter((prev) => prev + 1);
            fetchCentralEnv()
                .then((freshCentralEnv) => {
                    setCentralEnv(freshCentralEnv);
                    if (clusterIdToRetrieve === 'new') {
                        const updatedCluster = {
                            ...selectedCluster,
                            slimCollector: freshCentralEnv.kernelSupportAvailable,
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
                getClusterById(clusterIdToRetrieve)
                    .then((cluster) => {
                        // TODO: refactor to use useReducer effect
                        setSelectedCluster(cluster);

                        // stop polling after contact is established
                        if (selectedCluster?.healthStatus?.lastContact) {
                            setPollingDelay(null);
                        }
                    })
                    .catch(() => {
                        setMessageState({
                            type: 'error',
                            message: 'There was an error downloading the configuration files.',
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
        // event.target.name can be a dot path to property like:
        // dynamicConfig.admissionControllerConfig.timeoutSeconds
        if (get(selectedCluster, event.target.name) !== undefined) {
            const newClusterSettings = { ...selectedCluster };
            const newValue =
                event.target.type === 'checkbox' ? event.target.checked : event.target.value;
            set(newClusterSettings, event.target.name, newValue);
            setSelectedCluster(newClusterSettings);
        }
    }

    function onNext() {
        if (wizardStep === wizardSteps.FORM) {
            setSubmissionError('');
            saveCluster(selectedCluster)
                .then((response) => {
                    const newId = response.response.result.cluster; // really is nested like this
                    const clusterWithId = { ...selectedCluster, id: newId };
                    setSelectedCluster(clusterWithId);

                    setWizardStep(wizardSteps.DEPLOYMENT);

                    if (!selectedCluster?.healthStatus?.lastContact) {
                        setPollingDelay(clusterDetailPollingInterval);
                    }
                })
                .catch((error) => {
                    setSubmissionError(
                        error?.response?.data?.message || 'An unknown error has occurred.'
                    );
                });
        } else {
            unselectCluster();
        }
    }

    function toggleSA() {
        setCreateUpgraderSA(!createUpgraderSA);
    }

    function onDownload() {
        setSubmissionError('');
        downloadClusterYaml(selectedCluster.id, createUpgraderSA).catch((error) => {
            setSubmissionError(
                error?.response?.data?.message || 'We could not download the configuration files.'
            );
        });
    }

    /**
     * rendering section
     */
    if (!selectedClusterId) {
        return null;
    }

    const selectedClusterName = (selectedCluster && selectedCluster.name) || '';

    // @TODO: improve error handling when adding support for new clusters
    const isForm = wizardStep === wizardSteps.FORM;
    const iconClassName = 'h-4 w-4';

    const panelButtons = isBlocked ? (
        <div />
    ) : (
        <PanelButton
            icon={
                isForm ? (
                    <ArrowRight className={iconClassName} />
                ) : (
                    <Check className={iconClassName} />
                )
            }
            className={`mr-2 btn ${isForm ? 'btn-base' : 'btn-success'}`}
            onClick={onNext}
            disabled={isForm && Object.keys(validate(selectedCluster)).length !== 0}
            tooltip={isForm ? 'Next' : 'Finish'}
        >
            {isForm ? 'Next' : 'Finish'}
        </PanelButton>
    );

    return (
        <SidePanelAnimatedDiv isOpen={!!selectedClusterId}>
            <div
                className={`w-full h-full bg-base-100 rounded-tl-lg shadow-sidepanel ${
                    !isDarkMode ? '' : 'border-l border-base-400'
                }`}
            >
                <Panel
                    id="clusters-side-panel"
                    header={selectedClusterName}
                    headerComponents={panelButtons}
                    headerClassName={`flex w-full h-14 overflow-y-hidden bg-primary-200 rounded-tl-lg border-b ${
                        !isDarkMode ? 'border-base-100' : 'border-primary-400'
                    }`}
                    bodyClassName={isDarkMode ? 'bg-base-0' : 'bg-primary-200'}
                    onClose={unselectCluster}
                >
                    {!!messageState && (
                        <div className="m-4">
                            <Message type={messageState.type} message={messageState.message} />
                        </div>
                    )}
                    {submissionError && submissionError.length > 0 && (
                        <div className="w-full">
                            <div className="mb-4 mx-4">
                                <Message type="error" message={submissionError} />
                            </div>
                        </div>
                    )}
                    {!isBlocked && wizardStep === wizardSteps.FORM && (
                        <ClusterEditForm
                            centralEnv={centralEnv}
                            centralVersion={metadata.version}
                            selectedCluster={selectedCluster}
                            handleChange={onChange}
                            isLoading={loadingCounter > 0}
                        />
                    )}
                    {!isBlocked && wizardStep === wizardSteps.DEPLOYMENT && (
                        <ClusterDeployment
                            editing={!!selectedCluster}
                            createUpgraderSA={createUpgraderSA}
                            toggleSA={toggleSA}
                            onFileDownload={onDownload}
                            clusterCheckedIn={!!selectedCluster?.healthStatus?.lastContact}
                        />
                    )}
                </Panel>
            </div>
        </SidePanelAnimatedDiv>
    );
}

ClustersSidePanel.propTypes = {
    metadata: PropTypes.shape({ version: PropTypes.string, releaseBuild: PropTypes.bool })
        .isRequired,
    setSelectedClusterId: PropTypes.func.isRequired,
    selectedClusterId: PropTypes.string,
};

ClustersSidePanel.defaultProps = {
    selectedClusterId: '',
};

const mapStateToProps = createStructuredSelector({
    metadata: selectors.getMetadata,
});

export default connect(mapStateToProps)(ClustersSidePanel);
