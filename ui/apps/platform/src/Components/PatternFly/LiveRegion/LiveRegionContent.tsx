import React, { useContext } from 'react';
import LiveRegionContext from './liveRegionContext';

export type LiveRegionContentProps = {
    children: React.ReactNode;
    shouldFadeWhenUpdating?: boolean;
};

function LiveRegionContent({ children, shouldFadeWhenUpdating = false }: LiveRegionContentProps) {
    const { isUpdating } = useContext(LiveRegionContext);
    // eslint-disable-next-line no-nested-ternary
    const fadeClassNames = shouldFadeWhenUpdating
        ? isUpdating
            ? 'acs-live-region-content-fade acs-live-region-content-fade-out'
            : 'acs-live-region-content-fade acs-live-region-content-fade-in'
        : '';

    return <div className={`acs-live-region-content ${fadeClassNames}`}>{children}</div>;
}

export default LiveRegionContent;
