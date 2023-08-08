import React from 'react';
import { EmptyState, EmptyStateVariant, EmptyStateIcon } from '@patternfly/react-core';
import { SearchIcon } from '@patternfly/react-icons';

function NoDataEmptyState() {
    return (
        <EmptyState className="pf-u-h-100" variant={EmptyStateVariant.xs}>
            <EmptyStateIcon className="pf-u-font-size-xl" icon={SearchIcon} />
            <div>No data was found in the selected resources.</div>
        </EmptyState>
    );
}

export default NoDataEmptyState;
