import React, { useEffect, useState } from 'react';
import * as Icon from 'react-feather';
import Tooltip from 'rc-tooltip';

import PageHeader from 'Components/PageHeader';
import Panel from 'Components/Panel';
import CheckboxTable from 'Components/CheckboxTable';
import { defaultColumnClassName, wrapClassName, rtTrActionsClassName } from 'Components/Table';
import TableHeader from 'Components/TableHeader';
import { fetchClusterAsArray } from 'services/ClustersService';
import { toggleRow, toggleSelectAll } from 'utils/checkboxUtils';

import {
    checkInLabel,
    formatCollectionMethod,
    formatAdmissionController,
    formatLastCheckIn,
    formatSensorVersion,
    sensorVersionLabel
} from '../Integrations/Clusters/ClusterDetails';

const ClustersPage = () => {
    const [currentClusters, setCurrentClusters] = useState([]);
    const [selectedClusters, setSelectedClusters] = useState([]);
    const [tableRef, setTableRef] = useState(null);

    // @TODO, implement actual delete logic into this stub function
    const onDeleteHandler = cluster => e => {
        e.stopPropagation();
        setSelectedClusters([cluster.id]);
    };

    function renderRowActionButtons(cluster) {
        return (
            <div className="border-2 border-r-2 border-base-400 bg-base-100">
                <Tooltip placement="top" overlay={<div>Delete cluster</div>} mouseLeaveDelay={0}>
                    <button
                        type="button"
                        className="p-1 px-4 hover:bg-primary-200 text-primary-600 hover:text-primary-700"
                        onClick={onDeleteHandler(cluster)}
                    >
                        <Icon.Trash2 className="mt-1 h-4 w-4" />
                    </button>
                </Tooltip>
            </div>
        );
    }

    function toggleCluster(id) {
        const selection = toggleRow(id, selectedClusters);
        setSelectedClusters(selection);
    }

    function toggleAllClusters() {
        const rowsLength = selectedClusters.length;
        const ref = tableRef.reactTable;
        const selection = toggleSelectAll(rowsLength, selectedClusters, ref);
        setSelectedClusters(selection);
    }

    function editCluster() {}

    useEffect(() => {
        fetchClusterAsArray().then(clusters => {
            setCurrentClusters(clusters);
        });
    }, []);

    // @TODO: flesh out the new Clusters page layout, placeholders for now
    const paginationComponent = <div>Buttons and Pagination here</div>;

    const headerComponent = (
        <TableHeader length={currentClusters.length} type="Cluster" isViewFiltered={false} />
    );

    const clusterColumns = [
        {
            accessor: 'name',
            Header: 'Name',
            className: `${wrapClassName} ${defaultColumnClassName}`
        },
        {
            accessor: 'type',
            Header: 'Type'
        },
        {
            Header: 'Runtime Support',
            Cell: ({ original }) => formatCollectionMethod(original)
        },
        {
            Header: 'Admission Controller Webhook',
            Cell: ({ original }) => formatAdmissionController(original)
        },
        {
            Header: checkInLabel,
            Cell: ({ original }) => formatLastCheckIn(original)
        },
        {
            Header: sensorVersionLabel,
            Cell: ({ original }) => formatSensorVersion(original),
            className: `${wrapClassName} ${defaultColumnClassName} word-break`
        },
        {
            Header: '',
            accessor: '',
            headerClassName: 'hidden',
            className: rtTrActionsClassName,
            Cell: ({ original }) => renderRowActionButtons(original)
        }
    ];

    const selectedClusterId = '';

    return (
        <section className="flex flex-1 flex-col h-full">
            <div className="flex flex-1 flex-col">
                <PageHeader header="Clusters" />
                <div className="flex flex-1 relative">
                    <div className="shadow border-primary-300 bg-base-100 w-full overflow-hidden">
                        <Panel
                            headerTextComponent={headerComponent}
                            headerComponents={paginationComponent}
                        >
                            <div className="w-full">
                                <CheckboxTable
                                    ref={table => {
                                        setTableRef(table);
                                    }}
                                    rows={currentClusters}
                                    columns={clusterColumns}
                                    onRowClick={editCluster}
                                    toggleRow={toggleCluster}
                                    toggleSelectAll={toggleAllClusters}
                                    selection={selectedClusters}
                                    selectedRowId={selectedClusterId}
                                    noDataText="No clusters to show."
                                    minRows={20}
                                />
                            </div>
                        </Panel>
                    </div>
                </div>
            </div>
        </section>
    );
};

export default ClustersPage;
