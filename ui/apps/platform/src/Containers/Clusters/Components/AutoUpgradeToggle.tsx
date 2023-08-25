import React, { ReactElement, useState, useEffect } from 'react';

import ToggleSwitch from 'Components/ToggleSwitch';
import {
    isAutoUpgradeSupported,
    getAutoUpgradeConfig,
    saveAutoUpgradeConfig,
    AutoUpgradeConfig,
} from 'services/ClustersService';

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
        return <>Automatic upgrades are disabled for Cloud Service</>;
    }

    const toggleAutoUpgrade = () => {
        // @TODO, wrap this settings change in a confirmation prompt of some sort
        const previousValue = autoUpgradeConfig.enableAutoUpgrade;
        const newConfig = {
            ...autoUpgradeConfig,
            enableAutoUpgrade: !previousValue,
        };

        setAutoUpgradeConfig(newConfig); // optimistically set value before API call

        saveAutoUpgradeConfig(newConfig).catch(() => {
            // reverse the optimistic update of the control in the UI
            const rollbackConfig = {
                ...autoUpgradeConfig,
                enableAutoUpgrade: previousValue,
            };
            setAutoUpgradeConfig(rollbackConfig);

            // also, re-fetch the data from the server, just in case it did update but we didn't get the network response
            fetchConfig();
        });
    };

    return (
        <ToggleSwitch
            id="enableAutoUpgrade"
            toggleHandler={toggleAutoUpgrade}
            label="Automatically upgrade secured clusters"
            enabled={autoUpgradeConfig.enableAutoUpgrade}
        />
    );
}

export default AutoUpgradeToggle;
