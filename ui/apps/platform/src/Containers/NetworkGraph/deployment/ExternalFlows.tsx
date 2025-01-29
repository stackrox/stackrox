import React, { useMemo, useState } from 'react';
import { Button, Divider, Flex, FlexItem, Stack, StackItem } from '@patternfly/react-core';
import { PlusCircleIcon } from '@patternfly/react-icons';

import IPMatchFilter, { MatchType } from '../common/IPMatchFilter';
import CIDRFormModalButton from '../components/CIDRFormModalButton';
import { useCIDRFormModal } from '../components/CIDRFormModalProvider';
import ExternalIpsTable from '../external/ExternalIpsTable';
import { NetworkScopeHierarchy } from '../types/networkScopeHierarchy';

type ExternalFlowsFilter = {
    matchType: MatchType;
    externalIP: string;
};

type InternalFlowsProps = {
    deploymentId: string;
    scopeHierarchy: NetworkScopeHierarchy;
};

function ExternalFlows({ deploymentId, scopeHierarchy }: InternalFlowsProps) {
    const { setInitialCIDRFormValue } = useCIDRFormModal();

    const [tempFilter, setTempFilter] = useState<ExternalFlowsFilter>({
        matchType: 'Equals',
        externalIP: '',
    });
    const [appliedFilter, setAppliedFilter] = useState<ExternalFlowsFilter>({
        matchType: 'Equals',
        externalIP: '',
    });

    const clearFilters = () => {
        setTempFilter((prevFilter) => ({
            ...prevFilter,
            externalIP: '',
        }));
        setAppliedFilter((prevFilter) => ({
            ...prevFilter,
            externalIP: '',
        }));
    };

    const onCIDRFormModalOpen = () => {
        setInitialCIDRFormValue(tempFilter.externalIP);
    };

    const isFiltered = appliedFilter.externalIP !== '';

    const advancedFilters = useMemo(
        () => ({
            'Deployment ID': deploymentId,
            'External Source Address': appliedFilter.externalIP,
        }),
        [appliedFilter.externalIP, deploymentId]
    );

    return (
        <Stack hasGutter>
            <StackItem>
                <Flex direction={{ default: 'row' }}>
                    <FlexItem flex={{ default: 'flex_1' }}>
                        <IPMatchFilter
                            filter={tempFilter}
                            onChange={setTempFilter}
                            onSearch={setAppliedFilter}
                            onClear={clearFilters}
                        />
                    </FlexItem>
                </Flex>
            </StackItem>
            {isFiltered && (
                <StackItem>
                    <Flex direction={{ default: 'row' }}>
                        <FlexItem flex={{ default: 'flex_1' }}>
                            <CIDRFormModalButton
                                variant="link"
                                icon={<PlusCircleIcon />}
                                isInline
                                onOpenCallback={onCIDRFormModalOpen}
                            >
                                Add as CIDR block
                            </CIDRFormModalButton>
                        </FlexItem>
                        <FlexItem>
                            <Button variant="link" isInline onClick={clearFilters}>
                                Clear filters
                            </Button>
                        </FlexItem>
                    </Flex>
                </StackItem>
            )}

            <Divider component="hr" />
            <StackItem isFilled style={{ overflow: 'auto' }}>
                <ExternalIpsTable
                    scopeHierarchy={scopeHierarchy}
                    advancedFilters={advancedFilters}
                    setSelectedEntity={() => {
                        // TODO: Set up routing so this will take you to the external ip detail view
                    }}
                />
            </StackItem>
        </Stack>
    );
}

export default ExternalFlows;
