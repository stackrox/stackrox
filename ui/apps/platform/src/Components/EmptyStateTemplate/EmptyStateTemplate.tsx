import React from 'react';
import type { ComponentType, PropsWithChildren, ReactElement, ReactNode } from 'react';
import {
    EmptyState,
    EmptyStateIcon,
    EmptyStateBody,
    EmptyStateHeader,
} from '@patternfly/react-core';
import { CubesIcon } from '@patternfly/react-icons';

export type EmptyStateTemplateProps = {
    children?: ReactNode;
    title: string;
    headingLevel: 'h1' | 'h2' | 'h3' | 'h4';
    icon?: ComponentType<PropsWithChildren<unknown>>;
    iconClassName?: string;
};

function EmptyStateTemplate({
    children,
    title,
    headingLevel,
    icon = CubesIcon,
    iconClassName = '',
}: EmptyStateTemplateProps): ReactElement {
    return (
        <EmptyState variant="lg">
            <EmptyStateHeader
                titleText={<>{title}</>}
                icon={<EmptyStateIcon className={iconClassName} icon={icon} />}
                headingLevel={headingLevel}
            />
            <EmptyStateBody>{children}</EmptyStateBody>
        </EmptyState>
    );
}

export default EmptyStateTemplate;
