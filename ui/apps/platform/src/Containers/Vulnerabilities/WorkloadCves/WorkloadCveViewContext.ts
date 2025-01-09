import { createContext } from 'react';
import { QuerySearchFilter } from '../types';

export type WorkloadCveView = {
    getAbsoluteUrl: (path: string) => string;
    baseSearchFilter: QuerySearchFilter;
    pageTitle: string;
};

/**
 * The WorkloadCveViewContext provides dynamic values throughout the Workload CVE pages in order to support
 * sections for both "Application/User Workloads" and a separate section for "Platform Workloads"
 */
export const WorkloadCveViewContext = createContext<WorkloadCveView | undefined>(undefined);
