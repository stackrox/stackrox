import React, { useState, useEffect, ReactElement } from 'react';

import { fetchBackupIntegrationsHealth } from 'services/IntegrationHealthService';
import { fetchBackupIntegrations } from 'services/BackupIntegrationsService';
import integrationsList from 'Containers/Integrations/utils/integrationsList';
import IntegrationHealthWidgetVisual from './IntegrationHealthWidgetVisual';
import { mergeIntegrationResponses, IntegrationMergedItem } from '../utils/integrations';
import { getAxiosErrorMessage } from '../../../utils/responseErrorUtils';

type WidgetProps = {
    pollingCount: number;
};

const BackupIntegrationHealthWidget = ({ pollingCount }: WidgetProps): ReactElement => {
    const [backupsMerged, setBackupsMerged] = useState([] as IntegrationMergedItem[]);
    const [errorMessageFetching, setErrorMessageFetching] = useState('');

    useEffect(() => {
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
            });
    }, [pollingCount]);

    return (
        <IntegrationHealthWidgetVisual
            integrationText="Backup Integrations"
            integrationsMerged={backupsMerged}
            errorMessageFetching={errorMessageFetching}
        />
    );
};

export default BackupIntegrationHealthWidget;
