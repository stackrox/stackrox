/* eslint-disable react/no-array-index-key */
import React, { ReactElement } from 'react';
import {
    Button,
    Divider,
    PageSection,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
    DropdownItem,
} from '@patternfly/react-core';
import { TableComposable, Thead, Tbody, Tr, Th, Td } from '@patternfly/react-table';

import BulkActionsDropdown from 'Components/PatternFly/BulkActionsDropdown';
import AutoUpgradeToggle from './AutoUpgradeToggle';

function ClustersTable(): ReactElement {
    const columns = [
        'Name',
        'Cloud provider',
        'Cluster status',
        'Sensor upgrade',
        'Credential expiration',
    ];
    // @TODO: pass the rows down from the parent component
    const rows = [
        ['remote', 'GCP us-central1', 'Healthy', 'Up to date with Central', 'in 12 months'],
    ];

    function upgradeSelectedClusters() {
        // @TODO: Add logic for upgrading clusters
    }

    function deleteSelectedClusters() {
        // @TODO: Add logic for showing confirmation dialog for deleting clusters
    }

    // @TODO: add the URL search input
    // @TODO: add pagination when we need it or add it to at least see the number of clusters in the table
    return (
        <PageSection isFilled padding={{ default: 'noPadding' }}>
            <Toolbar id="toolbar" inset={{ default: 'insetLg' }}>
                <ToolbarContent>
                    <ToolbarItem>
                        <AutoUpgradeToggle />
                    </ToolbarItem>
                    <ToolbarItem variant="separator" />
                    <ToolbarItem>
                        <Button variant="primary">Add cluster</Button>
                    </ToolbarItem>
                    <ToolbarItem variant="separator" />
                    <ToolbarItem>
                        <BulkActionsDropdown>
                            <DropdownItem
                                key="upgrade"
                                component="button"
                                onClick={upgradeSelectedClusters}
                            >
                                Upgrade
                            </DropdownItem>
                            <DropdownItem
                                key="delete"
                                component="button"
                                onClick={deleteSelectedClusters}
                            >
                                Delete
                            </DropdownItem>
                        </BulkActionsDropdown>
                    </ToolbarItem>
                </ToolbarContent>
            </Toolbar>
            <Divider component="div" />
            {/* @TODO: Use checkboxes */}
            <TableComposable variant="compact" isStickyHeader>
                <Thead>
                    <Tr>
                        {/* TODO: https://github.com/stackrox/rox/pull/9396#discussion_r714272049 */}
                        {columns.map((column, columnIndex) => (
                            <Th key={columnIndex}>{column}</Th>
                        ))}
                    </Tr>
                </Thead>
                <Tbody>
                    {rows.map((row, rowIndex) => (
                        <Tr key={rowIndex}>
                            {/* @TODO: Render the clusters status, sensor upgrade, and credential expiration differently */}
                            {row.map((cell, cellIndex) => (
                                <Td key={`${rowIndex}_${cellIndex}`} dataLabel={columns[cellIndex]}>
                                    {cell}
                                </Td>
                            ))}
                        </Tr>
                    ))}
                </Tbody>
            </TableComposable>
            {/* @TODO: Add Confirmation Dialogs here */}
        </PageSection>
    );
}

export default ClustersTable;
