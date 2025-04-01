import React from 'react';
import { FormGroup, Radio } from '@patternfly/react-core';

import { DelegatedRegistryConfigEnabledFor } from 'services/DelegatedRegistryConfigService';

type ToggleDelegatedScanningProps = {
    enabledFor: DelegatedRegistryConfigEnabledFor;
    isEditing: boolean;
    onChangeEnabledFor: (newEnabledState: DelegatedRegistryConfigEnabledFor) => void;
};

function ToggleDelegatedScanning({
    enabledFor,
    isEditing,
    onChangeEnabledFor,
}: ToggleDelegatedScanningProps) {
    return (
        <FormGroup role="radiogroup" isInline label="Delegate scanning for" fieldId="enabledFor">
            <Radio
                label="None"
                isChecked={enabledFor === 'NONE'}
                isDisabled={!isEditing}
                id="choose-none"
                name="enabledFor"
                onChange={() => {
                    onChangeEnabledFor('NONE');
                }}
            />
            <Radio
                label="All registries"
                isChecked={enabledFor === 'ALL'}
                isDisabled={!isEditing}
                id="choose-all-registries"
                name="enabledFor"
                onChange={() => {
                    onChangeEnabledFor('ALL');
                }}
            />
            <Radio
                label="Specified registries"
                isChecked={enabledFor === 'SPECIFIC'}
                isDisabled={!isEditing}
                id="chose-specified-registries"
                name="enabledFor"
                onChange={() => {
                    onChangeEnabledFor('SPECIFIC');
                }}
            />
        </FormGroup>
    );
}

export default ToggleDelegatedScanning;
