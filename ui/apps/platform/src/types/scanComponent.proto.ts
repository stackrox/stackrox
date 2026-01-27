import type { EmbeddedVulnerability } from 'types/vulnerability.proto';

/** ===== Based on v2/scan_component.proto ===== */

export type ScanComponent = {
    name: string;
    version: string;
    vulns: EmbeddedVulnerability[];
    source: SourceType;
    topCvss?: number;
    riskScore: number;
    architecture: string;
    notes: Note[];
};

type Note = 'UNSPECIFIED' | 'UNSCANNED';

export type SourceType =
    | 'OS'
    | 'PYTHON'
    | 'JAVA'
    | 'RUBY'
    | 'NODEJS'
    | 'GO'
    | 'DOTNETCORERUNTIME'
    | 'INFRASTRUCTURE';
