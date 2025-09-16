import type { VirtualMachine } from 'services/VirtualMachineService';
import type { ScanComponent } from 'types/scanComponent.proto';
import type { CVSSV3Severity, EmbeddedVulnerability } from 'types/vulnerability.proto';

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

type AffectedComponent = Omit<ScanComponent, 'vulns'>;
type CVEId = EmbeddedVulnerability['cve'];

export type CVEEntry = EmbeddedVulnerability & {
    affectedComponents: AffectedComponent[];
};

// Dictionary keyed by CVE (example: "CVE-2024-00010": { affectedComponents: [...] })
type CVEDictionary = Record<CVEId, CVEEntry>;

export function getVirtualMachineCvesDictionaryData(virtualMachine: VirtualMachine): CVEDictionary {
    const components = virtualMachine?.scan?.components ?? [];

    const cvesWithAffectedComponents: CVEDictionary = {};

    components.forEach((comp) => {
        const { vulns, ...componentDetails } = comp;
        const componentWithoutVulns: AffectedComponent = componentDetails;
        vulns.forEach((vulnerability) => {
            if (cvesWithAffectedComponents[vulnerability.cve]) {
                cvesWithAffectedComponents[vulnerability.cve].affectedComponents.push(
                    componentWithoutVulns
                );
            } else {
                cvesWithAffectedComponents[vulnerability.cve] = {
                    ...vulnerability,
                    affectedComponents: [componentWithoutVulns],
                };
            }
        });
    });

    return cvesWithAffectedComponents;
}

export type CVEWithAffectedComponents = EmbeddedVulnerability & {
    cve: CVEId;
    affectedComponents: AffectedComponent[];
};

// Returns a sorted list of CVE entries with their affected components
export function getVirtualMachineCvesListData(
    virtualMachine?: VirtualMachine
): CVEWithAffectedComponents[] {
    if (!virtualMachine) {
        return [];
    }

    const dict = getVirtualMachineCvesDictionaryData(virtualMachine);
    return Object.keys(dict)
        .sort()
        .map((cve) => ({
            ...dict[cve],
            affectedComponents: dict[cve].affectedComponents,
        }));
}
