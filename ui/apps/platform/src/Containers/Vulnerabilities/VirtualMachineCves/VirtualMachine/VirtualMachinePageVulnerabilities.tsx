import React, { useCallback } from 'react';
import { PageSection } from '@patternfly/react-core';

import useRestQuery from 'hooks/useRestQuery';
import { getVirtualMachine } from 'services/VirtualMachineService';
import { getTableUIState } from 'utils/getTableUIState';

import { getVirtualMachineCveTableData } from '../aggregateUtils';
import VirtualMachineVulnerabilitiesTable from './VirtualMachineVulnerabilitiesTable';

export type VirtualMachinePageVulnerabilitiesProps = {
    virtualMachineId: string;
};

function VirtualMachinePageVulnerabilities({
    virtualMachineId,
}: VirtualMachinePageVulnerabilitiesProps) {
    const fetchVirtualMachines = useCallback(
        () => getVirtualMachine(virtualMachineId),
        [virtualMachineId]
    );

    const { data, isLoading, error } = useRestQuery(fetchVirtualMachines);

    const virtualMachineTableData = getVirtualMachineCveTableData(data);

    const tableState = getTableUIState({
        isLoading,
        data: virtualMachineTableData,
        error,
        searchFilter: {},
    });

    return (
        <PageSection variant="light" isFilled padding={{ default: 'padding' }}>
            <VirtualMachineVulnerabilitiesTable tableState={tableState} />
        </PageSection>
    );
}

export default VirtualMachinePageVulnerabilities;
