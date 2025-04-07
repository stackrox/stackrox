import React, { useState } from 'react';
import {
    Button,
    MenuToggleElement,
    MenuToggle,
    Select,
    SelectOption,
    TextInput,
} from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { MinusCircleIcon } from '@patternfly/react-icons';

import {
    DelegatedRegistry,
    DelegatedRegistryCluster,
} from 'services/DelegatedRegistryConfigService';

import { getClusterName } from '../cluster';

type DelegatedRegistriesTableProps = {
    registries: DelegatedRegistry[];
    clusters: DelegatedRegistryCluster[];
    selectedClusterId: string;
    isEditing: boolean;
    handlePathChange: (number, string) => void;
    handleClusterChange: (number, string) => void;
    deleteRow: (number) => void;
};

function DelegatedRegistriesTable({
    registries,
    clusters,
    selectedClusterId,
    isEditing,
    handlePathChange,
    handleClusterChange,
    deleteRow,
}: DelegatedRegistriesTableProps) {
    const [openRow, setRowOpen] = useState<number>(-1);
    function toggleSelect(rowToToggle: number) {
        setRowOpen((prev) => (rowToToggle === prev ? -1 : rowToToggle));
    }
    function onSelect(rowIndex, value) {
        handleClusterChange(rowIndex, value);
        setRowOpen(-1);
    }

    const defaultClusterName =
        selectedClusterId === '' ? 'None' : getClusterName(clusters, selectedClusterId);
    const defaultClusterItem = `Default cluster: ${defaultClusterName}`;

    return (
        <Table aria-label="Delegated registry exceptions table">
            <Thead>
                <Tr>
                    <Th width={40}>Source registry</Th>
                    <Th width={40}>Destination cluster (CLI/API only)</Th>
                    {isEditing && (
                        <Th>
                            <span className="pf-v5-screen-reader">Row action</span>
                        </Th>
                    )}
                </Tr>
            </Thead>
            <Tbody>
                {registries.map((registry, rowIndex) => {
                    const selectedClusterName =
                        registry.clusterId === ''
                            ? defaultClusterItem
                            : getClusterName(clusters, registry.clusterId);

                    // Options consist of valid clusters, plus destination cluster (in unlikely case that it is not valid).
                    const clusterSelectOptions: JSX.Element[] = clusters
                        .filter((cluster) => cluster.isValid || cluster.id === registry.clusterId)
                        .map((cluster) => {
                            return (
                                <SelectOption key={cluster.id} value={cluster.id}>
                                    {getClusterName(clusters, cluster.id)}
                                </SelectOption>
                            );
                        });

                    // Even path and clusterId combined is not a unique key.
                    /* eslint-disable react/no-array-index-key */
                    return (
                        <Tr key={rowIndex}>
                            <Td dataLabel="Source registry">
                                <TextInput
                                    aria-label="registry"
                                    isRequired
                                    isDisabled={!isEditing}
                                    type="text"
                                    value={registry.path}
                                    onChange={(_event, value) => handlePathChange(rowIndex, value)}
                                />
                            </Td>
                            <Td dataLabel="Destination cluster (CLI/API only)">
                                <Select
                                    onSelect={(_, value) => onSelect(rowIndex, value)}
                                    isOpen={openRow === rowIndex}
                                    selected={registry.clusterId}
                                    toggle={(toggleRef: React.Ref<MenuToggleElement>) => (
                                        <MenuToggle
                                            aria-label="Select destination cluster"
                                            ref={toggleRef}
                                            onClick={() => toggleSelect(rowIndex)}
                                            isDisabled={!isEditing}
                                            isExpanded={openRow === rowIndex}
                                        >
                                            {selectedClusterName}
                                        </MenuToggle>
                                    )}
                                >
                                    <SelectOption key="" value="">
                                        {defaultClusterItem}
                                    </SelectOption>
                                    <>{clusterSelectOptions}</>
                                </Select>
                            </Td>
                            {isEditing && (
                                <Td dataLabel="Row action" className="pf-v5-u-text-align-right">
                                    <Button
                                        variant="link"
                                        isInline
                                        icon={
                                            <MinusCircleIcon color="var(--pf-v5-global--danger-color--100)" />
                                        }
                                        onClick={() => deleteRow(rowIndex)}
                                    >
                                        Delete row
                                    </Button>
                                </Td>
                            )}
                        </Tr>
                    );
                })}
            </Tbody>
        </Table>
    );
    /* eslint-ensable react/no-array-index-key */
}

export default DelegatedRegistriesTable;
