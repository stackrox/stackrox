import { VulnerabilitySeverity } from 'types/cve.proto';

export type FixableStatus = 'Fixable' | 'Not fixable';

export type DefaultFilters = {
    Severity: VulnerabilitySeverity[];
    Fixable: FixableStatus[];
};

export type VulnMgmtLocalStorage = {
    preferences: {
        defaultFilters: DefaultFilters;
    };
};

const detailsTabValues = ['Vulnerabilities', 'Resources'] as const;

export type DetailsTab = typeof detailsTabValues[number];

export function isDetailsTab(value: unknown): value is DetailsTab {
    return detailsTabValues.some((tab) => tab === value);
}

const cveStatusTabValues = ['Observed', 'Deferred', 'False Positive'] as const;

export type CveStatusTab = typeof cveStatusTabValues[number];

export function isValidCveStatusTab(value: unknown): value is CveStatusTab {
    return cveStatusTabValues.some((tab) => tab === value);
}
