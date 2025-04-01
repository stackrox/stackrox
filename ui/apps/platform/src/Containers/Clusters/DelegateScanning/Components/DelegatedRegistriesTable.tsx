import React, { useState } from 'react';
import { Button, TextInput } from '@patternfly/react-core';
import { Select, SelectOption } from '@patternfly/react-core/deprecated';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { MinusCircleIcon } from '@patternfly/react-icons';

import {
    DelegatedRegistry,
    DelegatedRegistryCluster,
} from 'services/DelegatedRegistryConfigService';

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
        selectedClusterId === ''
            ? 'None'
            : (clusters.find(({ id }) => id === selectedClusterId)?.name ?? selectedClusterId);
    const defaultClusterItem = `Default cluster (${defaultClusterName})`;

    const clusterSelectOptions: JSX.Element[] = clusters.map((cluster) => {
        return (
            <SelectOption key={cluster.id} value={cluster.id}>
                {cluster.name}
            </SelectOption>
        );
    });

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
                    const selectedClusterItem =
                        registry.clusterId === ''
                            ? defaultClusterItem
                            : (clusters.find((cluster) => registry.clusterId === cluster.id)
                                  ?.name ?? registry.clusterId);

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
                                    toggleAriaLabel="Select a cluster"
                                    onToggle={() => toggleSelect(rowIndex)}
                                    onSelect={(_, value) => onSelect(rowIndex, value)}
                                    isOpen={openRow === rowIndex}
                                    isDisabled={!isEditing}
                                    selections={selectedClusterItem}
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
