import type { VirtualMachine } from 'services/VirtualMachineService';
import type { CVSSV3Severity } from 'types/vulnerability.proto';

// Most if not all functions in this file will be removed once backend filtering is implemented.

export function countVirtualMachineSeverities(
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
