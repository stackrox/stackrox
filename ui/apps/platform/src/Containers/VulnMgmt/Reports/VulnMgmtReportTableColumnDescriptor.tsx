import React, { ReactElement } from 'react';
import { Button, ButtonVariant, Flex, FlexItem, Tooltip } from '@patternfly/react-core';
import { ExclamationCircleIcon } from '@patternfly/react-icons';

import LinkShim from 'Components/PatternFly/LinkShim';
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
                <Button variant={ButtonVariant.link} isInline component={LinkShim} href={url}>
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
        accessor: 'lastRunStatus',
        Cell: ({ value }): ReactElement => {
            const lastRunTime = value?.lastRunTime;

            if (value?.reportStatus === 'FAILURE') {
                return (
                    <Tooltip
                        content={
                            <div>
                                <div>{value?.errorMsg || 'Unrecognized error'}</div>
                                <div>
                                    (attempted at:{' '}
                                    {lastRunTime ? (
                                        <DateTimeFormat time={lastRunTime} isInline />
                                    ) : (
                                        ''
                                    )}
                                    )
                                </div>
                            </div>
                        }
                        isContentLeftAligned
                        maxWidth="24rem"
                    >
                        <Flex
                            alignItems={{ default: 'alignItemsCenter' }}
                            spaceItems={{ default: 'spaceItemsXs' }}
                            display={{ default: 'inlineFlex' }}
                        >
                            <FlexItem>
                                <ExclamationCircleIcon className="pf-u-danger-color-100" />
                            </FlexItem>
                            <FlexItem>Error</FlexItem>
                        </Flex>
                    </Tooltip>
                );
            }

            return lastRunTime ? <DateTimeFormat time={lastRunTime} /> : <span>Not run yet</span>;
        },
    },
];

export default VulnMgmtReportTableColumnDescriptor;
