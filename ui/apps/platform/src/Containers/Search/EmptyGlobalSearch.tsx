import React, { ReactElement, ReactNode } from 'react';
import {
    EmptyState,
    EmptyStateIcon,
    EmptyStateBody,
    EmptyStateVariant,
    Title,
} from '@patternfly/react-core';
import { CubesIcon } from '@patternfly/react-icons';

type EmptyGlobalSearchProps = {
    children: ReactNode;
    title: string;
};

function EmptyGlobalSearch({ children, title }: EmptyGlobalSearchProps): ReactElement {
    return (
        <EmptyState variant={EmptyStateVariant.large}>
            <EmptyStateIcon icon={CubesIcon} />
            <Title headingLevel="h1" size="lg">
                {title}
            </Title>
            <EmptyStateBody>{children}</EmptyStateBody>
        </EmptyState>
    );
}

export default EmptyGlobalSearch;
