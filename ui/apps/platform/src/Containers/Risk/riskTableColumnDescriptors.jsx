import React from 'react';
import { Link } from 'react-router-dom';
import PropTypes from 'prop-types';
import find from 'lodash/find';
import { Tooltip } from '@patternfly/react-core';
import { CheckIcon, ExclamationCircleIcon } from '@patternfly/react-icons';

import { sortValue, sortDate } from 'sorters/sorters';
import { riskBasePath } from 'routePaths';
import { getDateTime } from 'utils/dateUtils';

function DeploymentNameColumn({ original }) {
    const isSuspicious = find(original.baselineStatuses, {
        anomalousProcessesExecuted: true,
    });
    // Borrow layout from IconText component.
    return (
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
                    <Link to={`${riskBasePath}/${original.deployment.id}`}>
                        {original.deployment.name}
                    </Link>
                </span>
            </span>
        </div>
    );
}

DeploymentNameColumn.propTypes = {
    original: PropTypes.shape({
        deployment: PropTypes.shape({
            id: PropTypes.string.isRequired,
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
        Cell: ({ value }) => <span>{getDateTime(value)}</span>,
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
