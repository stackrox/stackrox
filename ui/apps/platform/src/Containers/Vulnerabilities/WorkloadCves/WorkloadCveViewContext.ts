import { createContext } from 'react';
import { NonEmptyArray } from 'utils/type.utils';

import type { VulnerabilityState } from 'types/cve.proto';
import { QuerySearchFilter, WorkloadEntityTab } from '../types';

export type WorkloadCveView = {
    urlBuilder: {
        vulnMgmtBase: (subPath: string) => string;
        cveList: (vulnerabilityState: VulnerabilityState) => string;
        cveDetails: (cve: string, vulnerabilityState: VulnerabilityState) => string;
        imageList: (vulnerabilityState: VulnerabilityState) => string;
        imageDetails: (id: string, vulnerabilityState: VulnerabilityState) => string;
        workloadList: (vulnerabilityState: VulnerabilityState) => string;
        workloadDetails: (
            deployment: {
                id: string;
                namespace: string;
                name: string;
                type: string;
            },
            vulnerabilityState: VulnerabilityState
        ) => string;
    };
    baseSearchFilter: QuerySearchFilter;
    pageTitle: string;
    overviewEntityTabs: NonEmptyArray<WorkloadEntityTab>;
    pageTitleDescription?: string;
    viewContext: string;
};

/**
 * The WorkloadCveViewContext provides dynamic values throughout the Workload CVE pages in order to support
 * sections for both "Application/User Workloads" and a separate section for "Platform Workloads"
 */
export const WorkloadCveViewContext = createContext<WorkloadCveView | undefined>(undefined);
