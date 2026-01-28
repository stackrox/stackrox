import { Link } from 'react-router-dom-v5-compat';
import { Flex, FlexItem, Title } from '@patternfly/react-core';

import WidgetCard from 'Components/PatternFly/WidgetCard';
import useURLSearch from 'hooks/useURLSearch';
import { riskBasePath } from 'routePaths';
import { getUrlQueryStringForSearchFilter } from 'utils/searchUtils';
import DeploymentsAtMostRiskTable from './DeploymentsAtMostRiskTable';
import useDeploymentsAtRisk from '../hooks/useDeploymentsAtRisk';
import NoDataEmptyState from './NoDataEmptyState';

function DeploymentsAtMostRisk() {
    const { searchFilter } = useURLSearch();
    const { data: deployments, isLoading, error } = useDeploymentsAtRisk(searchFilter);
    const urlQueryString = getUrlQueryStringForSearchFilter(searchFilter);
    return (
        <WidgetCard
            isLoading={isLoading}
            error={error}
            header={
                <Flex
                    direction={{ default: 'row' }}
                    alignItems={{ default: 'alignItemsCenter' }}
                    justifyContent={{ default: 'justifyContentSpaceBetween' }}
                >
                    <FlexItem>
                        <Title headingLevel="h2">Deployments at most risk</Title>
                    </FlexItem>
                    <FlexItem>
                        <Link to={`${riskBasePath}?${urlQueryString}`}>View all</Link>
                    </FlexItem>
                </Flex>
            }
        >
            {deployments && deployments.length > 0 ? (
                <DeploymentsAtMostRiskTable deployments={deployments} searchFilter={searchFilter} />
            ) : (
                <NoDataEmptyState />
            )}
        </WidgetCard>
    );
}

export default DeploymentsAtMostRisk;
