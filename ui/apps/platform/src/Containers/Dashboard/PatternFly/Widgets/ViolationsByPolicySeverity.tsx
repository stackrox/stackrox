import React from 'react';
import { gql, useQuery } from '@apollo/client';
import { Flex, FlexItem, Title, Button, Divider } from '@patternfly/react-core';

import LinkShim from 'Components/PatternFly/LinkShim';
import useURLSearch from 'hooks/useURLSearch';
import { violationsBasePath } from 'routePaths';

import { SearchFilter } from 'types/search';
import { getQueryString } from 'utils/queryStringUtils';
import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
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

const mostRecentAlertsQuery = gql`
    query mostRecentAlerts($query: String) {
        violations(
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
    const query = getRequestQueryStringForSearchFilter(searchFilter);
    const {
        data: alertCountData,
        loading: alertCountLoading,
        error: alertCountError,
    } = useAlertGroups(query);
    const {
        data: currentRecentAlertsData,
        previousData: previousRecentAlertsData,
        loading: recentAlertsLoading,
        error: recentAlertsError,
    } = useQuery(mostRecentAlertsQuery);

    const severityCounts = (alertCountData && alertCountData[0]?.counts) ?? [];
    const recentAlertsData = currentRecentAlertsData || previousRecentAlertsData;

    const counts = {
        LOW_SEVERITY: 0,
        MEDIUM_SEVERITY: 0,
        HIGH_SEVERITY: 0,
        CRITICAL_SEVERITY: 0,
    };
    severityCounts.forEach(({ severity, count }) => {
        counts[severity] = parseInt(count, 10);
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
                        <Title headingLevel="h2">## policy violations by severity</Title>
                    </FlexItem>
                    <FlexItem>
                        <Button
                            variant="secondary"
                            component={LinkShim}
                            href={getViewAllLink(searchFilter)}
                        >
                            View All
                        </Button>
                    </FlexItem>
                </Flex>
            }
        >
            {severityCounts && recentAlertsData && (
                <>
                    <PolicyViolationTiles searchFilter={searchFilter} counts={counts} />
                    <Divider component='div' className="pf-u-my-lg"/>
                    <MostRecentViolations alerts={recentAlertsData} />
                </>
            )}
        </WidgetCard>
    );
}

export default ViolationsByPolicySeverity;
