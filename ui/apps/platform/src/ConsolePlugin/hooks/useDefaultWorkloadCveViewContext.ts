import { useMemo } from 'react';
import { useActiveNamespace } from '@openshift-console/dynamic-plugin-sdk';

import useURLSearch from 'hooks/useURLSearch';
import type { VulnerabilityState } from 'types/cve.proto';
import { ALL_NAMESPACES_KEY } from 'ConsolePlugin/constants';
import {
    getOverviewPagePath,
    getWorkloadEntityPagePath,
    parseQuerySearchFilter,
} from 'Containers/Vulnerabilities/utils/searchUtils';
import type { WorkloadCveView } from 'Containers/Vulnerabilities/WorkloadCves/WorkloadCveViewContext';

const acsSecurityVulnerabilitiesBase = '/acs/security/vulnerabilities';

export function useDefaultWorkloadCveViewContext(): WorkloadCveView {
    const [activeNamespace] = useActiveNamespace();
    const { searchFilter } = useURLSearch();

    return useMemo(() => {
        const querySearchFilter = parseQuerySearchFilter(searchFilter);
        return {
            baseSearchFilter: {
                'Image CVE Count': ['>0'],
                Namespace:
                    activeNamespace === ALL_NAMESPACES_KEY
                        ? querySearchFilter.Namespace
                        : [activeNamespace],
            },
            pageTitle: '',
            overviewEntityTabs: ['CVE', 'Image', 'Deployment'],
            viewContext: '',
            urlBuilder: {
                vulnMgmtBase: (subPath: string) => `${acsSecurityVulnerabilitiesBase}/${subPath}`,
                cveList: (vulnerabilityState: VulnerabilityState) =>
                    `${acsSecurityVulnerabilitiesBase}/${getOverviewPagePath('Workload', {
                        vulnerabilityState,
                        entityTab: 'CVE',
                    })}`,
                cveDetails: (cve: string, vulnerabilityState: VulnerabilityState) =>
                    `${acsSecurityVulnerabilitiesBase}/${getWorkloadEntityPagePath('CVE', cve, vulnerabilityState)}`,
                imageList: (vulnerabilityState: VulnerabilityState) =>
                    `${acsSecurityVulnerabilitiesBase}/${getOverviewPagePath('Workload', {
                        vulnerabilityState,
                        entityTab: 'Image',
                    })}`,
                imageDetails: (id: string, vulnerabilityState: VulnerabilityState) =>
                    `${acsSecurityVulnerabilitiesBase}/${getWorkloadEntityPagePath('Image', id, vulnerabilityState)}`,
                workloadList: (vulnerabilityState: VulnerabilityState) =>
                    `${acsSecurityVulnerabilitiesBase}/${getOverviewPagePath('Workload', {
                        vulnerabilityState,
                        entityTab: 'Deployment',
                    })}`,
                workloadDetails: (workload: {
                    id: string;
                    name: string;
                    namespace: string;
                    type: string;
                }) =>
                    `/k8s/ns/${workload.namespace}/${workload.type.toLowerCase()}s/${workload.name}/security`,
            },
        };
    }, [activeNamespace, searchFilter]);
}
