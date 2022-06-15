import React, { ReactNode } from 'react';
import { Flex } from '@patternfly/react-core';
import { ExclamationCircleIcon } from '@patternfly/react-icons';
import { SVGIconProps } from '@patternfly/react-icons/dist/js/createIcon';

import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';

export type WidgetErrorEmptyStateProps = {
    children: ReactNode;
    height: `${number}px`;
    title: string;
};

function ErrorIcon(props: SVGIconProps) {
    return (
        <ExclamationCircleIcon
            {...props}
            style={{ color: 'var(--pf-global--danger-color--200)' }}
        />
    );
}

export default function WidgetErrorEmptyState({
    children,
    title,
    height,
}: WidgetErrorEmptyStateProps) {
    return (
        <>
            <Flex
                alignContent={{ default: 'alignContentCenter' }}
                justifyContent={{ default: 'justifyContentCenter' }}
                style={{ height }}
            >
                <EmptyStateTemplate icon={ErrorIcon} title={title} headingLevel="h3">
                    {children}
                </EmptyStateTemplate>
            </Flex>
        </>
    );
}
