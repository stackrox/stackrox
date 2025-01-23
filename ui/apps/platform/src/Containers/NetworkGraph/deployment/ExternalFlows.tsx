import React, { useState } from 'react';
import { Button, Divider, Flex, FlexItem, Stack, StackItem } from '@patternfly/react-core';
import { PlusCircleIcon } from '@patternfly/react-icons';

import AdvancedFlowsFilter, {
    defaultAdvancedFlowsFilters,
} from '../common/AdvancedFlowsFilter/AdvancedFlowsFilter';
import { AdvancedFlowsFilterType } from '../common/AdvancedFlowsFilter/types';
import { getAllUniquePorts } from '../utils/flowUtils';
import IPMatchFilter, { MatchType } from '../common/IPMatchFilter';
import CIDRFormModalButton from '../components/CIDRFormModalButton';
import { useCIDRFormModal } from '../components/CIDRFormModalProvider';

type ExternalFlowsFilter = {
    matchType: MatchType;
    externalIP: string;
};

type InternalFlowsProps = {
    deploymentId: string;
};

// eslint-disable-next-line @typescript-eslint/no-unused-vars
function ExternalFlows({ deploymentId }: InternalFlowsProps) {
    const { setInitialCIDRFormValue } = useCIDRFormModal();

    const [tempFilter, setTempFilter] = useState<ExternalFlowsFilter>({
        matchType: 'Equals',
        externalIP: '',
    });
    const [appliedFilter, setAppliedFilter] = useState<ExternalFlowsFilter>({
        matchType: 'Equals',
        externalIP: '',
    });
    const [advancedFilters, setAdvancedFilters] = React.useState<AdvancedFlowsFilterType>(
        defaultAdvancedFlowsFilters
    );

    // TODO: Fetch external IPs connected to a deployment using the deploymentID

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

    // TODO: Show all unique ports
    const allUniquePorts = getAllUniquePorts([]);
    const isFiltered = appliedFilter.externalIP !== '';

    // TODO: Filter network flows based on the match type and external IP

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
                    <FlexItem>
                        <AdvancedFlowsFilter
                            filters={advancedFilters}
                            setFilters={setAdvancedFilters}
                            allUniquePorts={allUniquePorts}
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
            <StackItem isFilled style={{ overflow: 'auto' }}></StackItem>
        </Stack>
    );
}

export default ExternalFlows;
