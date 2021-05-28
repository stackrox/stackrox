import React, { ReactElement, useState } from 'react';
import { useSelector, useDispatch } from 'react-redux';
import { createSelector } from 'reselect';
import { Button, Flex, FlexItem, Tooltip } from '@patternfly/react-core';
import { UndoIcon, ShareSquareIcon, DownloadIcon } from '@patternfly/react-icons';

import { actions as baselineSimulationActions } from 'reducers/network/baselineSimulation';
import { selectors } from 'reducers';
import { notifyNetworkPolicyModification } from 'services/NetworkService';
import download from 'utils/download';
import { FetchBaselineGeneratedNetworkPolicyResult } from './useFetchBaselineGeneratedNetworkPolicy';
import NotifiersModal from './NotifiersModal';

import './NetworkPolicyYAMLOptions.css';

const revertButtonLabel = 'Revert to most recently applied YAML.';
const shareButtonLabel = 'Share YAML.';
const downloadButtonLabel = 'Download YAML.';

type Deployment = {
    name: string;
};

const selectedDeploymentNameSelector = createSelector(
    selectors.getSelectedNode,
    (selectedDeployment: Deployment) => selectedDeployment?.name || ''
);

const selectedClusterId = createSelector(
    selectors.getSelectedNetworkClusterId,
    (selectedNetworkClusterId: string) => selectedNetworkClusterId
);

export type NetworkPolicyYAMLOptionsProps = {
    networkPolicy: FetchBaselineGeneratedNetworkPolicyResult['data'];
    undoAvailable: boolean;
    isUndoOn: boolean;
};

function NetworkPolicyYAMLOptions({
    networkPolicy,
    undoAvailable,
    isUndoOn,
}: NetworkPolicyYAMLOptionsProps): ReactElement {
    const [isNotifiersModalOpen, setIsNotifiersModalOpen] = useState(false);
    const deploymentName = useSelector(selectedDeploymentNameSelector);
    const clusterId = useSelector(selectedClusterId);
    const dispatch = useDispatch();

    const hasYaml = !!networkPolicy?.modification?.applyYaml;

    function toggleUndoPreview() {
        const newState = !isUndoOn;
        dispatch(baselineSimulationActions.toggleUndoPreview(newState));
    }

    function toggleNotifiersModal() {
        setIsNotifiersModalOpen(!isNotifiersModalOpen);
    }

    function sharePolicy(selectedNotifiers: string[]): Promise<never | Record<string, unknown>> {
        return notifyNetworkPolicyModification(
            clusterId,
            selectedNotifiers,
            networkPolicy?.modification || ''
        ) as Promise<Record<string, unknown>>; // type assertion necessary because Redux action is not yet typed
    }

    function downloadYamlFile() {
        if (hasYaml) {
            const currentDateString = new Date().toISOString();
            download(
                `${deploymentName}-network-policy-${currentDateString}.yaml`,
                networkPolicy?.modification?.applyYaml,
                'yaml'
            );
        }
    }

    return (
        <Flex>
            <FlexItem alignSelf={{ default: 'alignSelfCenter' }}>
                <span id="network-policy-yaml-options-label">YAML Options</span>
            </FlexItem>
            <Flex aria-labelledby="network-policy-yaml-options-label">
                <FlexItem>
                    <Tooltip content={<span>{revertButtonLabel}</span>}>
                        <Button
                            variant="tertiary"
                            isSmall
                            aria-label={revertButtonLabel}
                            onClick={toggleUndoPreview}
                            isDisabled={isUndoOn || !undoAvailable}
                        >
                            <UndoIcon />
                        </Button>
                    </Tooltip>
                </FlexItem>
                <FlexItem>
                    <Tooltip content={<span>{shareButtonLabel}</span>}>
                        <Button
                            variant="tertiary"
                            isSmall
                            aria-label={shareButtonLabel}
                            onClick={toggleNotifiersModal}
                            isDisabled={!hasYaml || isUndoOn}
                        >
                            <ShareSquareIcon />
                        </Button>
                    </Tooltip>
                    <NotifiersModal
                        closeModal={toggleNotifiersModal}
                        sharePolicy={sharePolicy}
                        isModalOpen={isNotifiersModalOpen}
                    />
                </FlexItem>
                <FlexItem>
                    <Tooltip content={<span>{downloadButtonLabel}</span>}>
                        <Button
                            variant="tertiary"
                            isSmall
                            aria-label={downloadButtonLabel}
                            onClick={downloadYamlFile}
                            isDisabled={!hasYaml || isUndoOn}
                        >
                            <DownloadIcon />
                        </Button>
                    </Tooltip>
                </FlexItem>
            </Flex>
        </Flex>
    );
}

export default NetworkPolicyYAMLOptions;
