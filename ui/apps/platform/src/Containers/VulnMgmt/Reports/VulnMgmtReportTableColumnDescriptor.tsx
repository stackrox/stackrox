import React, { ReactElement } from 'react';
import { Link } from 'react-router-dom';
import { Button, ButtonVariant } from '@patternfly/react-core';

import DateTimeFormat from 'Components/PatternFly/DateTimeFormat';
import FixabilityLabelsList from 'Components/PatternFly/FixabilityLabelsList';
import SeverityLabelsList from 'Components/PatternFly/SeverityLabelsList';
import { vulnManagementReportsPath } from 'routePaths';

const VulnMgmtReportTableColumnDescriptor = [
    {
        Header: 'Report',
        accessor: 'report.name',
        sortField: 'Report Name',
        Cell: ({ original }) => {
            const url = `${vulnManagementReportsPath}/${original.id as string}`;
            return (
                <Button
                    variant={ButtonVariant.link}
                    isInline
                    component={(props) => <Link {...props} to={url} />}
                >
                    {original?.name}
                </Button>
            );
        },
    },
    {
        Header: 'Description',
        accessor: 'description',
        Cell: ({ value }): ReactElement => {
            return <span>{value}</span>;
        },
    },
    {
        Header: 'CVE fixability type',
        accessor: 'vulnReportFilters.fixability',
        Cell: ({ value }): ReactElement => <FixabilityLabelsList fixability={value} />,
    },
    {
        Header: 'CVE severities',
        accessor: 'vulnReportFilters.severities',
        Cell: ({ value }): ReactElement => <SeverityLabelsList severities={value} />,
    },
    {
        Header: 'Last run',
        accessor: 'runStatus',
        Cell: ({ value }): ReactElement => {
            const lastRunAttempt = value?.lastTimeRun;
            return lastRunAttempt ? (
                <DateTimeFormat time={lastRunAttempt} />
            ) : (
                <span>Not run yet</span>
            );
        },
    },
];

export default VulnMgmtReportTableColumnDescriptor;
