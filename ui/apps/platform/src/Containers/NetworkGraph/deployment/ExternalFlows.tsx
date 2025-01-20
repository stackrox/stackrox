import { Divider, Flex, FlexItem, Stack, StackItem } from '@patternfly/react-core';
import React, { useState } from 'react';
import AdvancedFlowsFilter, {
    defaultAdvancedFlowsFilters,
} from '../common/AdvancedFlowsFilter/AdvancedFlowsFilter';
import { AdvancedFlowsFilterType } from '../common/AdvancedFlowsFilter/types';
import { getAllUniquePorts } from '../utils/flowUtils';
import IPMatchFilter, { MatchType } from '../common/IPMatchFilter';

type InternalFlowsProps = {
    deploymentId: string;
};

// eslint-disable-next-line @typescript-eslint/no-unused-vars
function ExternalFlows({ deploymentId }: InternalFlowsProps) {
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    const [selectedMatchType, setSelectedMatchType] = useState<MatchType>('Equals');
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    const [selectedExternalIP, setSelectedExternalIP] = useState('');
    const [advancedFilters, setAdvancedFilters] = React.useState<AdvancedFlowsFilterType>(
        defaultAdvancedFlowsFilters
    );

    // TODO: Fetch external IPs connected to a deployment using the deploymentID

    const onSearch = ({ matchType, externalIP }) => {
        setSelectedMatchType(matchType);
        setSelectedExternalIP(externalIP);
    };

    // TODO: Show all unique ports
    const allUniquePorts = getAllUniquePorts([]);

    // TODO: Filter network flows based on the match type and external IP

    return (
        <Stack>
            <StackItem>
                <Flex direction={{ default: 'row' }}>
                    <FlexItem flex={{ default: 'flex_1' }}>
                        <IPMatchFilter onSearch={onSearch} />
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
            <Divider component="hr" className="pf-v5-u-py-md" />
            <StackItem isFilled style={{ overflow: 'auto' }}></StackItem>
        </Stack>
    );
}

export default ExternalFlows;
