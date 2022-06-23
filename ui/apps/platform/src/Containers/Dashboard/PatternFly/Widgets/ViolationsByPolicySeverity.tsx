import React from 'react';
import { Flex, FlexItem, Title, Button } from '@patternfly/react-core';

import LinkShim from 'Components/PatternFly/LinkShim';
import useURLSearch from 'hooks/useURLSearch';
import { violationsBasePath } from 'routePaths';

import { SearchFilter } from 'types/search';
import { getQueryString } from 'utils/queryStringUtils';
import WidgetCard from './WidgetCard';
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

function ViolationsByPolicySeverity() {
    const { searchFilter } = useURLSearch();

    return (
        <WidgetCard
            isLoading
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
            <PolicyViolationTiles />
            <MostRecentViolations />
        </WidgetCard>
    );
}

export default ViolationsByPolicySeverity;
