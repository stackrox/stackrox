import type { EmbeddedVulnerability } from 'types/vulnerability.proto';

/** ===== Based on v2/scan_component.proto ===== */

export type ScanComponent = {
    name: string;
    version: string;
    license: License;
    vulns: EmbeddedVulnerability[];
    source: SourceType;
    location: string;
    topCvss?: number;
    riskScore: number;
    fixedBy: string;
    executables: ScanComponentExecutable[];
    architecture: string;
};

type ScanComponentExecutable = {
    path: string;
    dependencies: string[];
};

export type SourceType =
    | 'OS'
    | 'PYTHON'
    | 'JAVA'
    | 'RUBY'
    | 'NODEJS'
    | 'GO'
    | 'DOTNETCORERUNTIME'
    | 'INFRASTRUCTURE';

type License = {
    name: string;
    type: string;
    url: string;
};
