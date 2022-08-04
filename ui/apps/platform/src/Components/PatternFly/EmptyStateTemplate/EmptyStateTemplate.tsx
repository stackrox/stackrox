import React, { ReactElement, ReactNode } from 'react';
import {
    EmptyState,
    EmptyStateIcon,
    EmptyStateBody,
    EmptyStateVariant,
    Title,
} from '@patternfly/react-core';
import { CubesIcon } from '@patternfly/react-icons';

type EmptyStateTemplateProps = {
    children?: ReactNode;
    title: string;
    headingLevel: 'h1' | 'h2' | 'h3' | 'h4';
    icon?: React.ComponentType;
};

function EmptyStateTemplate({
    children,
    title,
    headingLevel,
    icon = CubesIcon,
}: EmptyStateTemplateProps): ReactElement {
    return (
        <EmptyState variant={EmptyStateVariant.large}>
            <EmptyStateIcon icon={icon} />
            <Title headingLevel={headingLevel} size="lg">
                {title}
            </Title>
            <EmptyStateBody>{children}</EmptyStateBody>
        </EmptyState>
    );
}

export default EmptyStateTemplate;
