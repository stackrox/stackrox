import { WorkloadCveViewContext } from 'Containers/Vulnerabilities/WorkloadCves/WorkloadCveViewContext';

import { VulnerabilitiesOverviewContainer } from '../Components/VulnerabilitiesOverviewContainer';
import { useDefaultWorkloadCveViewContext } from '../hooks/useDefaultWorkloadCveViewContext';
import { useAnalyticsPageView } from '../hooks/useAnalyticsPageView';

export function AdministrationNamespaceSecurityTab() {
    useAnalyticsPageView();

    const context = useDefaultWorkloadCveViewContext();

    return (
        <WorkloadCveViewContext.Provider value={context}>
            <VulnerabilitiesOverviewContainer />
        </WorkloadCveViewContext.Provider>
    );
}
