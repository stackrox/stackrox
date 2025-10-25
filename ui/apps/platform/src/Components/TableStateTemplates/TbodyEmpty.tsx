import type { ReactElement, ReactNode } from 'react';
import { Flex, Text } from '@patternfly/react-core';
import { FileAltIcon } from '@patternfly/react-icons';

import EmptyStateTemplate from 'Components/EmptyStateTemplate/EmptyStateTemplate';
import type { EmptyStateTemplateProps } from 'Components/EmptyStateTemplate/EmptyStateTemplate';

import { TbodyFullCentered } from './TbodyFullCentered';

export type TbodyEmptyProps = {
    colSpan: number;
    children?: ReactNode;
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
}: TbodyEmptyProps): ReactElement {
    return (
        <TbodyFullCentered colSpan={colSpan}>
            <EmptyStateTemplate headingLevel={headingLevel} title={title} icon={FileAltIcon}>
                <Flex direction={{ default: 'column' }}>
                    <Text>{message}</Text>
                    {children}
                </Flex>
            </EmptyStateTemplate>
        </TbodyFullCentered>
    );
}
