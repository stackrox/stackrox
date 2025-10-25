import { Button, Text } from '@patternfly/react-core';
import type { ButtonProps } from '@patternfly/react-core';
import { SearchIcon } from '@patternfly/react-icons';

import EmptyStateTemplate from 'Components/EmptyStateTemplate';
import type { EmptyStateTemplateProps } from 'Components/EmptyStateTemplate/EmptyStateTemplate';

import { TbodyFullCentered } from './TbodyFullCentered';

export type TbodyFilteredEmptyProps = {
    colSpan: number;
    onClearFilters?: ButtonProps['onClick'];
    headingLevel?: EmptyStateTemplateProps['headingLevel'];
    title?: string;
    message?: string;
};

export function TbodyFilteredEmpty({
    colSpan,
    onClearFilters,
    headingLevel = 'h2',
    title = 'No results found',
    message = 'No results were found with the applied filters.',
}: TbodyFilteredEmptyProps) {
    return (
        <TbodyFullCentered colSpan={colSpan}>
            <EmptyStateTemplate title={title} headingLevel={headingLevel} icon={SearchIcon}>
                <Text>{message}</Text>
                <Button variant="link" onClick={onClearFilters}>
                    Clear filters
                </Button>
            </EmptyStateTemplate>
        </TbodyFullCentered>
    );
}
