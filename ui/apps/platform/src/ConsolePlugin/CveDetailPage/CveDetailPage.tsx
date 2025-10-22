import React from 'react';
import { NamespaceBar, useActiveNamespace } from '@openshift-console/dynamic-plugin-sdk';

import useURLSearch from 'hooks/useURLSearch';
import { hideColumnIf } from 'hooks/useManagedColumns';
import { ALL_NAMESPACES_KEY } from 'ConsolePlugin/constants';
import { useDefaultWorkloadCveViewContext } from 'ConsolePlugin/hooks/useDefaultWorkloadCveViewContext';
import { WorkloadCveViewContext } from 'Containers/Vulnerabilities/WorkloadCves/WorkloadCveViewContext';
import {
    deploymentSearchFilterConfig,
    imageComponentSearchFilterConfig,
    imageSearchFilterConfig,
    namespaceSearchFilterConfig,
} from 'Containers/Vulnerabilities/searchFilterConfig';
import ImageCvePage from 'Containers/Vulnerabilities/WorkloadCves/ImageCve/ImageCvePage';

export function CveDetailPage() {
    const [activeNamespace] = useActiveNamespace();
    const { searchFilter, setSearchFilter } = useURLSearch();
    const context = useDefaultWorkloadCveViewContext();
    const searchFilterConfig = [
        imageSearchFilterConfig,
        imageComponentSearchFilterConfig,
        deploymentSearchFilterConfig,
        ...(activeNamespace === ALL_NAMESPACES_KEY ? [namespaceSearchFilterConfig] : []),
    ];

    return (
        <WorkloadCveViewContext.Provider value={context}>
            <NamespaceBar
                // Force clear Namespace filter when the user changes the namespace via the NamespaceBar
                onNamespaceChange={() => setSearchFilter({ ...searchFilter, Namespace: [] })}
            />
            <ImageCvePage
                searchFilterConfig={searchFilterConfig}
                showVulnerabilityStateTabs={false}
                vulnerabilityState="OBSERVED"
                imageTableColumnOverrides={{}}
                deploymentTableColumnOverrides={{
                    cluster: hideColumnIf(true),
                    namespace: hideColumnIf(activeNamespace !== ALL_NAMESPACES_KEY),
                }}
            />
        </WorkloadCveViewContext.Provider>
    );
}
