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
import { EnabledSelections } from 'types/dedicatedRegistryConfig.proto';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import { Cluster } from 'types/cluster.proto';

type DelegatedScanningSettingsProps = {
    enabledFor: EnabledSelections;
    onChangeEnabledFor: (newEnabledState: EnabledSelections) => void;
    clusters?: Cluster[];
    selectedClusterId?: string;
    setSelectedClusterId: (newClusterId: string) => void;
};

function DelegatedScanningSettings({
    enabledFor,
    onChangeEnabledFor,
    clusters = [],
    selectedClusterId = '',
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
                <Flex className="pf-u-mt-md pf-u-mb-lg">
                    <FlexItem>
                        <FormLabelGroup
                            label="Select default cluster to delegate to"
                            isRequired
                            fieldId="selectedClusterId"
                            touched={{}}
                            errors={{}}
                        >
                            <Select
                                className="cluster-select"
                                isPlain
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
                        </FormLabelGroup>
                    </FlexItem>
                </Flex>
            </CardBody>
        </Card>
    );
}

export default DelegatedScanningSettings;
