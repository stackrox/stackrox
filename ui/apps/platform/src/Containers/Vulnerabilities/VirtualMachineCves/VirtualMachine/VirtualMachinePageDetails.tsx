import {
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Flex,
    PageSection,
} from '@patternfly/react-core';

import type { VMDetail } from 'services/VirtualMachineService';
import { getDateTime } from 'utils/dateUtils';

import ExpandableLabelSection from '../../components/ExpandableLabelSection';
import { agentStatusDisplayMap, stateDisplayMap } from './virtualMachineUtils';

function recordToLabelArray(record: Record<string, string>): { key: string; value: string }[] {
    return Object.entries(record).map(([key, value]) => ({ key, value }));
}

export type VirtualMachinePageDetailsProps = {
    virtualMachineDetail: VMDetail;
};

function VirtualMachinePageDetails({ virtualMachineDetail }: VirtualMachinePageDetailsProps) {
    const {
        state,
        agentStatus,
        guestOs,
        lastUpdated,
        latestScan,
        facts,
        vsockCid,
        labels,
        annotations,
    } = virtualMachineDetail;

    return (
        <PageSection isFilled padding={{ default: 'padding' }}>
            <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsXl' }}>
                <DescriptionList columnModifier={{ default: '1Col', lg: '2Col' }}>
                    <DescriptionListGroup>
                        <DescriptionListTerm>Status</DescriptionListTerm>
                        <DescriptionListDescription>
                            {stateDisplayMap[state] ?? state}
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                    <DescriptionListGroup>
                        <DescriptionListTerm>Agent Status</DescriptionListTerm>
                        <DescriptionListDescription>
                            {agentStatusDisplayMap[agentStatus] ?? agentStatus}
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                    <DescriptionListGroup>
                        <DescriptionListTerm>Operating System</DescriptionListTerm>
                        <DescriptionListDescription>{guestOs || '-'}</DescriptionListDescription>
                    </DescriptionListGroup>
                    {lastUpdated && (
                        <DescriptionListGroup>
                            <DescriptionListTerm>Last Updated</DescriptionListTerm>
                            <DescriptionListDescription>
                                {getDateTime(lastUpdated)}
                            </DescriptionListDescription>
                        </DescriptionListGroup>
                    )}
                    {latestScan && (
                        <>
                            {latestScan.scanTime && (
                                <DescriptionListGroup>
                                    <DescriptionListTerm>Last Scan Time</DescriptionListTerm>
                                    <DescriptionListDescription>
                                        {getDateTime(latestScan.scanTime)}
                                    </DescriptionListDescription>
                                </DescriptionListGroup>
                            )}
                            <DescriptionListGroup>
                                <DescriptionListTerm>Scan OS</DescriptionListTerm>
                                <DescriptionListDescription>
                                    {latestScan.scanOs || '-'}
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>Top CVSS</DescriptionListTerm>
                                <DescriptionListDescription>
                                    {latestScan.topCvss || '-'}
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                        </>
                    )}
                    <DescriptionListGroup>
                        <DescriptionListTerm>Node</DescriptionListTerm>
                        <DescriptionListDescription>
                            {facts.nodeName || '-'}
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                    <DescriptionListGroup>
                        <DescriptionListTerm>IP Addresses</DescriptionListTerm>
                        <DescriptionListDescription>
                            {facts.ipAddresses || '-'}
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                    <DescriptionListGroup>
                        <DescriptionListTerm>Pods</DescriptionListTerm>
                        <DescriptionListDescription>
                            {facts.activePods || '-'}
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                    <DescriptionListGroup>
                        <DescriptionListTerm>Boot Order</DescriptionListTerm>
                        <DescriptionListDescription>
                            {facts.bootOrder || '-'}
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                    <DescriptionListGroup>
                        <DescriptionListTerm>Vsock CID</DescriptionListTerm>
                        <DescriptionListDescription>{vsockCid || '-'}</DescriptionListDescription>
                    </DescriptionListGroup>
                </DescriptionList>
                <ExpandableLabelSection toggleText="Labels" labels={recordToLabelArray(labels)} />
                <ExpandableLabelSection
                    toggleText="Annotations"
                    labels={recordToLabelArray(annotations)}
                />
            </Flex>
        </PageSection>
    );
}

export default VirtualMachinePageDetails;
