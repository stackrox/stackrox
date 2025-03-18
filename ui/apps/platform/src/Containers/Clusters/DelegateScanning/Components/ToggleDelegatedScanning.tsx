import React from 'react';
import { Card, CardBody, Flex, FlexItem, Radio } from '@patternfly/react-core';

import FormLabelGroup from 'Components/PatternFly/FormLabelGroup';
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
        <Card className="pf-v5-u-mb-lg">
            <CardBody>
                <FormLabelGroup
                    label="Delegate scanning for"
                    fieldId="enabledFor"
                    touched={{}}
                    errors={{}}
                >
                    <Flex className="pf-v5-u-mt-md pf-v5-u-mb-lg">
                        <FlexItem>
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
                        </FlexItem>
                        <FlexItem>
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
                        </FlexItem>
                        <FlexItem>
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
                        </FlexItem>
                    </Flex>
                </FormLabelGroup>
            </CardBody>
        </Card>
    );
}

export default ToggleDelegatedScanning;
