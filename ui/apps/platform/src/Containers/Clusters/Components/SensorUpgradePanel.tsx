import React from 'react';
import {
    DescriptionList,
    DescriptionListGroup,
    DescriptionListTerm,
    DescriptionListDescription,
    Divider,
    Panel,
    PanelHeader,
    PanelMain,
    PanelMainBody,
    Button,
} from '@patternfly/react-core';
import upperFirst from 'lodash/upperFirst';

import { findUpgradeState } from '../cluster.helpers';
import SensorUpgrade from './SensorUpgrade';
import { SensorUpgradeStatus } from '../clusterTypes';

export type SensorUpgradePanelProps = {
    actionProps?: {
        clusterId: string;
        upgradeSingleCluster: (clusterId: string) => void;
    };
    centralVersion: string;
    sensorVersion: string;
    upgradeStatus: SensorUpgradeStatus;
};

function SensorUpgradePanel({
    centralVersion,
    sensorVersion,
    upgradeStatus,
    actionProps,
}: SensorUpgradePanelProps) {
    const { upgradabilityStatusReason, mostRecentProcess } = upgradeStatus;

    const upgradeState = findUpgradeState(upgradeStatus);

    const statusReason =
        upgradeState?.type === 'failure'
            ? (mostRecentProcess?.progress?.upgradeStatusDetail ?? '')
            : upperFirst(upgradabilityStatusReason);

    return (
        <Panel variant="bordered">
            <PanelHeader>Sensor upgrade status</PanelHeader>
            <Divider />
            <PanelMain>
                <PanelMainBody>
                    <DescriptionList>
                        <DescriptionListGroup>
                            <DescriptionListTerm>Status</DescriptionListTerm>
                            <DescriptionListDescription>
                                <SensorUpgrade upgradeStatus={upgradeStatus} />
                            </DescriptionListDescription>
                        </DescriptionListGroup>
                        <DescriptionListGroup>
                            <DescriptionListTerm>Status reasoning</DescriptionListTerm>
                            <DescriptionListDescription>
                                {statusReason || 'Unknown'}
                            </DescriptionListDescription>
                        </DescriptionListGroup>
                        {actionProps && (
                            <DescriptionListGroup>
                                <DescriptionListTerm>Available action</DescriptionListTerm>
                                <DescriptionListDescription>
                                    {upgradeState?.actionText ? (
                                        <Button
                                            isInline
                                            onClick={() => {
                                                actionProps.upgradeSingleCluster(
                                                    actionProps.clusterId
                                                );
                                            }}
                                            variant="secondary"
                                        >
                                            {upgradeState.actionText}
                                        </Button>
                                    ) : (
                                        'None'
                                    )}
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                        )}
                        <DescriptionListGroup>
                            <DescriptionListTerm>Sensor version</DescriptionListTerm>
                            <DescriptionListDescription>{sensorVersion}</DescriptionListDescription>
                        </DescriptionListGroup>
                        <DescriptionListGroup>
                            <DescriptionListTerm>Central version</DescriptionListTerm>
                            <DescriptionListDescription>
                                {centralVersion}
                            </DescriptionListDescription>
                        </DescriptionListGroup>
                    </DescriptionList>
                </PanelMainBody>
            </PanelMain>
        </Panel>
    );
}

export default SensorUpgradePanel;
