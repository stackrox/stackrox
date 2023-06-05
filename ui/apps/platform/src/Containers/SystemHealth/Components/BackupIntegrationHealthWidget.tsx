import React, { useState, useEffect, ReactElement } from 'react';

import { fetchBackupIntegrationsHealth } from 'services/IntegrationHealthService';
import { fetchBackupIntegrations } from 'services/BackupIntegrationsService';
import integrationsList from 'Containers/Integrations/utils/integrationsList';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import IntegrationHealthWidgetVisual from './IntegrationHealthWidgetVisual';
import { mergeIntegrationResponses, IntegrationMergedItem } from '../utils/integrations';

type WidgetProps = {
    pollingCount: number;
};

const BackupIntegrationHealthWidget = ({ pollingCount }: WidgetProps): ReactElement => {
    const [isFetching, setIsFetching] = useState(false);
    const [backupsMerged, setBackupsMerged] = useState([] as IntegrationMergedItem[]);
    const [errorMessageFetching, setErrorMessageFetching] = useState('');

    useEffect(() => {
        setIsFetching(true);
        Promise.all([fetchBackupIntegrationsHealth(), fetchBackupIntegrations()])
            .then(([integrationsHealth, externalBackups]) => {
                setBackupsMerged(
                    mergeIntegrationResponses(
                        integrationsHealth,
                        externalBackups,
                        integrationsList.backups
                    )
                );
                setErrorMessageFetching('');
            })
            .catch((error) => {
                setBackupsMerged([]);
                setErrorMessageFetching(getAxiosErrorMessage(error));
            })
            .finally(() => {
                setIsFetching(false);
            });
    }, [pollingCount]);
    const isFetchingInitialRequest = isFetching && pollingCount === 0;

    return (
        <IntegrationHealthWidgetVisual
            integrationText="Backup Integrations"
            integrationsMerged={backupsMerged}
            errorMessageFetching={errorMessageFetching}
            isFetchingInitialRequest={isFetchingInitialRequest}
        />
    );
};

export default BackupIntegrationHealthWidget;
