import React from 'react';
import { PageSection } from '@patternfly/react-core';

import EmptyStateTemplate from 'Components/EmptyStateTemplate';

export type VirtualMachinePageVulnerabilitiesProps = {
    virtualMachineId: string;
};

function VirtualMachinePageVulnerabilities({
    virtualMachineId,
}: VirtualMachinePageVulnerabilitiesProps) {
    return (
        <PageSection variant="light" isFilled padding={{ default: 'padding' }}>
            <EmptyStateTemplate title="Virtual Machine Vulnerabilities" headingLevel="h2">
                Virtual machine vulnerabilities table will be implemented here for{' '}
                {virtualMachineId}.
            </EmptyStateTemplate>
        </PageSection>
    );
}

export default VirtualMachinePageVulnerabilities;
