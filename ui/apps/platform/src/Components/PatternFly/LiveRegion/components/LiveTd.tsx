import React, { useContext } from 'react';
import { Flex, FlexItem, Skeleton } from '@patternfly/react-core';
import { Td, TdProps } from '@patternfly/react-table';

import LiveRegionContext from '../liveRegionContext';

export type LiveTdProps = Omit<TdProps, 'ref'>;

function LiveTd({ children, ...props }: LiveTdProps) {
    const { isUpdating } = useContext(LiveRegionContext);
    return (
        <Td {...props}>
            {isUpdating ? (
                /*
                  This layout prevents column-shift by keeping the previous content in the DOM
                  and setting its visibility to hidden. This way, the Skeleton can be rendered
                  without affecting the layout of the table.
                */
                <Flex style={{ position: 'relative' }} alignItems={{ default: 'alignItemsCenter' }}>
                    <Skeleton className="pf-v5-u-w-100" style={{ position: 'absolute' }} />
                    <FlexItem style={{ visibility: 'hidden' }}>{children}</FlexItem>
                </Flex>
            ) : (
                children
            )}
        </Td>
    );
}

export default LiveTd;
