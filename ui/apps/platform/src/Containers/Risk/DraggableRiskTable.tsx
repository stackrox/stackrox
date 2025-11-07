import { useState, useRef } from 'react';
import { useDrag, useDrop, DndProvider } from 'react-dnd';
import { HTML5Backend } from 'react-dnd-html5-backend';
import type { Identifier, XYCoord } from 'dnd-core';
import {
    Table,
    Thead,
    Tr,
    Th,
    Tbody,
    Td,
    Caption,
} from '@patternfly/react-table';
import { GripVerticalIcon } from '@patternfly/react-icons';
import { Link } from 'react-router-dom-v5-compat';
import find from 'lodash/find';
import { Tooltip } from '@patternfly/react-core';
import { CheckIcon, ExclamationCircleIcon } from '@patternfly/react-icons';

import { riskBasePath } from 'routePaths';
import { getDateTime } from 'utils/dateUtils';

const ITEM_TYPE = 'RISK_TABLE_ROW';

interface DragItem {
    index: number;
    id: string;
    type: string;
}

interface DeploymentRow {
    deployment: {
        id: string;
        name: string;
        created: string;
        cluster: string;
        namespace: string;
        priority: number;
    };
    baselineStatuses: Array<{
        anomalousProcessesExecuted: boolean;
    }>;
}

interface DraggableRowProps {
    row: DeploymentRow;
    index: number;
    moveRow: (dragIndex: number, hoverIndex: number) => void;
    onRowClick: (row: DeploymentRow) => void;
    selectedDeploymentId?: string;
}

function DraggableRow({ row, index, moveRow, onRowClick, selectedDeploymentId }: DraggableRowProps) {
    const ref = useRef<HTMLTableRowElement>(null);
    const [{ handlerId }, drop] = useDrop<DragItem, void, { handlerId: Identifier | null }>({
        accept: ITEM_TYPE,
        collect(monitor) {
            return {
                handlerId: monitor.getHandlerId(),
            };
        },
        hover(item: DragItem, monitor) {
            if (!ref.current) {
                return;
            }
            const dragIndex = item.index;
            const hoverIndex = index;

            if (dragIndex === hoverIndex) {
                return;
            }

            const hoverBoundingRect = ref.current?.getBoundingClientRect();
            const hoverMiddleY = (hoverBoundingRect.bottom - hoverBoundingRect.top) / 2;
            const clientOffset = monitor.getClientOffset();
            const hoverClientY = (clientOffset as XYCoord).y - hoverBoundingRect.top;

            if (dragIndex < hoverIndex && hoverClientY < hoverMiddleY) {
                return;
            }

            if (dragIndex > hoverIndex && hoverClientY > hoverMiddleY) {
                return;
            }

            moveRow(dragIndex, hoverIndex);
            item.index = hoverIndex;
        },
    });

    const [{ isDragging }, drag, preview] = useDrag({
        type: ITEM_TYPE,
        item: () => {
            return { id: row.deployment.id, index };
        },
        collect: (monitor: any) => ({
            isDragging: monitor.isDragging(),
        }),
    });

    preview(drop(ref));

    const isSuspicious = find(row.baselineStatuses, {
        anomalousProcessesExecuted: true,
    });

    const isSelected = row.deployment.id === selectedDeploymentId;
    const opacity = isDragging ? 0.4 : 1;
    const priority = row.deployment.priority;
    const asInt = parseInt(String(priority), 10);
    const displayPriority = Number.isNaN(asInt) || asInt < 1 ? '-' : String(priority);

    return (
        <Tr
            ref={ref}
            style={{ opacity, cursor: isDragging ? 'grabbing' : 'default' }}
            data-handler-id={handlerId}
            onClick={() => onRowClick(row)}
            isSelectable
            isRowSelected={isSelected}
        >
            <Td
                style={{ width: '40px', paddingRight: '0' }}
                dataLabel="Drag handle"
            >
                <div ref={drag} style={{ cursor: 'grab', display: 'flex', alignItems: 'center' }}>
                    <GripVerticalIcon />
                </div>
            </Td>
            <Td dataLabel="Name">
                <div className="flex items-center">
                    <span className="pf-v5-u-display-inline-flex pf-v5-u-align-items-center">
                        {isSuspicious ? (
                            <Tooltip content="Abnormal processes discovered">
                                <ExclamationCircleIcon color="var(--pf-v5-global--danger-color--100)" />
                            </Tooltip>
                        ) : (
                            <Tooltip content="No abnormal processes discovered">
                                <CheckIcon />
                            </Tooltip>
                        )}
                        <span className="pf-v5-u-pl-sm pf-v5-u-text-nowrap">
                            <Link to={`${riskBasePath}/${row.deployment.id}`}>
                                {row.deployment.name}
                            </Link>
                        </span>
                    </span>
                </div>
            </Td>
            <Td dataLabel="Created">{getDateTime(row.deployment.created)}</Td>
            <Td dataLabel="Cluster">{row.deployment.cluster}</Td>
            <Td dataLabel="Namespace">{row.deployment.namespace}</Td>
            <Td dataLabel="Priority">{displayPriority}</Td>
        </Tr>
    );
}

interface DraggableRiskTableProps {
    currentDeployments: DeploymentRow[];
    onRowClick: (row: DeploymentRow) => void;
    selectedDeploymentId?: string;
    onReorder: (fromIndex: number, toIndex: number) => void;
}

function DraggableRiskTableInner({
    currentDeployments,
    onRowClick,
    selectedDeploymentId,
    onReorder,
}: DraggableRiskTableProps) {
    const [rows, setRows] = useState(currentDeployments);

    // Update rows when currentDeployments changes
    if (currentDeployments !== rows) {
        setRows(currentDeployments);
    }

    const moveRow = (dragIndex: number, hoverIndex: number) => {
        const newRows = [...rows];
        const dragRow = newRows[dragIndex];
        newRows.splice(dragIndex, 1);
        newRows.splice(hoverIndex, 0, dragRow);
        setRows(newRows);
    };

    const handleDragEnd = () => {
        // Find which rows changed position
        const originalIndices = currentDeployments.map((r) => r.deployment.id);
        const newIndices = rows.map((r) => r.deployment.id);

        for (let i = 0; i < newIndices.length; i++) {
            const originalIndex = originalIndices.indexOf(newIndices[i]);
            if (originalIndex !== i) {
                onReorder(originalIndex, i);
                break;
            }
        }
    };

    return (
        <Table aria-label="Risk deployments table" variant="compact">
            <Caption>Deployments at risk - drag to reorder</Caption>
            <Thead>
                <Tr>
                    <Th />
                    <Th>Name</Th>
                    <Th>Created</Th>
                    <Th>Cluster</Th>
                    <Th>Namespace</Th>
                    <Th>Priority</Th>
                </Tr>
            </Thead>
            <Tbody onDragEnd={handleDragEnd}>
                {rows.map((row, index) => (
                    <DraggableRow
                        key={row.deployment.id}
                        row={row}
                        index={index}
                        moveRow={moveRow}
                        onRowClick={onRowClick}
                        selectedDeploymentId={selectedDeploymentId}
                    />
                ))}
            </Tbody>
        </Table>
    );
}

export default function DraggableRiskTable(props: DraggableRiskTableProps) {
    return (
        /* @ts-expect-error DndProvider types do not expect children as props */
        <DndProvider backend={HTML5Backend}>
            <DraggableRiskTableInner {...props} />
        </DndProvider>
    );
}
