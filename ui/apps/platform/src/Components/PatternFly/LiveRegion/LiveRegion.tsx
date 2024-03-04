import React from 'react';
import LiveRegionContext from './liveRegionContext';

export type LiveRegionProps = {
    isUpdating: boolean;
    children: React.ReactNode;
};

function LiveRegion({ isUpdating, children }: LiveRegionProps) {
    return (
        <LiveRegionContext.Provider value={{ isUpdating }}>
            {React.Children.map(children, (child) => {
                if (React.isValidElement(child)) {
                    return React.cloneElement(child, {
                        ...child.props,
                        className: `${child.props.className ?? ''} acs-live-region`,
                        role: child.props.role ?? 'region',
                        'aria-live': child.props['aria-live'] ?? 'polite',
                        'aria-busy': isUpdating ? 'true' : 'false',
                    });
                }
                return child;
            })}
        </LiveRegionContext.Provider>
    );
}

export default LiveRegion;
