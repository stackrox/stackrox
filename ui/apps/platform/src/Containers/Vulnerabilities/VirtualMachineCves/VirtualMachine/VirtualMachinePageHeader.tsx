import { Alert, Flex, Label, LabelGroup, Title } from '@patternfly/react-core';

import TechnologyPreviewLabel from 'Components/PatternFly/PreviewLabel/TechnologyPreviewLabel';
import type { VirtualMachine } from 'services/VirtualMachineService';
import { getDateTime } from 'utils/dateUtils';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import HeaderLoadingSkeleton from '../../components/HeaderLoadingSkeleton';

export type VirtualMachinePageHeaderProps = {
    virtualMachine: VirtualMachine | undefined;
    isLoading: boolean;
    error: Error | undefined;
};

function VirtualMachinePageHeader({
    virtualMachine,
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

    if (!virtualMachine) {
        return null;
    }

    return (
        <Flex direction={{ default: 'column' }} alignItems={{ default: 'alignItemsFlexStart' }}>
            <Flex alignItems={{ default: 'alignItemsCenter' }}>
                <Title headingLevel="h1">{virtualMachine.name}</Title>
                <TechnologyPreviewLabel />
            </Flex>
            <LabelGroup numLabels={5}>
                <Label>
                    In: {virtualMachine.clusterName}/{virtualMachine.namespace}
                </Label>
                {virtualMachine.scan?.scanTime && (
                    <Label>Scan time: {getDateTime(virtualMachine.scan.scanTime)}</Label>
                )}
            </LabelGroup>
        </Flex>
    );
}

export default VirtualMachinePageHeader;
