import React, { useMemo, useState } from 'react';
import { Divider, Flex, FlexItem, Stack, StackItem } from '@patternfly/react-core';

import { UseURLPaginationResult } from 'hooks/useURLPagination';
import { UseUrlSearchReturn } from 'hooks/useURLSearch';

import AdvancedFlowsFilter, {
    defaultAdvancedFlowsFilters,
} from '../common/AdvancedFlowsFilter/AdvancedFlowsFilter';
import { AdvancedFlowsFilterType } from '../common/AdvancedFlowsFilter/types';
import { getAllUniquePorts } from '../utils/flowUtils';
import IPMatchFilter, { MatchType } from '../common/IPMatchFilter';
import ExternalIpsTable from '../external/ExternalIpsTable';
import { NetworkScopeHierarchy } from '../types/networkScopeHierarchy';

type ExternalFlowsFilter = {
    matchType: MatchType;
    externalIP: string;
};

type ExternalFlowsProps = {
    deploymentId: string;
    scopeHierarchy: NetworkScopeHierarchy;
    urlSearchFiltering: UseUrlSearchReturn;
    urlPagination: UseURLPaginationResult;
    onExternalIPSelect: (externalIP: string) => void;
};

function ExternalFlows({
    deploymentId,
    scopeHierarchy,
    urlSearchFiltering,
    urlPagination,
    onExternalIPSelect,
}: ExternalFlowsProps) {
    const { searchFilter, setSearchFilter } = urlSearchFiltering;
    const [appliedFilter, setAppliedFilter] = useState<ExternalFlowsFilter>({
        matchType: 'Equals',
        externalIP: '',
    });

    const onSearch = ({ matchType, externalIP }) => {
        setAppliedFilter({ matchType, externalIP });
    };

    const advancedFilters = useMemo(
        () => ({
            'Deployment ID': deploymentId,
            'External Source Address': appliedFilter.externalIP,
        }),
        [appliedFilter.externalIP, deploymentId]
    );

    return (
        <Stack>
            <StackItem>
                <Flex direction={{ default: 'row' }}>
                    <FlexItem flex={{ default: 'flex_1' }}>
                        <IPMatchFilter
                            searchFilter={searchFilter}
                            setSearchFilter={setSearchFilter}
                        />
                    </FlexItem>
                </Flex>
            </StackItem>
            <Divider component="hr" />
            <StackItem isFilled style={{ overflow: 'auto' }}>
                <ExternalIpsTable
                    scopeHierarchy={scopeHierarchy}
                    onExternalIPSelect={onExternalIPSelect}
                    urlPagination={urlPagination}
                    urlSearchFiltering={urlSearchFiltering}
                />
            </StackItem>
        </Stack>
    );
}

export default ExternalFlows;
