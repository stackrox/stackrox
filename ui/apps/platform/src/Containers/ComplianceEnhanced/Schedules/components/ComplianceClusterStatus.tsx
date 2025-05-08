import React, { ReactElement } from 'react';
import { Button, Icon, Popover, SimpleList, SimpleListItem } from '@patternfly/react-core';
import { CheckCircleIcon, ExclamationCircleIcon } from '@patternfly/react-icons';

import IconText from 'Components/PatternFly/IconText/IconText';
import PopoverBodyContent from 'Components/PopoverBodyContent';

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
                  icon: (
                      <Icon>
                          <ExclamationCircleIcon color="var(--pf-v5-global--danger-color--100)" />
                      </Icon>
                  ),
                  statusText: 'Unhealthy',
              }
            : {
                  icon: (
                      <Icon>
                          <CheckCircleIcon color="var(--pf-v5-global--success-color--100)" />
                      </Icon>
                  ),
                  statusText: 'Healthy',
              };
    }

    function getErrorsList(errors: string[]): ReactElement {
        return (
            <SimpleList>
                {errors.map((error) => {
                    return <SimpleListItem key={error}>{error}</SimpleListItem>;
                })}
            </SimpleList>
        );
    }

    const statusObj = getClusterStatusObject(errors);

    return statusObj.statusText === 'Healthy' ? (
        <IconText icon={statusObj.icon} text={statusObj.statusText} />
    ) : (
        <Popover
            aria-label="Reveal errors"
            bodyContent={
                <PopoverBodyContent
                    headerContent={errors.length === 1 ? 'Error' : 'Errors'}
                    bodyContent={getErrorsList(errors)}
                />
            }
        >
            <Button variant="link" className="pf-v5-u-p-0">
                <IconText icon={statusObj.icon} text={statusObj.statusText} />
            </Button>
        </Popover>
    );
}

export default ComplianceClusterStatus;
