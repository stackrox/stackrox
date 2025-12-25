import { useActiveNamespace } from '@openshift-console/dynamic-plugin-sdk';

import { WorkloadCveViewContext } from 'Containers/Vulnerabilities/WorkloadCves/WorkloadCveViewContext';

import { VulnerabilitiesOverviewContainer } from '../Components/VulnerabilitiesOverviewContainer';
import { useDefaultWorkloadCveViewContext } from '../hooks/useDefaultWorkloadCveViewContext';
import { useNamespaceScope } from '../ScopeContext';

export function ProjectSecurityTab() {
    const [activeNamespace] = useActiveNamespace();
    // Set namespace scope for API requests
    useNamespaceScope(activeNamespace);
    const context = useDefaultWorkloadCveViewContext();

    return (
        <WorkloadCveViewContext.Provider value={context}>
            <VulnerabilitiesOverviewContainer />
        </WorkloadCveViewContext.Provider>
    );
}
