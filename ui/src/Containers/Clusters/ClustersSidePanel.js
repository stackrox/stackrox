import React, { useEffect, useState } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { selectors } from 'reducers';
import { createStructuredSelector } from 'reselect';
import * as Icon from 'react-feather';
import cloneDeep from 'lodash/cloneDeep';
import get from 'lodash/get';
import set from 'lodash/set';

import Message from 'Components/Message';
import Panel from 'Components/Panel';
import PanelButton from 'Components/PanelButton';
import useInterval from 'hooks/useInterval';
import { getClusterById, saveCluster, downloadClusterYaml } from 'services/ClustersService';

import ClusterEditForm from './ClusterEditForm';
import ClusterDeployment from './ClusterDeployment';
import {
    clusterDetailPollingInterval,
    clusterTypeOptions,
    formatUpgradeMessage,
    getUpgradeStatusDetail,
    newClusterDefault,
    parseUpgradeStatus,
    wizardSteps
} from './cluster.helpers';
import CollapsibleCard from '../../Components/CollapsibleCard';

function ClustersSidePanel({ metadata, selectedClusterId, setSelectedClusterId, upgradeStatus }) {
    const defaultCluster = cloneDeep(newClusterDefault);
    const envAwareClusterDefault = {
        ...defaultCluster,
        mainImage: metadata.releaseBuild ? 'stackrox.io/main' : 'stackrox/main',
        collectorImage: metadata.releaseBuild
            ? 'collector.stackrox.io/collector'
            : 'stackrox/collector'
    };

    const [selectedCluster, setSelectedCluster] = useState(envAwareClusterDefault);
    const [wizardStep, setWizardStep] = useState(wizardSteps.FORM);
    const [messageState, setMessageState] = useState(null);
    const [pollingCount, setPollingCount] = useState(0);
    const [pollingDelay, setPollingDelay] = useState(null);
    const [submissionError, setSubmissionError] = useState('');

    const [createUpgraderSA, setCreateUpgraderSA] = useState(true);

    function unselectCluster() {
        setSubmissionError('');
        setSelectedClusterId('');
        setSelectedCluster(envAwareClusterDefault);
        setMessageState(null);
        setWizardStep(wizardSteps.FORM);
        setPollingDelay(null);
    }

    useEffect(
        () => {
            const clusterIdToRetrieve = selectedCluster.id || selectedClusterId;
            if (clusterIdToRetrieve && clusterIdToRetrieve !== 'new') {
                setMessageState(null);
                // don't want to cache or memoize, because we always want the latest real-time data
                getClusterById(clusterIdToRetrieve)
                    .then(cluster => {
                        // TODO: refactor to use useReducer effect
                        setSelectedCluster(cluster);

                        // stop polling after contact is established
                        if (
                            selectedCluster &&
                            selectedCluster.status &&
                            selectedCluster.status.lastContact
                        ) {
                            setPollingDelay(null);
                        }
                    })
                    .catch(() => {
                        setMessageState({
                            blocking: true,
                            type: 'error',
                            message: 'There was an error downloading the configuration files.'
                        });
                    });
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
     * naive implementation of form handler
     *  - replace with more robust system, probably react-final-form
     *
     * @param   {Event}  event  native JS Event object from an onChange event in an input
     *
     * @return  {nothing}       Side effect: change the corresponding property in selectedCluster
     */
    function onChange(event) {
        if (get(selectedCluster, event.target.name) !== undefined) {
            const newClusterSettings = { ...selectedCluster };
            const newValue =
                event.target.type === 'checkbox' ? event.target.checked : event.target.value;
            set(newClusterSettings, event.target.name, newValue);
            setSelectedCluster(newClusterSettings);
        }
    }

    function onClusterTypeChange(newClusterType) {
        if (
            clusterTypeOptions.find(value => value === newClusterType) !== undefined &&
            selectedCluster.type !== newClusterType
        ) {
            const newClusterSettings = { ...selectedCluster, type: newClusterType.value };

            setSelectedCluster(newClusterSettings);
        }
    }

    function onNext() {
        if (wizardStep === wizardSteps.FORM) {
            setSubmissionError('');
            saveCluster(selectedCluster)
                .then(response => {
                    const newId = response.response.result.cluster; // really is nested like this
                    const clusterWithId = { ...selectedCluster, id: newId };
                    setSelectedCluster(clusterWithId);

                    setWizardStep(wizardSteps.DEPLOYMENT);

                    if (
                        !(
                            selectedCluster &&
                            selectedCluster.status &&
                            selectedCluster.status.lastContact
                        )
                    ) {
                        setPollingDelay(clusterDetailPollingInterval);
                    }
                })
                .catch(error => {
                    const serverError = get(
                        error,
                        'response.data.message',
                        'An unknown error has occurred.'
                    );

                    setSubmissionError(serverError);
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
        downloadClusterYaml(selectedCluster.id, createUpgraderSA).catch(error => {
            const serverError = get(
                error,
                'response.data.message',
                'We could not download the configuration files.'
            );

            setSubmissionError(serverError);
        });
    }

    /**
     * rendering section
     */
    if (!selectedClusterId) {
        return null;
    }
    const showFormStyles =
        wizardStep === wizardSteps.FORM && !(messageState && messageState.blocking);
    const showDeploymentStyles =
        wizardStep === wizardSteps.DEPLOYMENT && !(messageState && messageState.blocking);
    const selectedClusterName = (selectedCluster && selectedCluster.name) || '';

    // @TODO: improve error handling when adding support for new clusters
    const panelButtons = (
        <PanelButton
            icon={
                showFormStyles ? (
                    <Icon.ArrowRight className="h-4 w-4" />
                ) : (
                    <Icon.Check className="h-4 w-4" />
                )
            }
            text={showFormStyles ? 'Next' : 'Finish'}
            className={`mr-2 btn ${showFormStyles ? 'btn-base' : 'btn-success'}`}
            onClick={onNext}
        />
    );

    const showPanelButtons = !messageState || !messageState.blocking;

    const parsedUpgradeStatus = parseUpgradeStatus(upgradeStatus);
    const upgradeStatusDetail = upgradeStatus && getUpgradeStatusDetail(upgradeStatus);
    const upgradeMessage =
        upgradeStatus && formatUpgradeMessage(parsedUpgradeStatus, upgradeStatusDetail);

    return (
        <Panel
            header={selectedClusterName}
            headerComponents={showPanelButtons ? panelButtons : <div />}
            bodyClassName="pt-4"
            className="bg-base-100 w-full h-full absolute pin-r pin-t md:w-1/2 min-w-72 md:relative"
            onClose={unselectCluster}
        >
            {!!messageState && (
                <div className="m-4">
                    <Message type={messageState.type} message={messageState.message} />
                </div>
            )}
            {!!upgradeMessage && (
                <div className="px-4 w-full">
                    <CollapsibleCard
                        title="Upgrade Status"
                        cardClassName="border border-base-400 mb-2"
                        titleClassName="border-b border-base-300 bg-primary-200 leading-normal cursor-pointer flex justify-between items-center hover:bg-primary-300 hover:border-primary-300"
                    >
                        <div className="m-4">
                            <Message type={upgradeMessage.type} message={upgradeMessage.message} />
                            {upgradeMessage.detail !== '' && (
                                <div className="mt-2 flex flex-col items-center">
                                    <div className="bg-base-200">
                                        <div className="whitespace-normal overflow-x-scroll">
                                            {upgradeMessage.detail}
                                        </div>
                                    </div>
                                </div>
                            )}
                        </div>
                    </CollapsibleCard>
                </div>
            )}
            {submissionError && submissionError.length > 0 && (
                <div className="w-full">
                    <div className="mb-4 mx-4">
                        <Message type="error" message={submissionError} />
                    </div>
                </div>
            )}
            {showFormStyles && (
                <ClusterEditForm
                    selectedCluster={selectedCluster}
                    handleChange={onChange}
                    onClusterTypeChange={onClusterTypeChange}
                />
            )}
            {showDeploymentStyles && (
                <ClusterDeployment
                    editing={!!selectedCluster}
                    createUpgraderSA={createUpgraderSA}
                    toggleSA={toggleSA}
                    onFileDownload={onDownload}
                    clusterCheckedIn={
                        !!(
                            selectedCluster &&
                            selectedCluster.status &&
                            selectedCluster.status.lastContact
                        )
                    }
                />
            )}
        </Panel>
    );
}

ClustersSidePanel.propTypes = {
    metadata: PropTypes.shape({ version: PropTypes.string, releaseBuild: PropTypes.bool })
        .isRequired,
    setSelectedClusterId: PropTypes.func.isRequired,
    selectedClusterId: PropTypes.string,
    upgradeStatus: PropTypes.shape({})
};

ClustersSidePanel.defaultProps = {
    selectedClusterId: '',
    upgradeStatus: null
};

const mapStateToProps = createStructuredSelector({
    metadata: selectors.getMetadata
});

export default connect(mapStateToProps)(ClustersSidePanel);
