import PropTypes from 'prop-types';
import React from 'react';

/*
 * Display the contents of a health status cell in Clusters list or Cluster side panel
 * using composition so flexible content has consistent layout at the right of an icon
 *
 * The combination of `items-start` with `leading-normal` and `text-xs`
 * causes vertical alignment of `h-4` icon and the first line of text descendants:
 * - with similar intention but better visual appearance than `items-baseline`
 * - even if the parent table cell has `items-center`
 */
const HealthStatus = ({ children, Icon, iconColor }) => (
    <div className="flex flex-row items-start leading-normal">
        <span className={`flex-shrink-0 mr-2 ${iconColor}`}>
            <Icon className="h-4 w-4" />
        </span>
        {children}
    </div>
);

HealthStatus.propTypes = {
    children: PropTypes.element.isRequired, // flex-row assumes a child element wraps multiple grandchildren
    Icon: PropTypes.element.isRequired,
    iconColor: PropTypes.string.isRequired,
};

export default HealthStatus;
