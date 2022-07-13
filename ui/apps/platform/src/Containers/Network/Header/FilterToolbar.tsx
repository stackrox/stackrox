import React from 'react';
import { Toolbar, ToolbarGroup, ToolbarItem, ToolbarItemVariant } from '@patternfly/react-core';

import NetworkSearch from './NetworkSearch';
import ClusterSelect from './ClusterSelect';
import NamespaceSelect from './NamespaceSelect';
import TimeWindowSelector from './TimeWindowSelector';

interface FilterToolbarProps {
    isDisabled: boolean;
}

function FilterToolbar({ isDisabled }: FilterToolbarProps) {
    return (
        <Toolbar
            data-testid="network-graph-toolbar"
            className="pf-u-px-md pf-u-px-lg-on-xl pf-u-py-sm"
        >
            <ToolbarGroup spacer={{ default: 'spacerNone' }}>
                <ToolbarItem>
                    <ClusterSelect isDisabled={isDisabled} />
                </ToolbarItem>
                <ToolbarItem>
                    <NamespaceSelect isDisabled={isDisabled} />
                </ToolbarItem>
                <ToolbarItem variant={ToolbarItemVariant.separator} />
                <ToolbarItem className="pf-u-flex-grow-1">
                    <NetworkSearch isDisabled={isDisabled} />
                </ToolbarItem>
                <ToolbarItem>
                    <TimeWindowSelector isDisabled={isDisabled} />
                </ToolbarItem>
            </ToolbarGroup>
        </Toolbar>
    );
}

export default FilterToolbar;
