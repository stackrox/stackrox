import React from 'react';
import { Link } from 'react-router-dom';
import { Popover, TabAction } from '@patternfly/react-core';
import { HelpIcon } from '@patternfly/react-icons';

import { systemConfigPath } from 'routePaths';
import PopoverBodyContent from './PopoverBodyContent';

type ReportJobsHelpActionProps = {
    reportType: 'Vulnerability' | 'Scan schedule';
};

function ReportJobsHelpAction({ reportType }: ReportJobsHelpActionProps) {
    return (
        <Popover
            aria-label="All report jobs help text"
            bodyContent={
                <PopoverBodyContent
                    headerContent="All report jobs"
                    bodyContent={
                        <>
                            This function displays the requested jobs from different users and
                            includes their statuses accordingly. While the function provides the
                            ability to monitor and audit your active and past requested jobs, we
                            suggest configuring the{' '}
                            <Link to={systemConfigPath}>{reportType} report retention limit</Link>{' '}
                            based on your needs in order to ensure optimal user experience. All the
                            report jobs will be kept in your system until they exceed the limit set
                            by you.
                        </>
                    }
                />
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
