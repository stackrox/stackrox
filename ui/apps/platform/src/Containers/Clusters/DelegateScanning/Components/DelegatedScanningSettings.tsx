import React, { useState } from 'react';
import {
    Flex,
    FlexItem,
    FormGroup,
    FormHelperText,
    HelperText,
    HelperTextItem,
    MenuToggleElement,
    MenuToggle,
    Select,
    SelectOption,
} from '@patternfly/react-core';

import { DelegatedRegistryCluster } from 'services/DelegatedRegistryConfigService';

import { getClusterName } from '../cluster';

type DelegatedScanningSettingsProps = {
    clusters: DelegatedRegistryCluster[];
    defaultClusterId: string;
    isEditing: boolean;
    setDefaultClusterId: (newClusterId: string) => void;
};

function DelegatedScanningSettings({
    clusters,
    defaultClusterId,
    isEditing,
    setDefaultClusterId,
}: DelegatedScanningSettingsProps) {
    const [isOpen, setIsOpen] = useState(false);

    // Options consist of valid clusters, plus default cluster (in unlikely case that it is not valid).
    const clusterSelectOptions: JSX.Element[] = clusters
        .filter((cluster) => cluster.isValid || cluster.id === defaultClusterId)
        .map((cluster) => (
            <SelectOption key={cluster.id} value={cluster.id}>
                <span>{getClusterName(clusters, cluster.id)}</span>
            </SelectOption>
        ));

    const onClusterSelect = (_, value) => {
        setIsOpen(false);
        setDefaultClusterId(value);
    };

    const defaultClusterName =
        defaultClusterId === '' ? 'None' : getClusterName(clusters, defaultClusterId);

    return (
        <FormGroup label="Default cluster to delegate to">
            <Flex>
                <FlexItem>
                    <Select
                        onOpenChange={setIsOpen}
                        onSelect={onClusterSelect}
                        isOpen={isOpen}
                        selected={defaultClusterId}
                        shouldFocusToggleOnSelect
                        toggle={(toggleRef: React.Ref<MenuToggleElement>) => (
                            <MenuToggle
                                aria-label="Select default cluster"
                                ref={toggleRef}
                                onClick={() => setIsOpen(!isOpen)}
                                isDisabled={!isEditing}
                                isExpanded={isOpen}
                            >
                                {defaultClusterName}
                            </MenuToggle>
                        )}
                    >
                        <SelectOption key="" value="">
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
