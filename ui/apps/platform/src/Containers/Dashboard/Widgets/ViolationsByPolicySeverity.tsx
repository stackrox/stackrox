import { Link } from 'react-router-dom-v5-compat';
import { Divider, Flex, FlexItem, Stack, StackItem, Title } from '@patternfly/react-core';

import WidgetCard from 'Components/PatternFly/WidgetCard';
import { filteredWorkflowViewKey } from 'Components/FilteredWorkflowViewSelector/useFilteredWorkflowViewURLState';
import { fullWorkflowView } from 'Components/FilteredWorkflowViewSelector/types';
import useURLSearch from 'hooks/useURLSearch';
import { violationsBasePath } from 'routePaths';
import type { SearchFilter } from 'types/search';
import { getQueryString } from 'utils/queryStringUtils';
import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';

import { severities } from 'constants/severities';
import pluralize from 'pluralize';
import MostRecentViolations from './MostRecentViolations';
import PolicyViolationTiles from './PolicyViolationTiles';
import useAlertCountsBySeverity from '../hooks/useAlertCountsBySeverity';
import useMostRecentAlerts from '../hooks/useMostRecentAlerts';

function getViewAllLink(searchFilter: SearchFilter) {
    const queryString = getQueryString({
        s: {
            ...searchFilter,
        },
        sortOption: { field: 'Severity', direction: 'desc' },
        [filteredWorkflowViewKey]: fullWorkflowView,
    });
    return `${violationsBasePath}${queryString}`;
}

function ViolationsByPolicySeverity() {
    const { searchFilter } = useURLSearch();
    const query = getRequestQueryStringForSearchFilter(searchFilter);
    const {
        data: alertCountData,
        isLoading: alertCountLoading,
        error: alertCountError,
    } = useAlertCountsBySeverity(query);
    const {
        data: recentAlertsData,
        isLoading: recentAlertsLoading,
        error: recentAlertsError,
    } = useMostRecentAlerts({
        ...searchFilter,
        Severity: severities.CRITICAL_SEVERITY,
    });

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
                <Flex
                    direction={{ default: 'row' }}
                    alignItems={{ default: 'alignItemsCenter' }}
                    justifyContent={{ default: 'justifyContentSpaceBetween' }}
                >
                    <FlexItem>
                        <Title headingLevel="h2">
                            {`${totalCount} policy ${pluralize(
                                'violation',
                                totalCount
                            )} by severity`}
                        </Title>
                    </FlexItem>
                    <FlexItem>
                        <Link to={getViewAllLink(searchFilter)}>View all</Link>
                    </FlexItem>
                </Flex>
            }
        >
            {alertCountData && recentAlertsData && (
                <Stack>
                    <PolicyViolationTiles searchFilter={searchFilter} counts={counts} />
                    <Divider component="div" className="pf-v6-u-py-md" />
                    <StackItem isFilled>
                        <MostRecentViolations alerts={recentAlertsData} />
                    </StackItem>
                </Stack>
            )}
        </WidgetCard>
    );
}

export default ViolationsByPolicySeverity;
