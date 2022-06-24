import React from 'react';
import { gql, useQuery } from '@apollo/client';
import { Flex, FlexItem, Title, Button, Divider } from '@patternfly/react-core';

import LinkShim from 'Components/PatternFly/LinkShim';
import useURLSearch from 'hooks/useURLSearch';
import { violationsBasePath } from 'routePaths';
import { SearchFilter } from 'types/search';
import { DeploymentAlert } from 'types/alert.proto';
import { getQueryString } from 'utils/queryStringUtils';
import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';

import { severities } from 'constants/severities';
import WidgetCard from './WidgetCard';
import MostRecentViolations from './MostRecentViolations';
import PolicyViolationTiles from './PolicyViolationTiles';
import useAlertGroups from '../hooks/useAlertGroups';

function getViewAllLink(searchFilter: SearchFilter) {
    const queryString = getQueryString({
        s: {
            ...searchFilter,
        },
        sortOption: { field: 'Severity', direction: 'desc' },
    });
    return `${violationsBasePath}${queryString}`;
}

export const mostRecentAlertsQuery = gql`
    query mostRecentAlerts($query: String) {
        alerts: violations(
            query: $query
            pagination: { limit: 3, sortOption: { field: "Violation Time", reversed: true } }
        ) {
            id
            time
            deployment {
                clusterName
                namespace
                name
            }
            policy {
                name
                severity
            }
        }
    }
`;

function ViolationsByPolicySeverity() {
    const { searchFilter } = useURLSearch();
    const countsFilterQuery = getRequestQueryStringForSearchFilter(searchFilter);
    const mostRecentFilterQuery = getRequestQueryStringForSearchFilter({
        ...searchFilter,
        Severity: severities.CRITICAL_SEVERITY,
    });
    const {
        data: alertCountData,
        loading: alertCountLoading,
        error: alertCountError,
    } = useAlertGroups(countsFilterQuery);
    const {
        data: currentRecentAlertsData,
        previousData: previousRecentAlertsData,
        loading: recentAlertsLoading,
        error: recentAlertsError,
    } = useQuery<{ alerts: DeploymentAlert[] }>(mostRecentAlertsQuery, {
        variables: { query: mostRecentFilterQuery },
    });

    const severityCounts = (alertCountData && alertCountData[0]?.counts) ?? [];
    const recentAlertsData = currentRecentAlertsData || previousRecentAlertsData;

    const counts = {
        LOW_SEVERITY: 0,
        MEDIUM_SEVERITY: 0,
        HIGH_SEVERITY: 0,
        CRITICAL_SEVERITY: 0,
    };
    let totalCount = 0;
    severityCounts.forEach(({ severity, count }) => {
        counts[severity] = parseInt(count, 10);
        totalCount += counts[severity];
    });

    return (
        <WidgetCard
            isLoading={
                alertCountLoading || recentAlertsLoading || !alertCountData || !recentAlertsData
            }
            error={alertCountError || recentAlertsError}
            header={
                <Flex direction={{ default: 'row' }}>
                    <FlexItem grow={{ default: 'grow' }}>
                        <Title headingLevel="h2">
                            {totalCount === 1
                                ? `1 policy violation by severity`
                                : `${totalCount} policy violations by severity`}
                        </Title>
                    </FlexItem>
                    <FlexItem>
                        <Button
                            variant="secondary"
                            component={LinkShim}
                            href={getViewAllLink(searchFilter)}
                        >
                            View all
                        </Button>
                    </FlexItem>
                </Flex>
            }
        >
            {severityCounts && recentAlertsData && (
                <>
                    <PolicyViolationTiles searchFilter={searchFilter} counts={counts} />
                    <Divider component="div" className="pf-u-my-lg" />
                    <MostRecentViolations alerts={recentAlertsData.alerts} />
                </>
            )}
        </WidgetCard>
    );
}

export default ViolationsByPolicySeverity;
