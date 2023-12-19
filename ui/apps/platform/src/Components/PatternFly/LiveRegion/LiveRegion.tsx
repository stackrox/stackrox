import React from 'react';
import LiveRegionContext from './liveRegionContext';

export type LiveRegionProps = {
    children: React.ReactNode;
    isUpdating: boolean;
    className?: string;
};

function LiveRegion({ isUpdating, children, className = '' }: LiveRegionProps) {
    return (
        <LiveRegionContext.Provider value={{ isUpdating }}>
            <div
                className={`acs-live-region ${className}`}
                role="region"
                aria-live="polite"
                aria-busy={isUpdating ? 'true' : 'false'}
            >
                {children}
            </div>
        </LiveRegionContext.Provider>
    );
}

export default LiveRegion;
