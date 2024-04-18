import React from 'react';
import { Text } from '@patternfly/react-core';
import { FileAltIcon } from '@patternfly/react-icons';

import EmptyStateTemplate, {
    EmptyStateTemplateProps,
} from 'Components/EmptyStateTemplate/EmptyStateTemplate';

import { TbodyFullCentered } from './TbodyFullCentered';

export type TbodyEmptyProps = {
    colSpan: number;
    children?: React.ReactNode;
    headingLevel?: EmptyStateTemplateProps['headingLevel'];
    title?: string;
    message?: string;
};

export function TbodyEmpty({
    colSpan,
    children,
    headingLevel = 'h2',
    title = 'No results found',
    message = 'There are currently no entities found in the system',
}: TbodyEmptyProps) {
    return (
        <TbodyFullCentered colSpan={colSpan}>
            <EmptyStateTemplate headingLevel={headingLevel} title={title} icon={FileAltIcon}>
                <Text>{message}</Text>
                {children}
            </EmptyStateTemplate>
        </TbodyFullCentered>
    );
}
