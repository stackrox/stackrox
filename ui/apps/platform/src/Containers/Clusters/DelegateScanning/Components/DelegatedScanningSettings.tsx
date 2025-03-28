import React from 'react';
import {
    Flex,
    FlexItem,
    FormGroup,
    FormHelperText,
    HelperText,
    HelperTextItem,
} from '@patternfly/react-core';
import { Select, SelectOption } from '@patternfly/react-core/deprecated';

import { DelegatedRegistryCluster } from 'services/DelegatedRegistryConfigService';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';

type DelegatedScanningSettingsProps = {
    clusters?: DelegatedRegistryCluster[];
    isEditing: boolean;
    selectedClusterId?: string;
    setSelectedClusterId: (newClusterId: string) => void;
};

function DelegatedScanningSettings({
    clusters = [],
    isEditing,
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

    const selectedClusterName =
        clusters.find((cluster) => selectedClusterId === cluster.id)?.name ?? 'None';

    return (
        <FormGroup label="Default cluster to delegate to">
            <Flex>
                <FlexItem>
                    <Select
                        className="cluster-select"
                        placeholderText={
                            <span>
                                <span style={{ position: 'relative', top: '1px' }}>None</span>
                            </span>
                        }
                        toggleAriaLabel="Select a cluster"
                        onToggle={(_e, v) => toggleIsClusterOpen(v)}
                        onSelect={onClusterSelect}
                        isDisabled={!isEditing}
                        isOpen={isClusterOpen}
                        selections={selectedClusterName}
                    >
                        <SelectOption key="no-cluster-selected" value="" isPlaceholder>
                            <span>None</span>
                        </SelectOption>
                        <>{clusterSelectOptions}</>
                    </Select>
                </FlexItem>
            </Flex>
            <FormHelperText>
                <HelperText>
                    <HelperTextItem>
                        Select a cluster to process CLI and API-originated scanning requests
                    </HelperTextItem>
                </HelperText>
            </FormHelperText>
        </FormGroup>
    );
}

export default DelegatedScanningSettings;
