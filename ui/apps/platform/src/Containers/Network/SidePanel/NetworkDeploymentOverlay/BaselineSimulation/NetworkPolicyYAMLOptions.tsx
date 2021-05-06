/* eslint-disable no-console */
import React, { ReactElement, useState } from 'react';
import { useSelector } from 'react-redux';
import { createSelector } from 'reselect';
import { Button, Flex, FlexItem, Tooltip } from '@patternfly/react-core';
import { UndoIcon, ShareSquareIcon, DownloadIcon } from '@patternfly/react-icons';

import { selectors } from 'reducers';
import { notifyNetworkPolicyModification } from 'services/NetworkService';
import download from 'utils/download';
import { FetchBaselineGeneratedNetworkPolicyResult } from './useFetchBaselineGeneratedNetworkPolicy';
import NotifiersModal from './NotifiersModal';

// TODO: decide if this way of overriding PatternFly vars is worth adopting
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
};

function NetworkPolicyYAMLOptions({ networkPolicy }: NetworkPolicyYAMLOptionsProps): ReactElement {
    const [isNotifiersModalOpen, setIsNotifiersModalOpen] = useState(false);
    const deploymentName = useSelector(selectedDeploymentNameSelector);
    const clusterId = useSelector(selectedClusterId);
    const hasYaml = !!networkPolicy?.modification?.applyYaml;

    function revertPolicy() {
        // TODO: work with Saif to display the undo policy in the graph
        // eslint-disable-next-line no-console
        console.log({ networkPolicy });
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
                            onClick={revertPolicy}
                            isDisabled={!hasYaml}
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
                            isDisabled={!hasYaml}
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
                            isDisabled={!hasYaml}
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
