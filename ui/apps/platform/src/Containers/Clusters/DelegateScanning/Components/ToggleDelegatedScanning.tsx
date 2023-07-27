import React from 'react';
import { Card, CardBody, Checkbox } from '@patternfly/react-core';

import { DelegatedRegistryConfigEnabledFor } from 'services/DelegatedRegistryConfigService';

type ToggleDelegatedScanningProps = {
    enabledFor: DelegatedRegistryConfigEnabledFor;
    toggleDelegation: () => void;
};

function ToggleDelegatedScanning({ enabledFor, toggleDelegation }: ToggleDelegatedScanningProps) {
    return (
        <Card className="pf-u-mb-lg">
            <CardBody>
                <Checkbox
                    label="Enable delegated image scanning"
                    isChecked={enabledFor !== 'NONE'}
                    onChange={toggleDelegation}
                    id="enabledFor"
                    name="enabledFor"
                />
            </CardBody>
        </Card>
    );
}

export default ToggleDelegatedScanning;
