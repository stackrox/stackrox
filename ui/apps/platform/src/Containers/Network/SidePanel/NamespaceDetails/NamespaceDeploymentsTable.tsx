import React, { ReactElement } from 'react';
import capitalize from 'lodash/capitalize';
import * as Icon from 'react-feather';

import Table, { rtTrActionsClassName } from 'Components/Table';
import RowActionButton from 'Components/RowActionButton';
import NoResultsMessage from 'Components/NoResultsMessage';
import { filterModes, filterLabels } from 'constants/networkFilterModes';
import { getNetworkFlows } from 'utils/networkUtils/getNetworkFlows';
import { sortValue } from 'sorters/sorters';

type NamespaceDeploymentsTableProps = {
    deployments: [];
    page: number;
    onNavigateToDeploymentById: (id, entityType) => void;
    filterState: number;
};

function NamespaceDeploymentsTable({
    deployments,
    page,
    onNavigateToDeploymentById,
    filterState,
}: NamespaceDeploymentsTableProps): ReactElement {
    const filterStateString =
        filterState !== filterModes.all ? capitalize(filterLabels[filterState]) : 'Network';

    const columns = [
        {
            Header: 'Deployment',
            accessor: 'data.name',
            Cell: ({ value }) => <span>{value}</span>,
        },
        {
            Header: `${filterStateString} Flows`,
            accessor: 'data.edges',
            Cell: ({ value }) => {
                const { networkFlows } = getNetworkFlows(value, filterState);
                return <span>{networkFlows.length}</span>;
            },
            sortMethod: sortValue,
        },
        {
            accessor: '',
            headerClassName: 'hidden',
            className: rtTrActionsClassName,
            Cell: ({ original }) => {
                const { deploymentId, type } = original.data;
                function onClickHandler() {
                    onNavigateToDeploymentById(deploymentId, type);
                }
                return (
                    <div className="border-2 border-r-2 border-base-400 bg-base-100 flex">
                        <RowActionButton
                            text="Navigate to Deployment"
                            onClick={onClickHandler}
                            icon={<Icon.ArrowUpRight className="my-1 h-4 w-4" />}
                        />
                    </div>
                );
            },
        },
    ];

    if (!deployments.length) {
        return <NoResultsMessage message="No namespace deployments" />;
    }
    return (
        <Table
            rows={deployments}
            columns={columns}
            noDataText="No namespace deployments"
            page={page}
            idAttribute="data.id"
        />
    );
}

export default NamespaceDeploymentsTable;
