import React, { useState } from 'react';
import {
    Button,
    FormHelperText,
    HelperText,
    HelperTextItem,
    MenuToggleElement,
    MenuToggle,
    Select,
    SelectOption,
    TextInput,
} from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { MinusCircleIcon } from '@patternfly/react-icons';
import { FormikContextType } from 'formik';
import get from 'lodash/get';
import * as yup from 'yup';

import {
    DelegatedRegistry,
    DelegatedRegistryCluster,
    DelegatedRegistryConfig,
} from 'services/DelegatedRegistryConfigService';

import { getClusterName } from '../cluster';

export const pathRequiredMessage = 'Source registry is required';

// Limit validation to property that corresponds to TextInput element.
export const registriesSchema = yup.array().of(
    yup.object({
        path: yup.string().trim().required(pathRequiredMessage),
    })
);

type DelegatedRegistriesTableProps = {
    clusters: DelegatedRegistryCluster[];
    defaultClusterId: string;
    deleteRegistry: (indexToDelete: number) => void;
    formik: FormikContextType<DelegatedRegistryConfig>;
    isEditing: boolean;
    registries: DelegatedRegistry[];
    setRegistryClusterId: (indexToSet: number, clusterId: string) => void;
    setRegistryPath: (indexToSet: number, path: string) => void;
};

function DelegatedRegistriesTable({
    clusters,
    defaultClusterId,
    deleteRegistry,
    formik,
    isEditing,
    registries,
    setRegistryClusterId,
    setRegistryPath,
}: DelegatedRegistriesTableProps) {
    const [openRow, setRowOpen] = useState<number>(-1);
    function toggleSelect(rowToToggle: number) {
        setRowOpen((prev) => (rowToToggle === prev ? -1 : rowToToggle));
    }
    function onSelect(rowIndex, value) {
        setRegistryClusterId(rowIndex, value);
        setRowOpen(-1);
    }

    const defaultClusterName =
        defaultClusterId === '' ? 'None' : getClusterName(clusters, defaultClusterId);
    const defaultClusterItem = `Default cluster: ${defaultClusterName}`;

    const { errors, handleBlur, touched } = formik;

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

                    // Source reqistry helper text:
                    // no text if TextInput has valid (non-empty trimmed) value.
                    // pathErrorMessage with default validation variant if not yet touched
                    // pathErrorMessage with error validation variant if has been touched
                    // theoretical: any other validation error independent of touched
                    //
                    // Why lodash get instead of optional chaining?
                    // Unlike touched below, errors has TS2339 error (pardon pun) for array of objects.
                    const pathErrorMessage = get(errors, `registries[${rowIndex}].path`);
                    const pathValidatedVariant =
                        pathErrorMessage &&
                        (pathErrorMessage !== pathRequiredMessage ||
                            touched.registries?.[rowIndex]?.path)
                            ? 'error'
                            : 'default';

                    // Even path and clusterId combined is not a unique key.
                    /* eslint-disable react/no-array-index-key */
                    return (
                        <Tr key={rowIndex}>
                            <Td dataLabel="Source registry">
                                <TextInput
                                    aria-label="registry"
                                    isRequired
                                    isDisabled={!isEditing}
                                    name={`registries[${rowIndex}].path`}
                                    type="text"
                                    validated={pathValidatedVariant}
                                    value={registry.path}
                                    onBlur={handleBlur}
                                    onChange={(_event, value) => setRegistryPath(rowIndex, value)}
                                />
                                <FormHelperText>
                                    <HelperText>
                                        <HelperTextItem variant={pathValidatedVariant}>
                                            {pathErrorMessage}
                                        </HelperTextItem>
                                    </HelperText>
                                </FormHelperText>
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
                                        onClick={() => deleteRegistry(rowIndex)}
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
    /* eslint-enable react/no-array-index-key */
}

export default DelegatedRegistriesTable;
