import React, { useContext } from 'react';
import { Flex } from '@patternfly/react-core';

import LiveRegionContext from './liveRegionContext';

export type LiveRegionOverlayProps = {
    children: React.ReactNode;
};

function LiveRegionOverlay({ children }: LiveRegionOverlayProps) {
    const { isUpdating } = useContext(LiveRegionContext);
    if (!isUpdating || !children) {
        return null;
    }
    return (
        <Flex
            className="acs-live-region-overlay"
            direction={{ default: 'column' }}
            alignItems={{ default: 'alignItemsCenter' }}
        >
            {children}
        </Flex>
    );
}

export default LiveRegionOverlay;
