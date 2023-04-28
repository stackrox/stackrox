import React from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';
import find from 'lodash/find';
import dateFns from 'date-fns';
import { Tooltip } from '@patternfly/react-core';

import dateTimeFormat from 'constants/dateTimeFormat';
import { sortValue, sortDate } from 'sorters/sorters';

function DeploymentNameColumn({ original }) {
    const isSuspicious = find(original.baselineStatuses, {
        anomalousProcessesExecuted: true,
    });
    return (
        <div className="flex items-center">
            <span className="pr-1">
                {isSuspicious && (
                    <Tooltip content="Abnormal processes discovered">
                        <Icon.Circle className="h-2 w-2 text-alert-400" fill="#ffebf1" />
                    </Tooltip>
                )}
                {!isSuspicious && <Icon.Circle className="h-2 w-2" />}
            </span>
            {original.deployment.name}
        </div>
    );
}

DeploymentNameColumn.propTypes = {
    original: PropTypes.shape({
        deployment: PropTypes.shape({
            name: PropTypes.string.isRequired,
        }).isRequired,
        baselineStatuses: PropTypes.arrayOf(PropTypes.object).isRequired,
    }).isRequired,
};

const riskTableColumnDescriptors = [
    {
        Header: 'Name',
        accessor: 'deployment.name',
        searchField: 'Deployment',
        Cell: DeploymentNameColumn,
    },
    {
        Header: 'Created',
        accessor: 'deployment.created',
        searchField: 'Created',
        Cell: ({ value }) => <span>{dateFns.format(value, dateTimeFormat)}</span>,
        sortMethod: sortDate,
    },
    {
        Header: 'Cluster',
        searchField: 'Cluster',
        accessor: 'deployment.cluster',
    },
    {
        Header: 'Namespace',
        searchField: 'Namespace',
        accessor: 'deployment.namespace',
    },
    {
        Header: 'Priority',
        searchField: 'Deployment Risk Priority',
        accessor: 'deployment.priority',
        Cell: ({ value }) => {
            const asInt = parseInt(value, 10);
            return Number.isNaN(asInt) || asInt < 1 ? '-' : value;
        },
        sortMethod: sortValue,
    },
];

export default riskTableColumnDescriptors;
