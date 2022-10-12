import React, { ReactElement, useState, useEffect } from 'react';

import ToggleSwitch from 'Components/ToggleSwitch';
import {
    getAutoUpgradeConfig,
    saveAutoUpgradeConfig,
    AutoUpgradeConfig,
    Upgradability,
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

    function toggleAutoUpgrade(): void {
        if (!autoUpgradeConfig) {
            return;
        }

        // @TODO, wrap this settings change in a confirmation prompt of some sort
        const previousValue = autoUpgradeConfig.enableAutoUpgrade;
        const newConfig = {
            ...autoUpgradeConfig,
            enableAutoUpgrade: !previousValue,
            autoUpgradeAllowed: Upgradability.NOT_ALLOWED,
        };

        setAutoUpgradeConfig(newConfig); // optimistically set value before API call

        saveAutoUpgradeConfig(newConfig).catch(() => {
            // reverse the optimistic update of the control in the UI
            const rollbackConfig = {
                ...autoUpgradeConfig,
                enableAutoUpgrade: previousValue,
                autoUpgradeAllowed: Upgradability.NOT_ALLOWED,
            };
            setAutoUpgradeConfig(rollbackConfig);

            // also, re-fetch the data from the server, just in case it did update but we didn't get the network response
            fetchConfig();
        });
    }

    if (!autoUpgradeConfig) {
        return <></>;
    }
    return autoUpgradeConfig.autoUpgradeAllowed === Upgradability.ALLOWED ? (
        <ToggleSwitch
            id="enableAutoUpgrade"
            toggleHandler={toggleAutoUpgrade}
            label="Automatically upgrade secured clusters"
            enabled={autoUpgradeConfig.enableAutoUpgrade}
        />
    ) : (
        <>Auto upgrade not allowed in managed central</>
    );
}

export default AutoUpgradeToggle;
