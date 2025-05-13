import React, { ReactElement, ReactNode, useEffect, useState } from 'react';
import { Bullseye, Spinner } from '@patternfly/react-core';

/*
import { clustersBasePath, getIsRoutePathRendered } from 'routePaths';
*/
import usePermissions from 'hooks/usePermissions';
import useFeatureFlags from 'hooks/useFeatureFlags';
import { fetchSystemConfig } from 'services/SystemConfigService';
import { SystemConfig } from 'types/config.proto';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import SystemConfigDetails from './Details/SystemConfigDetails';
import SystemConfigForm from './Form/SystemConfigForm';

const SystemConfigPage = (): ReactElement => {
    /*
    const { hasReadAccess, hasReadWriteAccess } = usePermissions();
    */
    const { hasReadWriteAccess } = usePermissions();
    const hasReadWriteAccessForAdministration = hasReadWriteAccess('Administration');
    /*
    const isClustersRoutePathRendered = getIsRoutePathRendered({
        hasReadAccess,
        isFeatureFlagEnabled,
    })(clustersBasePath);
    */
    const isClustersRoutePathRendered = true; // TODO replace with the preceding after #2105 has been merged

    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isCustomizingPlatformComponentsEnabled = isFeatureFlagEnabled(
        'ROX_CUSTOMIZABLE_PLATFORM_COMPONENTS'
    );

    const [isEditing, setIsEditing] = useState(false);

    const [systemConfig, setSystemConfig] = useState<SystemConfig | null>(null);
    const [isLoading, setIsLoading] = useState(false);
    const [errorMessage, setErrorMessage] = useState('');

    useEffect(() => {
        setIsLoading(true);
        fetchSystemConfig()
            .then((data) => {
                setSystemConfig(data);
                setErrorMessage('');
            })
            .catch((error) => {
                setSystemConfig(null);
                setErrorMessage(getAxiosErrorMessage(error));
            })
            .finally(() => {
                setIsLoading(false);
            });
    }, []);

    function onEditConfig() {
        setIsEditing(true);
    }

    function onCancelEditConfig() {
        setIsEditing(false);
    }

    let content: ReactNode = null;

    if (isLoading) {
        content = (
            <Bullseye>
                <Spinner />
            </Bullseye>
        );
    } else if (!isEditing || !systemConfig) {
        return (
            <SystemConfigDetails
                systemConfig={systemConfig}
                errorMessage={errorMessage}
                onEditConfig={onEditConfig}
                isClustersRoutePathRendered={isClustersRoutePathRendered}
                hasReadWriteAccessForAdministration={hasReadWriteAccessForAdministration}
                isCustomizingPlatformComponentsEnabled={isCustomizingPlatformComponentsEnabled}
            />
        );
    } else {
        return (
            <SystemConfigForm
                systemConfig={systemConfig}
                setSystemConfig={setSystemConfig}
                onCancelEditConfig={onCancelEditConfig}
            />
        );
    }

    return content;
};

export default SystemConfigPage;
