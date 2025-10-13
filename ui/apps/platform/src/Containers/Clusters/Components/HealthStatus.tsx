import React from 'react';
import type { ReactElement, ReactNode } from 'react';

/*
 * Display the contents of a health status cell in Clusters list or Cluster side panel
 * using composition so flexible content has consistent layout at the right of an icon
 *
 * The combination of `items-start` with `leading-normal` and `text-xs`
 * causes vertical alignment of `h-4` icon and the first line of text descendants:
 * - with similar intention but better visual appearance than `items-baseline`
 * - even if the parent table cell has `items-center`
 */

type HealthStatusProps = {
    children: ReactNode;
    icon: ReactElement;
    iconColor: string;
    isList?: boolean;
};

function HealthStatus({
    children,
    icon,
    iconColor,
    isList = false,
}: HealthStatusProps): ReactElement {
    return (
        // flex-row assumes a child element wraps multiple grandchildren
        <div className={`leading-normal ${isList ? 'inline' : 'flex flex-row items-start'}`}>
            <span className={`align-middle flex-shrink-0 mr-2 ${iconColor}`}>{icon}</span>
            {children}
        </div>
    );
}

export default HealthStatus;
