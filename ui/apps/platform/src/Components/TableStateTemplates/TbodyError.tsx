import React from 'react';
import { ExclamationCircleIcon } from '@patternfly/react-icons';

import EmptyStateTemplate from 'Components/EmptyStateTemplate';
import { EmptyStateTemplateProps } from 'Components/EmptyStateTemplate/EmptyStateTemplate';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { TbodyFullCentered } from './TbodyFullCentered';

export type TbodyErrorProps = {
    colSpan: number;
    error: unknown;
    headingLevel?: EmptyStateTemplateProps['headingLevel'];
    title?: string;
    message?: string;
};

export function TbodyError({
    colSpan,
    error,
    headingLevel = 'h2',
    title = 'An error has occurred. Try clearing any filters or refreshing the page',
    message,
}: TbodyErrorProps) {
    return (
        <TbodyFullCentered colSpan={colSpan}>
            <EmptyStateTemplate
                headingLevel={headingLevel}
                title={title}
                icon={ExclamationCircleIcon}
                iconClassName="pf-v5-u-danger-color-100"
            >
                {message ?? getAxiosErrorMessage(error)}
            </EmptyStateTemplate>
        </TbodyFullCentered>
    );
}
