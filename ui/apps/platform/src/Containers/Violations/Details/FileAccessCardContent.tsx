import type { ReactElement } from 'react';
import { DescriptionList, Divider, Flex, Title } from '@patternfly/react-core';
import lowerCase from 'lodash/lowerCase';
import upperFirst from 'lodash/upperFirst';

import DescriptionListItem from 'Components/DescriptionListItem';
import { getDateTime } from 'utils/dateUtils';
import type { FileAccess, FileOperation } from 'types/fileAccess.proto';

function formatOperation(operation: FileOperation): string {
    // Convert SCREAMING_SNAKE_CASE to Sentence case
    return upperFirst(lowerCase(operation));
}

type FileAccessCardContentProps = {
    event: FileAccess;
};

function FileAccessCardContent({ event }: FileAccessCardContentProps): ReactElement {
    const { file, operation, moved, timestamp, process } = event;

    return (
        <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsMd' }}>
            <Divider component="div" />
            <Title headingLevel="h3" className="pf-v6-u-pb-sm">
                {file.nodePath}
            </Title>
            <DescriptionList columnModifier={{ default: '2Col' }}>
                <DescriptionListItem term="File operation" desc={formatOperation(operation)} />
                <DescriptionListItem term="Time" desc={getDateTime(timestamp)} />
                {file.nodePath && <DescriptionListItem term="Node path" desc={file.nodePath} />}
                {file.mountedPath && (
                    <DescriptionListItem term="Mounted path" desc={file.mountedPath} />
                )}
                {moved && (
                    <DescriptionListItem
                        term="Moved to"
                        // `mountedPath` is relevant to deployment-based events, `nodePath` is relevant to node-based events
                        desc={moved.mountedPath || moved.nodePath}
                    />
                )}
                {process?.signal?.execFilePath && (
                    <DescriptionListItem term="Process" desc={process.signal.execFilePath} />
                )}
                {Number.isInteger(process?.signal?.uid) && (
                    <DescriptionListItem term="Process UID" desc={process.signal.uid} />
                )}
            </DescriptionList>
            {file.meta && <Title headingLevel="h4">File metadata:</Title>}
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
                            term="File mode"
                            desc={`${Number(file.meta.mode).toString(8)}`}
                        />
                    )}
                </DescriptionList>
            )}
        </Flex>
    );
}

export default FileAccessCardContent;
