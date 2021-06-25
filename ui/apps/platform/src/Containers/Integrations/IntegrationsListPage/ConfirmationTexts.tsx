import React, { ReactElement } from 'react';
import pluralize from 'pluralize';

export type ConfirmationTextProps = {
    numIntegrations: number;
};

export function DeleteAPITokensConfirmationText({
    numIntegrations,
}: ConfirmationTextProps): ReactElement {
    return (
        <div>
            Are you sure you want to revoke {numIntegrations} api&nbsp;
            {pluralize('token', numIntegrations)}?
        </div>
    );
}

export function DeleteClusterInitBundlesConfirmationText({
    numIntegrations,
}: ConfirmationTextProps): ReactElement {
    return (
        <div>
            Are you sure you want to revoke {numIntegrations} cluster init&nbsp;
            {pluralize('bundle', numIntegrations)}? Revoking a cluster init bundle will cause the
            StackRox services installed with it in clusters to lose connectivity.
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
