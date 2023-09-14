import React, { CSSProperties } from 'react';
import { TableComposable, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table';
import { Bullseye, Button } from '@patternfly/react-core';
import { MinusCircleIcon } from '@patternfly/react-icons';

import { WatchedImage } from 'types/image.proto';
import { UseRestMutationReturn } from 'hooks/useRestMutation';
import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';
import { Empty } from 'services/types';

export type WatchedImagesTableProps = {
    className?: string;
    style?: CSSProperties;
    watchedImages: WatchedImage[];
    unwatchImage: UseRestMutationReturn<string, Empty>['mutate'];
    isUnwatchInProgress: boolean;
    'aria-labelledby'?: string;
};

function WatchedImagesTable({
    className,
    style,
    watchedImages,
    unwatchImage,
    isUnwatchInProgress,
    ...props
}: WatchedImagesTableProps) {
    return (
        <div className={className} style={style}>
            {watchedImages.length === 0 && (
                <Bullseye>
                    <EmptyStateTemplate title="No watched images found" headingLevel="h2" />
                </Bullseye>
            )}
            {watchedImages.length > 0 && (
                <TableComposable
                    aria-labelledby={props['aria-labelledby']}
                    variant="compact"
                    style={
                        {
                            '--pf-c-table--m-compact--cell--first-last-child--PaddingLeft': '0',
                        } as CSSProperties
                    }
                >
                    <Thead noWrap>
                        <Tr>
                            <Th>Image</Th>
                            <Th aria-label="Remove watched image" />
                        </Tr>
                    </Thead>
                    <Tbody>
                        {watchedImages.map(({ name }) => (
                            <Tr key={name}>
                                <Td dataLabel="Image">{name}</Td>
                                <Td
                                    dataLabel="Remove watched image"
                                    className="pf-u-text-align-right"
                                >
                                    <Button
                                        variant="link"
                                        isInline
                                        icon={
                                            <MinusCircleIcon color="var(--pf-global--Color--200)" />
                                        }
                                        onClick={() => unwatchImage(name)}
                                        disabled={isUnwatchInProgress}
                                    >
                                        Remove watch
                                    </Button>
                                </Td>
                            </Tr>
                        ))}
                    </Tbody>
                </TableComposable>
            )}
        </div>
    );
}

export default WatchedImagesTable;
