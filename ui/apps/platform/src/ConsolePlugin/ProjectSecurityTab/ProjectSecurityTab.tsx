import React from 'react';
import { WorkloadCveViewContext } from 'Containers/Vulnerabilities/WorkloadCves/WorkloadCveViewContext';

import { VulnerabilitiesOverviewContainer } from '../Components/VulnerabilitiesOverviewContainer';
import { useDefaultWorkloadCveViewContext } from '../hooks/useDefaultWorkloadCveViewContext';

export function ProjectSecurityTab() {
    const context = useDefaultWorkloadCveViewContext();

    return (
        <WorkloadCveViewContext.Provider value={context}>
            <VulnerabilitiesOverviewContainer />
        </WorkloadCveViewContext.Provider>
    );
}
