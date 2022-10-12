import React, { ReactElement, useState, useEffect } from 'react';
import { Switch } from '@patternfly/react-core';

import {
    getAutoUpgradeConfig,
    saveAutoUpgradeConfig,
    AutoUpgradeConfig,
    Upgradability,
} from 'services/ClustersService';

// TODO: Connect this to the APIs and use real data
function AutoUpgradeToggle(): ReactElement {
    const [isDisabled, setIsDisabled] = useState(true);
    const [autoUpgradeConfig, setAutoUpgradeConfig] = useState<AutoUpgradeConfig | null>(null);

    function fetchConfig(): void {
        getAutoUpgradeConfig()
            .then((config) => {
                setAutoUpgradeConfig(config);
                setIsDisabled(false);
            })
            .catch(() => {
                // TODO display message when there is a place for minor errors
                setIsDisabled(false);
            });
    }

    useEffect(() => {
        fetchConfig();
    }, []);

    function handleChange(value) {
        setIsDisabled(true);
        // @TODO: wrap this settings change in a confirmation prompt of some sort
        const newConfig = {
            ...autoUpgradeConfig,
            enableAutoUpgrade: value,
            autoUpgradeAllowed: Upgradability.NOT_ALLOWED,
        };

        setAutoUpgradeConfig(newConfig); // optimistically set value before API call

        saveAutoUpgradeConfig(newConfig)
            .then(() => {
                setIsDisabled(false);
            })
            .catch(() => {
                // reverse the optimistic update of the control in the UI
                const rollbackConfig = {
                    ...autoUpgradeConfig,
                    enableAutoUpgrade: value,
                    autoUpgradeAllowed: Upgradability.NOT_ALLOWED,
                };
                setAutoUpgradeConfig(rollbackConfig);

                // also, re-fetch the data from the server, just in case it did update but we didn't get the network response
                fetchConfig();
            });
    }

    const label = 'Automatically upgrade secured clusters';

    if (!autoUpgradeConfig) {
        return <></>;
    }
    return autoUpgradeConfig.autoUpgradeAllowed === Upgradability.ALLOWED ? (
        <Switch
            id="auto-upgrade-toggle"
            label={label}
            isChecked={autoUpgradeConfig.enableAutoUpgrade}
            onChange={handleChange}
            isReversed
            isDisabled={isDisabled}
        />
    ) : (
        <>Auto upgrade not allowed in managed central</>
    );
}

export default AutoUpgradeToggle;
