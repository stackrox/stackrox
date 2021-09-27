import React, { ReactElement, useState } from 'react';
import { Switch } from '@patternfly/react-core';

// TODO: Connect this to the APIs and use real data
function AutoUpgradeToggle(): ReactElement {
    const [isChecked, setIsChecked] = useState();

    function handleChange(value) {
        setIsChecked(value);
    }

    const label = 'Automatically upgrade secured clusters';

    return (
        <Switch
            id="auto-upgrade-toggle"
            label={label}
            isChecked={isChecked}
            onChange={handleChange}
            isReversed
        />
    );
}

export default AutoUpgradeToggle;
