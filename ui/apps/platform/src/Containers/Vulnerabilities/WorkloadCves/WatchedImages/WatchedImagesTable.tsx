import React, { CSSProperties } from 'react';
import { Table, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table';
import { Bullseye, Button, Icon } from '@patternfly/react-core';
import { MinusCircleIcon } from '@patternfly/react-icons';

import { WatchedImage } from 'types/image.proto';
import { UseRestMutationReturn } from 'hooks/useRestMutation';
import EmptyStateTemplate from 'Components/EmptyStateTemplate';
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
                <Table
                    aria-labelledby={props['aria-labelledby']}
                    variant="compact"
                    style={
                        {
                            '--pf-v5-c-table--m-compact--cell--first-last-child--PaddingLeft': '0',
                        } as CSSProperties
                    }
                >
                    <Thead noWrap>
                        <Tr>
                            <Th>Image</Th>
                            <Th>
                                <span className="pf-v5-screen-reader">Row action</span>
                            </Th>
                        </Tr>
                    </Thead>
                    <Tbody>
                        {watchedImages.map(({ name }) => (
                            <Tr key={name}>
                                <Td dataLabel="Image">{name}</Td>
                                <Td dataLabel="Row action" className="pf-v5-u-text-align-right">
                                    <Button
                                        variant="link"
                                        isInline
                                        icon={
                                            <Icon>
                                                <MinusCircleIcon color="var(--pf-v5-global--danger-color--100)" />
                                            </Icon>
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
                </Table>
            )}
        </div>
    );
}

export default WatchedImagesTable;
