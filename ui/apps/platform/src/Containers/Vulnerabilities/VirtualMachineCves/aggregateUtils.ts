import { severityRankings } from 'constants/vulnerabilities';
import type { VirtualMachine } from 'services/VirtualMachineService';
import type { VulnerabilitySeverity } from 'types/cve.proto';
import type { ScanComponent, SourceType } from 'types/scanComponent.proto';
import type { SearchFilter } from 'types/search';
import type { Advisory, EmbeddedVulnerability } from 'types/vulnerability.proto';
import { searchValueAsArray } from 'utils/searchUtils';

import { severityToQuerySeverityKeys } from '../components/BySeveritySummaryCard';
import type { ResourceCountByCveSeverityAndStatus } from '../components/CvesByStatusSummaryCard';
import { isVulnerabilitySeverityLabel } from '../types';
import type { FixableStatus } from '../types';
import { severityLabelToSeverity } from '../utils/searchUtils';
import {
    CVE_EPSS_PROBABILITY_SORT_FIELD,
    CVE_SEVERITY_SORT_FIELD,
    CVE_SORT_FIELD,
    CVE_STATUS_SORT_FIELD,
    CVSS_SORT_FIELD,
} from '../utils/sortFields';

// Most if not all functions in this file will be removed once backend filtering is implemented.

export function getVirtualMachineSeveritiesCount(
    virtualMachine: VirtualMachine
): Record<VulnerabilitySeverity, number> {
    const severityCounts: Record<VulnerabilitySeverity, number> = {
        CRITICAL_VULNERABILITY_SEVERITY: 0,
        IMPORTANT_VULNERABILITY_SEVERITY: 0,
        MODERATE_VULNERABILITY_SEVERITY: 0,
        LOW_VULNERABILITY_SEVERITY: 0,
        UNKNOWN_VULNERABILITY_SEVERITY: 0,
    };

    virtualMachine.scan.components.forEach((component) => {
        component.vulns.forEach((vuln) => {
            severityCounts[vuln.severity] += 1;
        });
    });

    return severityCounts;
}

export type CveComponentRow = {
    name: ScanComponent['name'];
    sourceType: SourceType;
    version: ScanComponent['version'];
    fixedBy: EmbeddedVulnerability['fixedBy'];
    advisory: Advisory;
};

export type CveTableRow = {
    cve: EmbeddedVulnerability['cve'];
    severity: VulnerabilitySeverity; // worst severity across components for this CVE
    isFixable: boolean; // true if any vulnerability is fixable across components
    cvss: number; // max score across components
    epssProbability: number; // should be the same across all components
    affectedComponents: CveComponentRow[];
};

export function getVirtualMachineCveSeverityStatusCounts(
    cveTableData: CveTableRow[]
): ResourceCountByCveSeverityAndStatus {
    const counts: ResourceCountByCveSeverityAndStatus = {
        critical: { total: 0, fixable: 0 },
        important: { total: 0, fixable: 0 },
        moderate: { total: 0, fixable: 0 },
        low: { total: 0, fixable: 0 },
        unknown: { total: 0, fixable: 0 },
    };

    cveTableData.forEach((cveTableRow) => {
        const severityKey = severityToQuerySeverityKeys[cveTableRow.severity];
        counts[severityKey].total += 1;
        counts[severityKey].fixable += cveTableRow.isFixable ? 1 : 0;
    });

    return counts;
}

function worstSeverity(a: VulnerabilitySeverity, b: VulnerabilitySeverity): VulnerabilitySeverity {
    return severityRankings[a] >= severityRankings[b] ? a : b;
}

export function getVirtualMachineCveTableData(virtualMachine?: VirtualMachine): CveTableRow[] {
    if (!virtualMachine) {
        return [];
    }

    const map = new Map<string, CveTableRow>();

    virtualMachine.scan?.components?.forEach((component) => {
        component.vulns?.forEach((vulnerability) => {
            const { advisory, cve, cvss, epss, fixedBy, severity } = vulnerability;

            let row = map.get(cve);
            if (!row) {
                row = {
                    cve,
                    severity,
                    isFixable: !!fixedBy,
                    cvss,
                    epssProbability: epss.epssProbability,
                    affectedComponents: [],
                };
                map.set(cve, row);
            }

            // update row with worse severity/score
            row.severity = worstSeverity(row.severity, severity);
            row.cvss = Math.max(row.cvss, cvss);

            // display fixable if any vulnerability is fixable
            row.isFixable = row.isFixable || !!fixedBy;

            row.affectedComponents.push({
                name: component.name,
                sourceType: component.source,
                version: component.version,
                fixedBy,
                advisory,
            });
        });
    });

    return Array.from(map.values());
}

export function applyVirtualMachineCveTableFilters(
    cveTableData: CveTableRow[],
    searchFilter: SearchFilter
): CveTableRow[] {
    if (!searchFilter || Object.keys(searchFilter).length === 0) {
        return cveTableData;
    }

    // normalize filters
    // - convert text filters to lowercase (CVE, Component, Component Version)
    // - map  severity ui labels to severity data values (enum)
    const cveFilters = searchValueAsArray(searchFilter.CVE).map((cve) => cve.toLowerCase());
    const componentFilters = searchValueAsArray(searchFilter.Component).map((component) =>
        component.toLowerCase()
    );
    const componentVersionFilters = searchValueAsArray(searchFilter['Component Version']).map(
        (version) => version.toLowerCase()
    );
    const severityFilters = searchValueAsArray(searchFilter.SEVERITY)
        .filter(isVulnerabilitySeverityLabel)
        .map(severityLabelToSeverity);

    const fixableFilters = searchValueAsArray(searchFilter.FIXABLE);

    return cveTableData.filter((cveTableRow) => {
        // "CVE" filter, case insensitive and substring
        if (cveFilters.length > 0) {
            const cveNameLowerCase = cveTableRow.cve.toLowerCase();
            if (!cveFilters.some((filter) => cveNameLowerCase.includes(filter))) {
                return false;
            }
        }

        // "SEVERITY" filter, exact
        if (severityFilters.length > 0) {
            if (!severityFilters.includes(cveTableRow.severity)) {
                return false;
            }
        }

        // "FIXABLE" filter, exact
        if (fixableFilters.length > 0) {
            const rowFixable: FixableStatus = cveTableRow.isFixable ? 'Fixable' : 'Not fixable';
            if (!fixableFilters.includes(rowFixable)) {
                return false;
            }
        }

        // "Component" filter, case insensitive and substring
        if (componentFilters.length > 0) {
            const components = cveTableRow.affectedComponents ?? [];
            const hasMatch = components.some((comp) => {
                const compNameLowerCase = comp.name.toLowerCase();
                return componentFilters.some((filter) => compNameLowerCase.includes(filter));
            });
            if (!hasMatch) {
                return false;
            }
        }

        // "Component Version" filter, case insensitive and substring
        if (componentVersionFilters.length > 0) {
            const components = cveTableRow.affectedComponents ?? [];
            const hasMatch = components.some((comp) => {
                const versionLowerCase = (comp.version ?? '').toLowerCase();
                return componentVersionFilters.some((filter) => versionLowerCase.includes(filter));
            });
            if (!hasMatch) {
                return false;
            }
        }

        return true; // passed all filter conditions
    });
}

export function applyVirtualMachineCveTableSort(
    rows: CveTableRow[],
    sortKey: string,
    reversed: boolean
): CveTableRow[] {
    const comparator = (a: CveTableRow, b: CveTableRow) => {
        let compareResult = 0;

        switch (sortKey) {
            case CVE_SORT_FIELD:
                // Intl.Collator sorting would be better here, but it's not used in the backend
                compareResult = a.cve.localeCompare(b.cve);
                break;
            case CVSS_SORT_FIELD:
                compareResult = a.cvss - b.cvss;
                break;
            case CVE_EPSS_PROBABILITY_SORT_FIELD:
                compareResult = a.epssProbability - b.epssProbability;
                break;
            case CVE_STATUS_SORT_FIELD:
                compareResult = Number(a.isFixable) - Number(b.isFixable);
                break;
            case CVE_SEVERITY_SORT_FIELD:
                compareResult = severityRankings[a.severity] - severityRankings[b.severity];
                break;
            default:
                break;
        }
        if (compareResult !== 0) {
            return reversed ? compareResult * -1 : compareResult;
        }

        // backup compare when rows are equal
        // doesn't appear to be a consistent behavior in the backend between vulnerability pages
        // however secondary sort of cve name seems to mimic node cve page behavior
        return a.cve.localeCompare(b.cve);
    };

    return [...rows].sort(comparator);
}
