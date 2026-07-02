import type { ReactElement } from 'react';
import { DescriptionList, Divider, Flex, Title } from '@patternfly/react-core';

import DescriptionListItem from 'Components/DescriptionListItem';
import { getDateTime } from 'utils/dateUtils';
import type { AclEntry, AclTag, AclType, FileAccess, FileOperation } from 'types/fileAccess.proto';

const fileOperations: Map<FileOperation, string> = new Map([
    ['OPEN', 'Open (Writable)'],
    ['CREATE', 'Create'],
    ['UNLINK', 'Delete (Unlink)'],
    ['RENAME', 'Rename'],
    ['PERMISSION_CHANGE', 'Permission change'],
    ['OWNERSHIP_CHANGE', 'Ownership change'],
    ['ACL_CHANGE', 'ACL change'],
]);

const aclTagLabels: Map<AclTag, string> = new Map([
    ['ACL_TAG_USER_OBJ', 'Owner'],
    ['ACL_TAG_USER', 'User'],
    ['ACL_TAG_GROUP_OBJ', 'Owning group'],
    ['ACL_TAG_GROUP', 'Group'],
    ['ACL_TAG_MASK', 'Mask'],
    ['ACL_TAG_OTHER', 'Other'],
]);

const aclTypeLabels: Map<AclType, string> = new Map([
    ['ACL_TYPE_ACCESS', 'Access'],
    ['ACL_TYPE_DEFAULT', 'Default'],
]);

const NO_ID = 0xffffffff;

// Map each octal digit (0-7) to its rwx permission string.
// Shared between formatFileMode (full 9-char mode) and ACL entry formatting.
const octalToPermission: Record<string, string> = {
    '0': '---',
    '1': '--x',
    '2': '-w-',
    '3': '-wx',
    '4': 'r--',
    '5': 'r-x',
    '6': 'rw-',
    '7': 'rwx',
};

function formatOctalDigit(digit: number): string {
    return octalToPermission[String(digit & 7)] || '---';
}

function formatAclEntry(entry: AclEntry): string {
    const tagLabel = aclTagLabels.get(entry.tag) || entry.tag;
    const permStr = formatOctalDigit(entry.perm);
    if ((entry.tag === 'ACL_TAG_USER' || entry.tag === 'ACL_TAG_GROUP') && entry.id !== NO_ID) {
        return `${tagLabel}(${entry.id}): ${permStr}`;
    }
    return `${tagLabel}: ${permStr}`;
}

function formatOperation(operation: FileOperation): string {
    return fileOperations.get(operation) || 'Unknown';
}

/**
 * Converts a numeric file mode to a Linux file permissions string.
 *
 * @param mode - The file mode as a base-10 number (e.g., 33188 for 0o100644)
 * @returns A string representation of the permissions (e.g., "rw-r--r--")
 *
 * @example
 * formatFileMode(33188) // returns "rw-r--r--" (0o644)
 * formatFileMode(33261) // returns "rwxr-xr-x" (0o755)
 * formatFileMode(16877) // returns "rwxr-xr-x" (directory with 0o755)
 */
function formatFileMode(mode: number): string {
    // Extract the permission bits (lower 9 bits) and convert to octal string
    const permissionBits = mode % 512; // 512 = 0o1000, equivalent to mode & 0o777
    const octalString = permissionBits.toString(8).padStart(3, '0');

    // Convert each octal digit to its permission string
    return octalString
        .split('')
        .map((digit) => octalToPermission[digit])
        .join('');
}

type FileAccessCardContentProps = {
    event: FileAccess;
};

function FileAccessCardContent({ event }: FileAccessCardContentProps): ReactElement {
    const { file, operation, moved, timestamp, process, hostname } = event;

    return (
        <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsMd' }}>
            <Divider component="div" />
            <Title headingLevel="h3" className="pf-v6-u-pb-sm">
                {file.actualPath}
            </Title>
            <DescriptionList columnModifier={{ default: '2Col' }}>
                <DescriptionListItem term="File operation" desc={formatOperation(operation)} />
                <DescriptionListItem term="Time" desc={getDateTime(timestamp)} />
                {hostname && <DescriptionListItem term="Hostname" desc={hostname} />}
                {file.actualPath && (
                    <DescriptionListItem term="Actual path" desc={file.actualPath} />
                )}
                {file.effectivePath && (
                    <DescriptionListItem term="Effective path" desc={file.effectivePath} />
                )}
                {moved && (
                    <DescriptionListItem
                        term="Moved to"
                        // `effectivePath` is relevant to deployment-based events, `actualPath` is relevant to node-based events
                        desc={moved.effectivePath || moved.actualPath}
                    />
                )}
                {process?.signal?.name && (
                    <DescriptionListItem term="Process name" desc={process.signal.name} />
                )}
                {process?.signal?.execFilePath && (
                    <DescriptionListItem
                        term="Process executable"
                        desc={process.signal.execFilePath}
                    />
                )}
                {Number.isInteger(process?.signal?.uid) && (
                    <DescriptionListItem term="Process UID" desc={process.signal.uid} />
                )}
            </DescriptionList>
            {file.meta && <Title headingLevel="h4">File metadata</Title>}
            {file.meta && (
                <DescriptionList columnModifier={{ default: '2Col' }}>
                    {file.meta.username && (
                        <DescriptionListItem term="Owner" desc={file.meta.username} />
                    )}
                    {file.meta.group && <DescriptionListItem term="Group" desc={file.meta.group} />}
                    {Number.isInteger(file.meta.uid) && (
                        <DescriptionListItem term="UID" desc={file.meta.uid} />
                    )}
                    {Number.isInteger(file.meta.gid) && (
                        <DescriptionListItem term="GID" desc={file.meta.gid} />
                    )}
                    {Number.isInteger(file.meta.mode) && (
                        <DescriptionListItem
                            term="Permissions"
                            desc={`${formatFileMode(Number(file.meta.mode))} (${Number(file.meta.mode).toString(8).padStart(4, '0')})`}
                        />
                    )}
                    {file.meta.aclType && (
                        <DescriptionListItem
                            term="ACL type"
                            desc={aclTypeLabels.get(file.meta.aclType) || file.meta.aclType}
                        />
                    )}
                    {file.meta.aclEntries?.length > 0 && (
                        <DescriptionListItem
                            term="ACL entries"
                            desc={file.meta.aclEntries.map(formatAclEntry).join(', ')}
                        />
                    )}
                </DescriptionList>
            )}
        </Flex>
    );
}

export default FileAccessCardContent;
