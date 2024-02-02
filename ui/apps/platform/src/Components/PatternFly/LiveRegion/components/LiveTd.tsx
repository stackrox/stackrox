import React, { useContext } from 'react';
import { Skeleton } from '@patternfly/react-core';
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
                <div style={{ position: 'relative' }}>
                    <Skeleton className="pf-u-w-100 pf-u-h-100" style={{ position: 'absolute' }} />
                    <div style={{ visibility: 'hidden' }}>{children}</div>
                </div>
            ) : (
                children
            )}
        </Td>
    );
}

export default LiveTd;
