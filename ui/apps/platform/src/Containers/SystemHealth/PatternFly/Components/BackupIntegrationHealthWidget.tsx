import React, { useState, useEffect, ReactElement } from 'react';

import { fetchBackupIntegrationsHealth } from 'services/IntegrationHealthService';
import { fetchBackupIntegrations } from 'services/BackupIntegrationsService';
import integrationsList from 'Containers/Integrations/utils/integrationsList';
import IntegrationHealthWidgetVisual from './IntegrationHealthWidgetVisual';
import { mergeIntegrationResponses, IntegrationMergedItem } from '../utils/integrations';

type WidgetProps = {
    pollingCount: number;
};

const BackupIntegrationHealthWidget = ({ pollingCount }: WidgetProps): ReactElement => {
    const [backupsMerged, setBackupsMerged] = useState([] as IntegrationMergedItem[]);
    const [backupsRequestHasError, setBackupsRequestHasError] = useState(false);

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
                setBackupsRequestHasError(false);
            })
            .catch(() => {
                setBackupsMerged([]);
                setBackupsRequestHasError(true);
            });
    }, [pollingCount]);

    return (
        <IntegrationHealthWidgetVisual
            id="backup-integrations"
            integrationText="Backup Integrations"
            integrationsMerged={backupsMerged}
            requestHasError={backupsRequestHasError}
        />
    );
};

export default BackupIntegrationHealthWidget;
