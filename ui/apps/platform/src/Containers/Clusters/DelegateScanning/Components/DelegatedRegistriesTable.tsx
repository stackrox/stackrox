/* eslint-disable @typescript-eslint/no-shadow */
/* eslint-disable no-param-reassign */
/* eslint-disable react/no-array-index-key */
// TODO: remove lint override after @typescript-eslint deps can be resolved to ^5.2.x
/* eslint-disable react/prop-types */
import React, { useState, useEffect } from 'react';
import { Button, Select, SelectOption, TextInput } from '@patternfly/react-core';
import {
    TableComposable,
    Thead,
    Tbody,
    Tr,
    Th,
    Td,
    TbodyProps,
    TrProps,
} from '@patternfly/react-table';
import styles from '@patternfly/react-styles/css/components/Table/table';
import MinusCircleIcon from '@patternfly/react-icons/dist/esm/icons/minus-circle-icon';

import {
    DelegatedRegistry,
    DelegatedRegistryCluster,
} from 'services/DelegatedRegistryConfigService';

type DelegatedRegistriesTableProps = {
    registries: DelegatedRegistry[];
    clusters: DelegatedRegistryCluster[];
    selectedClusterId: string;
    handlePathChange: (number, string) => void;
    handleClusterChange: (number, string) => void;
    deleteRow: (number) => void;
    // TODO: re-enable next type after @typescript-eslint deps can be resolved to ^5.2.x
    // updateRegistriesOrder: (DelegatedRegistry[]) => void;
};

function DelegatedRegistriesTable({
    registries,
    clusters,
    selectedClusterId,
    handlePathChange,
    handleClusterChange,
    deleteRow,
    // TODO: remove lint override after @typescript-eslint deps can be resolved to ^5.2.x
    // eslint-disable-next-line @typescript-eslint/ban-ts-comment
    // @ts-ignore
    updateRegistriesOrder,
}: DelegatedRegistriesTableProps) {
    const [draggedItemId, setDraggedItemId] = React.useState<string | null>(null);
    const [draggingToItemIndex, setDraggingToItemIndex] = React.useState<number | null>(null);
    const [isDragging, setIsDragging] = React.useState(false);

    const initialOrderIds = registries.map((reg) => reg.uuid as string);
    const [itemOrder, setItemOrder] = React.useState(initialOrderIds);
    const [tempItemOrder, setTempItemOrder] = React.useState<string[]>([]);

    const [openRow, setRowOpen] = useState<number>(-1);
    function toggleSelect(rowToToggle: number) {
        setRowOpen((prev) => (rowToToggle === prev ? -1 : rowToToggle));
    }
    function onSelect(rowIndex, value) {
        handleClusterChange(rowIndex, value);
        setRowOpen(-1);
    }

    useEffect(() => {
        const orderIds = registries.map((reg) => reg.uuid as string);
        setItemOrder(orderIds);
    }, [registries]);

    const clusterSelectOptions: JSX.Element[] = clusters.map((cluster) => {
        const optionLabel =
            cluster.id === selectedClusterId ? `${cluster.name} (default)` : cluster.name;
        return (
            <SelectOption key={cluster.id} value={cluster.id}>
                <span>{optionLabel}</span>
            </SelectOption>
        );
    });

    // Start PatternFly template for drag and drop
    const bodyRef = React.useRef<HTMLTableSectionElement>();

    const onDragStart: TrProps['onDragStart'] = (evt) => {
        evt.dataTransfer.effectAllowed = 'move';
        evt.dataTransfer.setData('text/plain', evt.currentTarget.id);
        // eslint-disable-next-line @typescript-eslint/no-shadow
        const draggedItemId = evt.currentTarget.id;

        evt.currentTarget.classList.add(styles.modifiers.ghostRow);
        evt.currentTarget.setAttribute('aria-pressed', 'true');

        setDraggedItemId(draggedItemId);
        setIsDragging(true);
    };

    const moveItem = (arr: string[], i1: string, toIndex: number) => {
        const fromIndex = arr.indexOf(i1);
        if (fromIndex === toIndex) {
            return arr;
        }
        const temp = arr.splice(fromIndex, 1);
        arr.splice(toIndex, 0, temp[0]);

        return arr;
    };

    const move = (itemOrder: string[]) => {
        const ulNode = bodyRef.current;
        if (ulNode?.children) {
            const nodes = Array.from(ulNode.children);
            if (nodes.map((node) => node.id).every((id, i) => id === itemOrder[i])) {
                return;
            }
            while (ulNode?.firstChild) {
                if (ulNode.lastChild) {
                    ulNode.removeChild(ulNode.lastChild);
                }
            }

            itemOrder.forEach((id) => {
                if (nodes.find((n) => n.id === id)) {
                    ulNode.appendChild(nodes.find((n) => n.id === id) as Element);
                }
            });
        }
    };

    const onDragCancel = () => {
        if (bodyRef?.current?.children) {
            Array.from(bodyRef.current.children).forEach((el) => {
                el.classList.remove(styles.modifiers.ghostRow);
                el.setAttribute('aria-pressed', 'false');
            });
            setDraggedItemId(null);
            setDraggingToItemIndex(null);
            setIsDragging(false);
        }
    };

    const onDragLeave: TbodyProps['onDragLeave'] = (evt) => {
        if (!isValidDrop(evt)) {
            move(itemOrder);
            setDraggingToItemIndex(null);
        }
    };

    function isValidDrop(evt: React.DragEvent<HTMLTableSectionElement | HTMLTableRowElement>) {
        if (bodyRef?.current?.getBoundingClientRect()) {
            const ulRect = bodyRef.current.getBoundingClientRect();
            return (
                evt.clientX > ulRect.x &&
                evt.clientX < ulRect.x + ulRect.width &&
                evt.clientY > ulRect.y &&
                evt.clientY < ulRect.y + ulRect.height
            );
        }
        return false;
    }

    const onDrop: TrProps['onDrop'] = (evt) => {
        if (isValidDrop(evt)) {
            setItemOrder(tempItemOrder);

            // the rest of this block was added to the PF drag and drop paradigm,
            // in order to keep the form data in sync with PF's visual drop order
            const newRegistries: DelegatedRegistry[] = tempItemOrder.map((tempItem) => {
                const newIndex = registries.findIndex((reg) => reg.uuid === tempItem) || 0;
                return registries[newIndex];
            });

            updateRegistriesOrder(newRegistries);
        } else {
            onDragCancel();
        }
    };

    function onDragOver(evt): TbodyProps['onDragOver'] {
        evt.preventDefault();

        const curListItem = (evt.target as HTMLTableSectionElement).closest('tr');
        if (
            !curListItem ||
            !bodyRef?.current?.contains(curListItem) ||
            curListItem.id === draggedItemId
        ) {
            return undefined;
        }
        const dragId = curListItem.id;
        const newDraggingToItemIndex = Array.from(bodyRef.current.children).findIndex(
            (item) => item.id === dragId
        );
        if (newDraggingToItemIndex !== draggingToItemIndex) {
            const tempItemOrder = moveItem(
                [...itemOrder],
                draggedItemId || '',
                newDraggingToItemIndex
            );
            move(tempItemOrder);
            setDraggingToItemIndex(newDraggingToItemIndex);
            setTempItemOrder(tempItemOrder);
        }

        return undefined;
    }

    const onDragEnd: TrProps['onDragEnd'] = (evt) => {
        const target = evt.target as HTMLTableRowElement;
        target.classList.remove(styles.modifiers.ghostRow);
        target.setAttribute('aria-pressed', 'false');
        setDraggedItemId(null);
        setDraggingToItemIndex(null);
        setIsDragging(false);
    };
    // End PatternFly template for drag and drop

    return (
        <TableComposable
            aria-label="Delegated registry exceptions table"
            className={(isDragging && styles.modifiers.dragOver) || ''}
        >
            <Thead>
                <Tr>
                    <Th>Order</Th>
                    <Th width={40}>Source registry</Th>
                    <Th width={40}>Destination cluster (CLI/API only)</Th>
                    <Td isActionCell />
                </Tr>
            </Thead>
            <Tbody
                // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                // @ts-ignore
                ref={bodyRef}
                onDragOver={onDragOver}
                onDrop={onDragOver}
                onDragLeave={onDragLeave}
            >
                {registries.map((registry, rowIndex) => (
                    <Tr
                        // note: in spite of best practice, we have to use the array index as key here,
                        //       because the value of path changes as the user types, and the input would lose focus
                        key={registry.uuid}
                        // id={itemOrder[rowIndex]}
                        id={registry.uuid}
                        draggable
                        onDrop={onDrop}
                        onDragEnd={onDragEnd}
                        onDragStart={onDragStart}
                    >
                        <Td
                            draggableRow={{
                                id: `draggable-row-${registry.path}`,
                            }}
                        />
                        <Td dataLabel="Source registry">
                            <TextInput
                                isRequired
                                type="email"
                                id="simple-form-email-01"
                                name="simple-form-email-01"
                                value={registry.path}
                                onChange={(value) => handlePathChange(rowIndex, value)}
                            />
                        </Td>
                        <Td dataLabel="Destination cluster (CLI/API only)">
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
                                onToggle={() => toggleSelect(rowIndex)}
                                onSelect={(_, value) => onSelect(rowIndex, value)}
                                isOpen={openRow === rowIndex}
                                selections={registry.clusterId}
                            >
                                {clusterSelectOptions}
                            </Select>
                        </Td>
                        <Td dataLabel="Delete row" isActionCell>
                            <Button
                                variant="plain"
                                aria-label="Delete row"
                                onClick={() => deleteRow(rowIndex)}
                            >
                                <MinusCircleIcon />
                            </Button>
                        </Td>
                    </Tr>
                ))}
            </Tbody>
        </TableComposable>
    );
}

export default DelegatedRegistriesTable;
