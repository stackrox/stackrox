import React from 'react';
import { Flex, Title, LabelGroup, Label, Alert } from '@patternfly/react-core';

import DeveloperPreviewLabel from 'Components/PatternFly/DeveloperPreviewLabel';
import { VirtualMachine } from 'services/VirtualMachineService';
import { getDateTime } from 'utils/dateUtils';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import HeaderLoadingSkeleton from '../../components/HeaderLoadingSkeleton';

export type VirtualMachinePageHeaderProps = {
    virtualMachineData: VirtualMachine | undefined;
    isLoading: boolean;
    error: Error | undefined;
};

function VirtualMachinePageHeader({
    virtualMachineData,
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

    if (!virtualMachineData) {
        return null;
    }

    return (
        <Flex direction={{ default: 'column' }} alignItems={{ default: 'alignItemsFlexStart' }}>
            <Flex alignItems={{ default: 'alignItemsCenter' }}>
                <Title headingLevel="h1">{virtualMachineData.name}</Title>
                <DeveloperPreviewLabel />
            </Flex>
            <LabelGroup numLabels={5}>
                <Label>
                    In: {virtualMachineData.clusterName}/{virtualMachineData.namespace}
                </Label>
                {virtualMachineData.scan?.scanTime && (
                    <Label>Scan time: {getDateTime(virtualMachineData.scan.scanTime)}</Label>
                )}
            </LabelGroup>
        </Flex>
    );
}

export default VirtualMachinePageHeader;
