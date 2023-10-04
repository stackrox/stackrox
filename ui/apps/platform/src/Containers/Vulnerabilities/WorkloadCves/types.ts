import { VulnerabilitySeverity } from 'types/cve.proto';

export type VulnerabilitySeverityLabel = 'Critical' | 'Important' | 'Moderate' | 'Low';
export type FixableStatus = 'Fixable' | 'Not fixable';

export type DefaultFilters = {
    Severity: VulnerabilitySeverityLabel[];
    Fixable: FixableStatus[];
};

// `QuerySearchFilter` is a restricted subset of the `SearchFilter` obtained from the URL that only
// supports search keys that are valid in the Workload CVE section of the app
export type QuerySearchFilter = Partial<{
    Severity: VulnerabilitySeverity[];
    Fixable: ('true' | 'false')[];
    CVE: string[];
    IMAGE: string[];
    DEPLOYMENT: string[];
    NAMESPACE: string[];
    CLUSTER: string[];
}>;

export type VulnMgmtLocalStorage = {
    preferences: {
        defaultFilters: DefaultFilters;
    };
};

export const detailsTabValues = ['Vulnerabilities', 'Resources'] as const;

export type DetailsTab = (typeof detailsTabValues)[number];

export function isDetailsTab(value: unknown): value is DetailsTab {
    return detailsTabValues.some((tab) => tab === value);
}

export const cveStatusTabValues = ['Observed', 'Deferred', 'False Positive'] as const;

export type CveStatusTab = (typeof cveStatusTabValues)[number];

export function isValidCveStatusTab(value: unknown): value is CveStatusTab {
    return cveStatusTabValues.some((tab) => tab === value);
}

export const entityTabValues = ['CVE', 'Image', 'Deployment'] as const;

export type EntityTab = (typeof entityTabValues)[number];

export function isValidEntityTab(value: unknown): value is EntityTab {
    return entityTabValues.some((tab) => tab === value);
}

export type WatchStatus = 'WATCHED' | 'NOT_WATCHED' | 'UNKNOWN';
