import { Link } from 'react-router-dom-v5-compat';
import PropTypes from 'prop-types';
import find from 'lodash/find';
import { Tooltip, Button } from '@patternfly/react-core';
import { CheckIcon, ExclamationCircleIcon, AngleUpIcon, AngleDownIcon } from '@patternfly/react-icons';

import { sortDate, sortValue } from 'sorters/sorters';
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

function ActionButtonsColumn({ original, onMoveUp, onMoveDown, loadingDeploymentId }) {
    const deploymentId = original.deployment.id;
    const isLoading = loadingDeploymentId === deploymentId;

    return (
        <div className="flex items-center gap-2">
            <Tooltip content="Move deployment up in risk ranking">
                <Button
                    variant="plain"
                    aria-label="Move up"
                    onClick={(e) => {
                        e.stopPropagation();
                        onMoveUp(deploymentId);
                    }}
                    isDisabled={isLoading}
                    icon={<AngleUpIcon />}
                />
            </Tooltip>
            <Tooltip content="Move deployment down in risk ranking">
                <Button
                    variant="plain"
                    aria-label="Move down"
                    onClick={(e) => {
                        e.stopPropagation();
                        onMoveDown(deploymentId);
                    }}
                    isDisabled={isLoading}
                    icon={<AngleDownIcon />}
                />
            </Tooltip>
        </div>
    );
}

ActionButtonsColumn.propTypes = {
    original: PropTypes.shape({
        deployment: PropTypes.shape({
            id: PropTypes.string.isRequired,
        }).isRequired,
    }).isRequired,
    onMoveUp: PropTypes.func.isRequired,
    onMoveDown: PropTypes.func.isRequired,
    loadingDeploymentId: PropTypes.string,
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
    {
        Header: 'Actions',
        accessor: 'actions',
        Cell: ActionButtonsColumn,
        sortable: false,
    },
];

export default riskTableColumnDescriptors;
export { ActionButtonsColumn };
