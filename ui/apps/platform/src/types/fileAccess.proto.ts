import type { ProcessIndicator } from './processIndicator.proto';

export type FileAccess = {
    file: File;
    operation: FileOperation;
    moved: File | null; // specific to RENAME activity, the new location / metadata of the file
    timestamp: string; // ISO 8601 date string
    process: ProcessIndicator;
    hostname: string; // The hostname/name of the node where the file activity occurred
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
    aclType: AclType | null; // only relevant for ACL_CHANGE events
    aclEntries: AclEntry[]; // only relevant for ACL_CHANGE events
};

export type AclTag =
    | 'ACL_TAG_UNSPECIFIED'
    | 'ACL_TAG_USER_OBJ'
    | 'ACL_TAG_USER'
    | 'ACL_TAG_GROUP_OBJ'
    | 'ACL_TAG_GROUP'
    | 'ACL_TAG_MASK'
    | 'ACL_TAG_OTHER';

export type AclType = 'ACL_TYPE_UNSPECIFIED' | 'ACL_TYPE_ACCESS' | 'ACL_TYPE_DEFAULT';

export type AclEntry = {
    tag: AclTag;
    perm: number; // Permission bits (e.g. 7 = rwx, 6 = rw-, 4 = r--)
    id: number; // uid or gid for USER/GROUP entries, 0xFFFFFFFF when not applicable
};

export type FileOperation =
    | 'CREATE'
    | 'UNLINK'
    | 'RENAME'
    | 'PERMISSION_CHANGE'
    | 'OWNERSHIP_CHANGE'
    | 'OPEN'
    | 'ACL_CHANGE';
