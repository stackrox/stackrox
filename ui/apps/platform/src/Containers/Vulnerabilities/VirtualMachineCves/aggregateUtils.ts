import { severityRankings } from 'constants/vulnerabilities';
import type { VirtualMachine } from 'services/VirtualMachineService';
import type { VulnerabilitySeverity } from 'types/cve.proto';
import type { ScanComponent, SourceType } from 'types/scanComponent.proto';
import type { SearchFilter } from 'types/search';
import type { Advisory, CVSSV3Severity, EmbeddedVulnerability } from 'types/vulnerability.proto';
import { searchValueAsArray } from 'utils/searchUtils';

import { isVulnerabilitySeverityLabel } from '../types';
import type { FixableStatus } from '../types';
import { severityLabelToSeverity } from '../utils/searchUtils';

// Most if not all functions in this file will be removed once backend filtering is implemented.

export function getVirtualMachineSeveritiesCount(
    virtualMachine: VirtualMachine
): Record<CVSSV3Severity, number> {
    const severityCounts: Record<CVSSV3Severity, number> = {
        CRITICAL: 0,
        HIGH: 0,
        MEDIUM: 0,
        LOW: 0,
        UNKNOWN: 0,
        NONE: 0,
    };

    virtualMachine.scan.components.forEach((component) => {
        component.vulns.forEach((vuln) => {
            const { severity } = vuln.cvssV3;
            severityCounts[severity] += 1;
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

function defaultCveTableSort(a: CveTableRow, b: CveTableRow): number {
    return severityRankings[b.severity] - severityRankings[a.severity] || b.cvss - a.cvss;
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

    return Array.from(map.values()).sort(defaultCveTableSort);
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
