import React, { useMemo, useState } from 'react';
import { Divider, Flex, FlexItem, Stack, StackItem } from '@patternfly/react-core';

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
    onExternalIPSelect: (externalIP: string) => void;
};

function ExternalFlows({ deploymentId, scopeHierarchy, onExternalIPSelect }: ExternalFlowsProps) {
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
                        <IPMatchFilter onSearch={onSearch} />
                    </FlexItem>
                </Flex>
            </StackItem>
            <Divider component="hr" />
            <StackItem isFilled style={{ overflow: 'auto' }}>
                <ExternalIpsTable
                    scopeHierarchy={scopeHierarchy}
                    advancedFilters={advancedFilters}
                    onExternalIPSelect={onExternalIPSelect}
                />
            </StackItem>
        </Stack>
    );
}

export default ExternalFlows;
