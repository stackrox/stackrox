import React from 'react';
import { PageSection } from '@patternfly/react-core';
import EmptyStateTemplate from 'Components/EmptyStateTemplate';

export type VirtualMachinePageDetailsProps = {
    virtualMachineId: string;
};

function VirtualMachinePageDetails({ virtualMachineId }: VirtualMachinePageDetailsProps) {
    return (
        <PageSection variant="light" isFilled padding={{ default: 'padding' }}>
            <EmptyStateTemplate title="Virtual Machine Details" headingLevel="h2">
                Virtual machine details content will be implemented here for {virtualMachineId}.
            </EmptyStateTemplate>
        </PageSection>
    );
}

export default VirtualMachinePageDetails;
