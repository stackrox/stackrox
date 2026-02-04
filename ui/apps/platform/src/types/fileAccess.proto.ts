import type { ProcessIndicator } from './processIndicator.proto';

export type FileAccess = {
    file: File;
    operation: FileOperation;
    moved: File | null; // specific to RENAME activity, the new location / metadata of the file
    timestamp: string; // ISO 8601 date string
    process: ProcessIndicator;
};

export type File = {
    effectivePath: string; // Relevant to deployment-based events, the path in the container
    actualPath: string; // The path on the node file system
    meta: FileMetadata | null;
};

export type FileMetadata = {
    uid: number | null; // only relevant for OWNERSHIP_CHANGE events
    gid: number | null; // only relevant for OWNERSHIP_CHANGE events
    // `mode` is the base-10 (uint32) representation of the file mode, which should be formatted as an octal string
    mode: number | null; // only relevant for PERMISSION_CHANGE events
    username: string | null; // only relevant for OWNERSHIP_CHANGE events
    group: string | null; // only relevant for OWNERSHIP_CHANGE events
};

export type FileOperation =
    | 'CREATE'
    | 'UNLINK'
    | 'RENAME'
    | 'PERMISSION_CHANGE'
    | 'OWNERSHIP_CHANGE'
    | 'OPEN';
