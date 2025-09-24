import React, { useState, useEffect } from 'react';
import type { ReactElement } from 'react';

import { Switch } from '@patternfly/react-core';
import {
    isAutoUpgradeSupported,
    getAutoUpgradeConfig,
    saveAutoUpgradeConfig,
} from 'services/ClustersService';
import type { AutoUpgradeConfig } from 'services/ClustersService';

function AutoUpgradeToggle(): ReactElement {
    const [autoUpgradeConfig, setAutoUpgradeConfig] = useState<AutoUpgradeConfig | null>(null);

    function fetchConfig(): void {
        getAutoUpgradeConfig()
            .then((config) => {
                setAutoUpgradeConfig(config);
            })
            .catch(() => {
                // TODO display message when there is a place for minor errors
            });
    }

    useEffect(() => {
        fetchConfig();
    }, []);

    if (!autoUpgradeConfig) {
        return <></>;
    }

    if (!isAutoUpgradeSupported(autoUpgradeConfig)) {
        return <>Automatic upgrades are disabled</>;
    }

    const toggleAutoUpgrade = (isChecked: boolean) => {
        const previousValue = autoUpgradeConfig.enableAutoUpgrade;

        const newConfig = {
            ...autoUpgradeConfig,
            enableAutoUpgrade: isChecked,
        };

        // optimistic UI update
        setAutoUpgradeConfig(newConfig);

        saveAutoUpgradeConfig(newConfig).catch(() => {
            // rollback on failure
            setAutoUpgradeConfig({
                ...autoUpgradeConfig,
                enableAutoUpgrade: previousValue,
            });
            fetchConfig();
        });
    };

    return (
        <Switch
            id="enableAutoUpgrade"
            label="Automatically upgrade secured clusters"
            isChecked={autoUpgradeConfig.enableAutoUpgrade}
            onChange={(_e, isChecked) => toggleAutoUpgrade(isChecked)}
        />
    );
}

export default AutoUpgradeToggle;
