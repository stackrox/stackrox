import React from 'react';
import {
    Card,
    CardBody,
    Flex,
    FlexItem,
    Radio,
    Select,
    SelectOption,
} from '@patternfly/react-core';

import FormLabelGroup from 'Components/PatternFly/FormLabelGroup';
import {
    EnabledSelections,
    DelegatedRegistryCluster,
} from 'services/DelegatedRegistryConfigService';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';

type DelegatedScanningSettingsProps = {
    enabledFor: EnabledSelections;
    onChangeEnabledFor: (newEnabledState: EnabledSelections) => void;
    clusters?: DelegatedRegistryCluster[];
    selectedClusterId?: string;
    setSelectedClusterId: (newClusterId: string) => void;
};

function DelegatedScanningSettings({
    enabledFor,
    onChangeEnabledFor,
    clusters = [],
    selectedClusterId,
    setSelectedClusterId,
}: DelegatedScanningSettingsProps) {
    const {
        isOpen: isClusterOpen,
        toggleSelect: toggleIsClusterOpen,
        closeSelect: closeClusterSelect,
    } = useSelectToggle();

    const clusterSelectOptions: JSX.Element[] = clusters.map((cluster) => (
        <SelectOption key={cluster.id} value={cluster.id}>
            <span>{cluster.name}</span>
        </SelectOption>
    ));

    const onClusterSelect = (_, value) => {
        closeClusterSelect();
        setSelectedClusterId(value);
    };

    return (
        <Card className="pf-u-mb-lg">
            <CardBody>
                <FormLabelGroup
                    label="Delegate scanning for"
                    isRequired
                    fieldId="enabledFor"
                    touched={{}}
                    errors={{}}
                >
                    <Flex className="pf-u-mt-md pf-u-mb-lg">
                        <FlexItem>
                            <Radio
                                label="All registries"
                                isChecked={enabledFor === 'ALL'}
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
                                id="chose-specified-registries"
                                name="enabledFor"
                                onChange={() => {
                                    onChangeEnabledFor('SPECIFIC');
                                }}
                            />
                        </FlexItem>
                    </Flex>
                </FormLabelGroup>
                <FormLabelGroup
                    label="Select default cluster to delegate to"
                    helperText="Select a cluster to process CLI and API-originated scanning requests"
                    isRequired
                    fieldId="selectedClusterId"
                    touched={{}}
                    errors={{}}
                >
                    <Flex>
                        <FlexItem>
                            <Select
                                className="cluster-select"
                                placeholderText={
                                    <span>
                                        <span style={{ position: 'relative', top: '1px' }}>
                                            None
                                        </span>
                                    </span>
                                }
                                toggleAriaLabel="Select a cluster"
                                onToggle={toggleIsClusterOpen}
                                onSelect={onClusterSelect}
                                isOpen={isClusterOpen}
                                selections={selectedClusterId}
                            >
                                {clusterSelectOptions}
                            </Select>
                        </FlexItem>
                    </Flex>
                </FormLabelGroup>
            </CardBody>
        </Card>
    );
}

export default DelegatedScanningSettings;
