import React from 'react';
import type { ReactElement } from 'react';
import pluralize from 'pluralize';

export type ConfirmationTextProps = {
    numIntegrations: number;
};

export function DeleteAPITokensConfirmationText({
    numIntegrations,
}: ConfirmationTextProps): ReactElement {
    return (
        <div>
            Are you sure you want to revoke {numIntegrations} API&nbsp;
            {pluralize('token', numIntegrations)}?
        </div>
    );
}

export function DeleteIntegrationsConfirmationText({
    numIntegrations,
}: ConfirmationTextProps): ReactElement {
    return (
        <div>
            Are you sure you want to delete {numIntegrations}&nbsp;
            {pluralize('integration', numIntegrations)}?
        </div>
    );
}
