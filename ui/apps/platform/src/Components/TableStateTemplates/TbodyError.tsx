import React from 'react';
import { ExclamationCircleIcon } from '@patternfly/react-icons';

import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';
import { EmptyStateTemplateProps } from 'Components/PatternFly/EmptyStateTemplate/EmptyStateTemplate';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { TbodyFullCentered } from './TbodyFullCentered';

export type TbodyErrorProps = {
    colSpan: number;
    error: unknown;
    headingLevel?: EmptyStateTemplateProps['headingLevel'];
    message?: string;
};

export function TbodyError({
    colSpan,
    error,
    headingLevel = 'h2',
    message = 'An error has occurred. Try clearing any filters or refreshing the page',
}: TbodyErrorProps) {
    return (
        <TbodyFullCentered colSpan={colSpan}>
            <EmptyStateTemplate
                headingLevel={headingLevel}
                title={getAxiosErrorMessage(error)}
                icon={ExclamationCircleIcon}
                iconClassName="pf-v5-u-danger-color-100"
            >
                {message}
            </EmptyStateTemplate>
        </TbodyFullCentered>
    );
}
