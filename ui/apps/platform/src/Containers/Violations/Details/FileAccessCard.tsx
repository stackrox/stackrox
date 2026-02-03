import { useState } from 'react';
import type { ReactElement } from 'react';
import {
    Card,
    CardBody,
    CardExpandableContent,
    CardHeader,
    CardTitle,
    Flex,
} from '@patternfly/react-core';

import FileAccessCardContent from './FileAccessCardContent';
import type { FileAccess } from 'types/fileAccess.proto';

type FileAccessCardProps = {
    fileAccess: FileAccess;
    message: string;
};

function FileAccessCard({ fileAccess, message }: FileAccessCardProps): ReactElement {
    const [isExpanded, setIsExpanded] = useState(true);

    function onExpand() {
        setIsExpanded((prev) => !prev);
    }

    return (
        <div className="pf-v5-u-pb-md">
            <Card isExpanded={isExpanded} isFlat>
                <CardHeader
                    onExpand={onExpand}
                    toggleButtonProps={{ 'aria-expanded': isExpanded, 'aria-label': 'Details' }}
                >
                    <CardTitle>{message}</CardTitle>
                </CardHeader>
                <CardExpandableContent>
                    <CardBody className="pf-v5-u-mt-lg">
                        <Flex
                            direction={{ default: 'column' }}
                            spaceItems={{ default: 'spaceItemsMd' }}
                        >
                            <FileAccessCardContent event={fileAccess} />
                        </Flex>
                    </CardBody>
                </CardExpandableContent>
            </Card>
        </div>
    );
}

export default FileAccessCard;
