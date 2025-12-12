import { PageSection, Title } from '@patternfly/react-core';
import {
    DocumentTitle,
    NamespaceBar,
    useActiveNamespace,
} from '@openshift-console/dynamic-plugin-sdk';

import { namespaceSearchFilterConfig } from 'Containers/Vulnerabilities/searchFilterConfig';
import { WorkloadCveViewContext } from 'Containers/Vulnerabilities/WorkloadCves/WorkloadCveViewContext';
import useURLSearch from 'hooks/useURLSearch';
import { deleteKeysCaseInsensitive } from 'utils/searchUtils';

import { VulnerabilitiesOverviewContainer } from '../Components/VulnerabilitiesOverviewContainer';
import { useDefaultWorkloadCveViewContext } from '../hooks/useDefaultWorkloadCveViewContext';
import { useNamespaceScope } from '../ScopeContext';

export function SecurityVulnerabilitiesPage() {
    const [activeNamespace] = useActiveNamespace();
    // Set namespace scope for API requests
    useNamespaceScope(activeNamespace);
    const { searchFilter, setSearchFilter } = useURLSearch();
    const context = useDefaultWorkloadCveViewContext();

    return (
        <WorkloadCveViewContext.Provider value={context}>
            <DocumentTitle>Workload vulnerabilities</DocumentTitle>
            <NamespaceBar
                // Force clear Namespace filter when the user changes the namespace via the NamespaceBar
                onNamespaceChange={() => {
                    const namespaceAttributes = namespaceSearchFilterConfig.attributes;
                    const keysToDelete = namespaceAttributes.map(({ searchTerm }) => searchTerm);
                    const result = deleteKeysCaseInsensitive(searchFilter, keysToDelete);
                    setSearchFilter(result);
                }}
            />
            <PageSection variant="light">
                <Title headingLevel="h1">Workload vulnerabilities</Title>
            </PageSection>
            <PageSection variant="light">
                <VulnerabilitiesOverviewContainer />
            </PageSection>
        </WorkloadCveViewContext.Provider>
    );
}
