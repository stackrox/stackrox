import React, { ReactElement } from 'react';
import { Button, Popover } from '@patternfly/react-core';
import { CheckCircleIcon, ExclamationCircleIcon } from '@patternfly/react-icons';

import IconText from 'Components/PatternFly/IconText/IconText';

type ClusterStatusObject = {
    icon: ReactElement;
    statusText: string;
};

export type ComplianceClusterStatusProps = {
    errors: string[];
};

function ComplianceClusterStatus({ errors }: ComplianceClusterStatusProps) {
    function getClusterStatusObject(errors: string[]): ClusterStatusObject {
        return errors && errors.length && errors[0] !== ''
            ? {
                  icon: <ExclamationCircleIcon color="var(--pf-global--danger-color--100)" />,
                  statusText: 'Unhealthy',
              }
            : {
                  icon: <CheckCircleIcon color="var(--pf-global--success-color--100)" />,
                  statusText: 'Healthy',
              };
    }

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

export default ComplianceClusterStatus;
