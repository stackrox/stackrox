import React from 'react';
import { EmptyState, EmptyStateVariant, EmptyStateIcon, Title } from '@patternfly/react-core';
import { SearchIcon } from '@patternfly/react-icons';

function NoDataEmptyState() {
    return (
        <EmptyState className="pf-u-h-100" variant={EmptyStateVariant.xs}>
            <EmptyStateIcon className="pf-u-font-size-xl" icon={SearchIcon} />
            <Title headingLevel="h3" size="md">
                No data was found in the selected scope.
            </Title>
        </EmptyState>
    );
}

export default NoDataEmptyState;
