import React from 'react';
import {
    EmptyState,
    EmptyStateVariant,
    EmptyStateIcon,
    EmptyStateHeader,
    EmptyStateFooter,
} from '@patternfly/react-core';
import { SearchIcon } from '@patternfly/react-icons';

function NoDataEmptyState() {
    return (
        <EmptyState className="pf-v5-u-h-100" variant={EmptyStateVariant.xs}>
            <EmptyStateHeader
                icon={<EmptyStateIcon className="pf-v5-u-font-size-xl" icon={SearchIcon} />}
            />
            <EmptyStateFooter>
                <div>No data was found in the selected resources.</div>
            </EmptyStateFooter>
        </EmptyState>
    );
}

export default NoDataEmptyState;
