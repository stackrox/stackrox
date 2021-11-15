import React, { ReactElement } from 'react';
import { Link } from 'react-router-dom';
import dateFns from 'date-fns';
import { Button, ButtonVariant } from '@patternfly/react-core';

import dateTimeFormat from 'constants/dateTimeFormat';
import { vulnManagementReportsPath } from 'routePaths';

const VulnMgmtReportTableColumnDescriptor = [
    {
        Header: 'Report',
        accessor: 'report.name',
        sortField: 'Report',
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
        accessor: 'filter.fixability',
        Cell: ({ value }): ReactElement => {
            return <span>{value}</span>;
        },
    },
    {
        Header: 'CVE severities',
        accessor: 'filter.severities',
        sortField: 'CVE severities',
        Cell: ({ value }): ReactElement => {
            return <span>{value}</span>;
        },
    },
    {
        Header: 'Last run',
        accessor: 'runStatus.lastTimeRun',
        sortField: 'Last run',
        Cell: ({ value }): ReactElement => {
            return <span>{dateFns.format(value, dateTimeFormat)}</span>;
        },
    },
];

export default VulnMgmtReportTableColumnDescriptor;
