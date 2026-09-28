import { Alert, Flex, Label, LabelGroup, Title } from '@patternfly/react-core';

import type { VMDetail } from 'services/VirtualMachineService';
import { getDateTime } from 'utils/dateUtils';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import HeaderLoadingSkeleton from '../../components/HeaderLoadingSkeleton';
import { agentStatusDisplayMap, stateDisplayMap } from './virtualMachineUtils';

export type VirtualMachinePageHeaderProps = {
    virtualMachineDetail: VMDetail | undefined;
    isLoading: boolean;
    error: Error | undefined;
};

function VirtualMachinePageHeader({
    virtualMachineDetail,
    isLoading,
    error,
}: VirtualMachinePageHeaderProps) {
    if (isLoading) {
        return (
            <HeaderLoadingSkeleton
                nameScreenreaderText="Loading Virtual Machine name"
                metadataScreenreaderText="Loading Virtual Machine metadata"
            />
        );
    }

    if (error) {
        return (
            <Alert
                variant="danger"
                title="Unable to fetch virtual machine data"
                component="p"
                isInline
            >
                {getAxiosErrorMessage(error)}
            </Alert>
        );
    }

    if (!virtualMachineDetail) {
        return null;
    }

    const { name, clusterName, namespace, state, agentStatus, latestScan, guestOs } =
        virtualMachineDetail;

    return (
        <Flex direction={{ default: 'column' }} alignItems={{ default: 'alignItemsFlexStart' }}>
            <Title headingLevel="h1">{name}</Title>
            <LabelGroup numLabels={5}>
                <Label>
                    In: {clusterName}/{namespace}
                </Label>
                <Label>Status: {stateDisplayMap[state] ?? state}</Label>
                <Label>Agent: {agentStatusDisplayMap[agentStatus] ?? agentStatus}</Label>
                {latestScan?.scanTime && (
                    <Label>Scan time: {getDateTime(latestScan.scanTime)}</Label>
                )}
                {guestOs && <Label>Guest OS: {guestOs}</Label>}
            </LabelGroup>
        </Flex>
    );
}

export default VirtualMachinePageHeader;
