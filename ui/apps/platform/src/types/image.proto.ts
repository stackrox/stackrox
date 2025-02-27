export type ImageName = {
    registry: string;
    remote: string;
    tag: string;
    fullName: string;
};

export type ListImage = {
    id: string;
    name: string;
    components?: number; // int32
    cves?: number; // int32;
    fixableCves: number; // int32
    created: string; // ISO 8601 date string
    lastUpdated: string; // ISO 8601 date string
    priority: string; // int64
};

export type ScanAudit = {
    id: string;
    eventTime: string;
    message: string;
    status: string;
    events: ScanAuditEvent[];
}

export type ScanAuditEvent = {
    time: string;
    status: string;
    message: string;
}

export type WatchedImage = {
    name: string;
};

export const sourceTypes = [
    'OS',
    'PYTHON',
    'JAVA',
    'RUBY',
    'NODEJS',
    'GO',
    'DOTNETCORERUNTIME',
    'INFRASTRUCTURE',
] as const;

export const sourceTypeLabels: Record<SourceType, string> = {
    OS: 'OS',
    PYTHON: 'Python',
    JAVA: 'Java',
    RUBY: 'Ruby',
    NODEJS: 'Node js',
    GO: 'Go',
    DOTNETCORERUNTIME: 'Dotnet Core Runtime',
    INFRASTRUCTURE: 'Infrastructure',
};

export type SourceType = (typeof sourceTypes)[number];
