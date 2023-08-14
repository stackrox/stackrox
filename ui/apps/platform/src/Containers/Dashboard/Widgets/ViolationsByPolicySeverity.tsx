import React from 'react';
import { gql, useQuery } from '@apollo/client';
import { Flex, FlexItem, Title, Button, Divider, Stack, StackItem } from '@patternfly/react-core';

import LinkShim from 'Components/PatternFly/LinkShim';
import WidgetCard from 'Components/PatternFly/WidgetCard';
import useURLSearch from 'hooks/useURLSearch';
import { violationsBasePath } from 'routePaths';
import { SearchFilter } from 'types/search';
import { Alert } from 'types/alert.proto';
import { getQueryString } from 'utils/queryStringUtils';
import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';

import { severities } from 'constants/severities';
import pluralize from 'pluralize';
import { ValueOf } from 'utils/type.utils';
import MostRecentViolations from './MostRecentViolations';
import PolicyViolationTiles from './PolicyViolationTiles';

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
                name
            }
            resource {
                resourceType
                name
            }
            policy {
                name
                severity
            }
        }
    }
`;

export const alertsBySeverityQuery = gql`
    query alertCountsBySeverity(
        $lowQuery: String
        $medQuery: String
        $highQuery: String
        $critQuery: String
    ) {
        ${severities.LOW_SEVERITY}: violationCount(query: $lowQuery)
        ${severities.MEDIUM_SEVERITY}: violationCount(query: $medQuery)
        ${severities.HIGH_SEVERITY}: violationCount(query: $highQuery)
        ${severities.CRITICAL_SEVERITY}: violationCount(query: $critQuery)
    }
`;

export type AlertCounts = {
    [severities.LOW_SEVERITY]: number;
    [severities.MEDIUM_SEVERITY]: number;
    [severities.HIGH_SEVERITY]: number;
    [severities.CRITICAL_SEVERITY]: number;
};

function searchQueryBySeverity(severity: ValueOf<typeof severities>, searchFilter: SearchFilter) {
    return getRequestQueryStringForSearchFilter({
        ...searchFilter,
        Severity: severity,
    });
}

function ViolationsByPolicySeverity() {
    const { searchFilter } = useURLSearch();
    const {
        data: alertCountData,
        loading: alertCountLoading,
        error: alertCountError,
    } = useQuery<AlertCounts>(alertsBySeverityQuery, {
        variables: {
            lowQuery: searchQueryBySeverity(severities.LOW_SEVERITY, searchFilter),
            medQuery: searchQueryBySeverity(severities.MEDIUM_SEVERITY, searchFilter),
            highQuery: searchQueryBySeverity(severities.HIGH_SEVERITY, searchFilter),
            critQuery: searchQueryBySeverity(severities.CRITICAL_SEVERITY, searchFilter),
        },
    });
    const {
        data: currentRecentAlertsData,
        previousData: previousRecentAlertsData,
        loading: recentAlertsLoading,
        error: recentAlertsError,
    } = useQuery<{ alerts: Alert[] }>(mostRecentAlertsQuery, {
        variables: { query: searchQueryBySeverity(severities.CRITICAL_SEVERITY, searchFilter) },
    });

    const recentAlertsData = currentRecentAlertsData || previousRecentAlertsData;

    const counts = {
        LOW_SEVERITY: 0,
        MEDIUM_SEVERITY: 0,
        HIGH_SEVERITY: 0,
        CRITICAL_SEVERITY: 0,
    };
    let totalCount = 0;
    Object.entries(alertCountData ?? {}).forEach(([severity, count]) => {
        counts[severity] = count;
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
                            {`${totalCount} policy ${pluralize(
                                'violation',
                                totalCount
                            )} by severity`}
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
            {alertCountData && recentAlertsData && (
                <Stack>
                    <PolicyViolationTiles searchFilter={searchFilter} counts={counts} />
                    <Divider component="div" className="pf-u-my-lg" />
                    <StackItem isFilled>
                        <MostRecentViolations alerts={recentAlertsData.alerts} />
                    </StackItem>
                </Stack>
            )}
        </WidgetCard>
    );
}

export default ViolationsByPolicySeverity;
