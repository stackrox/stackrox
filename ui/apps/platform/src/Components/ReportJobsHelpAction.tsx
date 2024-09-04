import React from 'react';
import { Popover, TabAction } from '@patternfly/react-core';
import { HelpIcon } from '@patternfly/react-icons';

import { systemConfigPath } from 'routePaths';
import usePermissions from 'hooks/usePermissions';
import PopoverBodyContent from './PopoverBodyContent';
import ExternalLink from './PatternFly/IconText/ExternalLink';

type ReportJobsHelpActionProps = {
    reportType: 'Vulnerability' | 'Scan schedule';
};

function ReportJobsHelpAction({ reportType }: ReportJobsHelpActionProps) {
    const { hasReadWriteAccess } = usePermissions();
    const hasAdministrationReadWriteAccess = hasReadWriteAccess('Administration');

    const bodyContent = hasAdministrationReadWriteAccess ? (
        <>
            This function displays requested jobs from various users, including their statuses. You
            can monitor and audit both active and past job requests. For an optimal experience, we
            recommend configuring the{' '}
            <ExternalLink>
                <a href={systemConfigPath} target="_blank" rel="noreferrer">
                    {reportType} report retention limit
                </a>
            </ExternalLink>{' '}
            to suit your needs. Report jobs will remain in the system until they exceed the limit
            you set.
        </>
    ) : (
        <>
            This function shows requested jobs from various users, including their statuses. While
            you can track and audit your active and past job requests, any changes to the{' '}
            <b>{reportType} report retention limit</b> must be configured by your administrator to
            ensure an optimal experience. Reports will remain in the system until they exceed the
            retention limit set by the administrator.
        </>
    );

    return (
        <Popover
            aria-label="All report jobs help text"
            bodyContent={
                <PopoverBodyContent headerContent="All report jobs" bodyContent={bodyContent} />
            }
            enableFlip
            position="top"
        >
            <TabAction aria-label="Help for report jobs tab">
                <HelpIcon />
            </TabAction>
        </Popover>
    );
}

export default ReportJobsHelpAction;
