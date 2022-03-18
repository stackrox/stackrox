import React from 'react';
import { Divider, Flex, FlexItem } from '@patternfly/react-core';

import NetworkSearch from './NetworkSearch';
import ClusterSelect from './ClusterSelect';
import TimeWindowSelector from './TimeWindowSelector';

interface FilterToolbarProps {
    isDisabled: boolean;
}

function FilterToolbar({ isDisabled }: FilterToolbarProps) {
    // Note that the outermost element of this component has the "theme-light" className. This
    // is to prevent rendering the NetworkSearch component with dark mode styles, which are not supported
    // in the PatternFly UI. Once we have a pure PF equivalent of the NetworkSearch component we can remove this.
    return (
        <Flex
            data-testid="network-graph-toolbar"
            className="theme-light pf-u-px-lg pf-u-py-sm"
            direction={{ default: 'row' }}
            alignItems={{ default: 'alignItemsCenter' }}
        >
            <FlexItem flex={{ default: 'flexNone' }}>
                <ClusterSelect isDisabled={isDisabled} />
            </FlexItem>
            <Divider component="div" isVertical />
            <FlexItem flex={{ default: 'flex_1' }}>
                <NetworkSearch isDisabled={isDisabled} />
            </FlexItem>
            <FlexItem flex={{ default: 'flexNone' }}>
                <TimeWindowSelector isDisabled={isDisabled} />
            </FlexItem>
        </Flex>
    );
}

export default FilterToolbar;
