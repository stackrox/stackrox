import React, { ReactElement } from 'react';
import { Tooltip } from '@patternfly/react-core';

import IconText from 'Components/PatternFly/IconText/IconText';

import { ClusterStatusObject } from '../compliance.coverage.utils';

export type StatusIconProps = {
    clusterStatusObject: ClusterStatusObject;
};

function StatusIcon({ clusterStatusObject }: StatusIconProps): ReactElement {
    const { icon, statusText, tooltipText } = clusterStatusObject;

    const iconText = <IconText icon={icon} text={statusText} />;

    return typeof tooltipText === 'string' ? (
        <Tooltip content={tooltipText} isContentLeftAligned>
            {iconText}
        </Tooltip>
    ) : (
        iconText
    );
}

export default StatusIcon;
