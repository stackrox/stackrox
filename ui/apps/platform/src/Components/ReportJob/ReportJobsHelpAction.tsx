import React from 'react';
import { Popover, TabAction } from '@patternfly/react-core';
import { HelpIcon } from '@patternfly/react-icons';

import { systemConfigPath } from 'routePaths';
import usePermissions from 'hooks/usePermissions';
import PopoverBodyContent from '../PopoverBodyContent';
import ExternalLink from '../PatternFly/IconText/ExternalLink';

type ReportJobsHelpActionProps = {
    reportType: 'Vulnerability' | 'Scan schedule';
};

function ReportJobsHelpAction({ reportType }: ReportJobsHelpActionProps) {
    const { hasReadWriteAccess } = usePermissions();
    const hasAdministrationReadWriteAccess = hasReadWriteAccess('Administration');

    const bodyContent = (
        <>
            <div>
                This function displays the requested jobs from different users and includes their
                statuses accordingly. While the function provides the ability to monitor and audit
                your active and past requested jobs, we suggest configuring the{' '}
                <strong>{reportType} report retention limit</strong> based on your needs in order to
                ensure optimal user experience. All the report jobs will be kept in your system
                until they exceed the limit set by you.
            </div>
            {hasAdministrationReadWriteAccess && (
                <div className="pf-v5-u-mt-sm">
                    <ExternalLink>
                        <a href={systemConfigPath} target="_blank" rel="noopener noreferrer">
                            System Configuration
                        </a>
                    </ExternalLink>
                </div>
            )}
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
