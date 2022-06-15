import React from 'react';
import { Flex, FlexItem, Title, Button } from '@patternfly/react-core';

import LinkShim from 'Components/PatternFly/LinkShim';
import useURLSearch from 'hooks/useURLSearch';
import DeploymentsAtMostRiskTable from './DeploymentsAtMostRiskTable';
import WidgetCard from './WidgetCard';
import useDeploymentsAtRisk from '../hooks/useDeploymentsAtRisk';

function DeploymentsAtMostRisk() {
    const { searchFilter } = useURLSearch();
    const { deployments, loading, error } = useDeploymentsAtRisk(searchFilter);
    return (
        <WidgetCard
            isLoading={loading}
            error={error}
            header={
                <Flex direction={{ default: 'row' }}>
                    <FlexItem grow={{ default: 'grow' }}>
                        <Title headingLevel="h2">Deployments at most risk</Title>
                    </FlexItem>
                    <FlexItem>
                        <Button variant="secondary" component={LinkShim} href="/main/risk">
                            View All
                        </Button>
                    </FlexItem>
                </Flex>
            }
        >
            <DeploymentsAtMostRiskTable deployments={deployments} searchFilter={searchFilter} />
        </WidgetCard>
    );
}

export default DeploymentsAtMostRisk;
