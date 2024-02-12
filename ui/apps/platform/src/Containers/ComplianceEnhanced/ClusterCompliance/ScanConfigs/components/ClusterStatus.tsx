import React from 'react';
import { Button, Popover } from '@patternfly/react-core';

import IconText from 'Components/PatternFly/IconText/IconText';

import { getClusterStatusObject } from '../compliance.scanConfigs.utils';

export type ClusterStatusProps = {
    errors: string[];
};

function ClusterStatus({ errors }: ClusterStatusProps) {
    const statusObj = getClusterStatusObject(errors);
    return statusObj.statusText === 'Healthy' ? (
        <IconText icon={statusObj.icon} text={statusObj.statusText} />
    ) : (
        <Popover
            aria-label="Reveal errors"
            headerContent={<div>{errors.length === 1 ? 'Error' : 'Errors'}</div>}
            bodyContent={<div>{errors.join(', ')}</div>}
        >
            <Button variant="link">
                <IconText icon={statusObj.icon} text={statusObj.statusText} />
            </Button>
        </Popover>
    );
}

export default ClusterStatus;
