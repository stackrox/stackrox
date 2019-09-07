import React, { useEffect, useState } from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';
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
    newClusterDefault,
    parseUpgradeStatus,
    wizardSteps
} from './cluster.helpers';

function ClustersSidePanel({ selectedClusterId, setSelectedClusterId }) {
    const [selectedCluster, setSelectedCluster] = useState(newClusterDefault);
    const [wizardStep, setWizardStep] = useState(wizardSteps.FORM);
    const [messageState, setMessageState] = useState(null);
    const [pollingCount, setPollingCount] = useState(0);
    const [pollingDelay, setPollingDelay] = useState(null);

    function unselectCluster() {
        setSelectedClusterId('');
        setSelectedCluster(newClusterDefault);
        setMessageState(null);
        setWizardStep(wizardSteps.FORM);
    }

    useEffect(
        () => {
            if (selectedClusterId && selectedClusterId !== 'new') {
                setMessageState(null);
                // don't want to cache or memoize, because we always want the latest real-time data
                getClusterById(selectedClusterId)
                    .then(cluster => {
                        const upgradeStatus = parseUpgradeStatus(cluster);
                        const upgradeMessage = formatUpgradeMessage(upgradeStatus);
                        setMessageState(upgradeMessage);

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
                            message: 'We could not retrieve the cluster with that ID.'
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
            saveCluster(selectedCluster).then(() => {
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
            });
        } else {
            unselectCluster();
        }
    }

    function onDownload() {
        downloadClusterYaml(selectedClusterId);
    }

    /**
     * rendering section
     */
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

    return selectedClusterId ? (
        <Panel
            header={selectedClusterName}
            headerComponents={showPanelButtons ? panelButtons : <div />}
            className="w-full h-full absolute pin-r pin-t md:w-1/2 min-w-72 md:relative"
            onClose={unselectCluster}
        >
            {!!messageState && (
                <div className="m-4">
                    <Message type={messageState.type} message={messageState.message} />
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
    ) : null;
}

ClustersSidePanel.propTypes = {
    setSelectedClusterId: PropTypes.func.isRequired,
    selectedClusterId: PropTypes.string
};

ClustersSidePanel.defaultProps = {
    selectedClusterId: ''
};

export default ClustersSidePanel;
