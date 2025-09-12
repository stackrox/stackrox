import React from 'react';
import { NamespaceBar, useActiveNamespace } from '@openshift-console/dynamic-plugin-sdk';

import { ALL_NAMESPACES_KEY } from 'ConsolePlugin/constants';
import useURLSearch from 'hooks/useURLSearch';
import ImagePage from 'Containers/Vulnerabilities/WorkloadCves/Image/ImagePage';
import { useDefaultWorkloadCveViewContext } from 'ConsolePlugin/hooks/useDefaultWorkloadCveViewContext';
import { WorkloadCveViewContext } from 'Containers/Vulnerabilities/WorkloadCves/WorkloadCveViewContext';
import { hideColumnIf } from 'hooks/useManagedColumns';

export function ImageDetailPage() {
    const { searchFilter, setSearchFilter } = useURLSearch();
    const context = useDefaultWorkloadCveViewContext();
    const [activeNamespace] = useActiveNamespace();

    return (
        <WorkloadCveViewContext.Provider value={context}>
            <NamespaceBar
                // Force clear Namespace filter when the user changes the namespace via the NamespaceBar
                onNamespaceChange={() => setSearchFilter({ ...searchFilter, Namespace: [] })}
            />
            <ImagePage
                showVulnerabilityStateTabs={false}
                vulnerabilityState="OBSERVED"
                deploymentResourceColumnOverrides={{
                    cluster: hideColumnIf(true),
                    namespace: hideColumnIf(activeNamespace !== ALL_NAMESPACES_KEY),
                }}
            />
        </WorkloadCveViewContext.Provider>
    );
}
