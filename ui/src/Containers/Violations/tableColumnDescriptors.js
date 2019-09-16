import React from 'react';
import PropTypes from 'prop-types';
import pluralize from 'pluralize';
import { sortDate, sortSeverity } from 'sorters/sorters';
import dateFns from 'date-fns';
import dateTimeFormat from 'constants/dateTimeFormat';
import Tooltip from 'rc-tooltip';
import * as Icon from 'react-feather';

import { severityLabels, lifecycleStageLabels } from 'messages/common';

import {
    wrapClassName,
    defaultHeaderClassName,
    defaultColumnClassName,
    rtTrActionsClassName
} from 'Components/Table';
import ViolationActionButtons from './ViolationActionButtons';

// Simple string value column.
// ////////////////////////////
function StringValueColumn({ value }) {
    return <span>{value}</span>;
}

StringValueColumn.propTypes = {
    value: PropTypes.string.isRequired
};

// Display the deployment status and name.
// ///////////////////////////////////////
function DeploymentColumn({ original }) {
    return (
        <div className="flex">
            <span
                className="pr-2"
                title={`${original.deployment.inactive ? 'Inactive' : 'Active'} Deployment`}
            >
                <Icon.Circle
                    className="h-2 w-2 text-success-600"
                    hidden={original.deployment.inactive}
                />
                <Icon.Slash
                    className="h-2 w-2 text-base-500"
                    hidden={!original.deployment.inactive}
                />
            </span>
            <span>{original.deployment.name}</span>
        </div>
    );
}

DeploymentColumn.propTypes = {
    original: PropTypes.shape({
        deployment: PropTypes.shape({
            inactive: PropTypes.bool.isRequired,
            name: PropTypes.string.isRequired
        }).isRequired
    }).isRequired
};

// Display the policy description and name.
// ////////////////////////////////////////
function PolicyColumn({ original }) {
    return (
        <Tooltip
            placement="top"
            mouseLeaveDelay={0}
            overlay={<div>{original.policy.description}</div>}
            overlayClassName="pointer-events-none text-white rounded max-w-xs p-2 text-sm text-center"
        >
            <div className="inline-block hover:text-primary-700 underline">
                {original.policy.name}
            </div>
        </Tooltip>
    );
}

PolicyColumn.propTypes = {
    original: PropTypes.shape({
        policy: PropTypes.shape({
            description: PropTypes.string.isRequired,
            name: PropTypes.string.isRequired
        }).isRequired
    }).isRequired
};

// Display the enforcement.
// ////////////////////////
function EnforcementColumn({ original }) {
    const count = original.enforcementCount;
    if (original.lifecycleStage === 'DEPLOY') {
        const message = count === 0 ? 'No' : 'Yes';
        return <span>{message}</span>;
    }
    const countMessage = `${count} ${pluralize('time', count)}`;
    const message = count === 0 ? 'No' : countMessage;
    return <span>{message}</span>;
}

EnforcementColumn.propTypes = {
    original: PropTypes.shape({
        lifecycleStage: PropTypes.string.isRequired,
        enforcementCount: PropTypes.number.isRequired
    }).isRequired
};

// Display the severity.
// /////////////////////
const getSeverityClassName = severityValue => {
    const severityClassMapping = {
        Low: 'px-2 rounded-full bg-base-200 border-2 border-base-300 text-base-600',
        Medium: 'px-2 rounded-full bg-warning-200 border-2 border-warning-300 text-warning-800',
        High: 'px-2 rounded-full bg-caution-200 border-2 border-caution-300 text-caution-800',
        Critical: 'px-2 rounded-full bg-alert-200 border-2 border-alert-300 text-alert-800'
    };
    const res = severityClassMapping[severityValue];
    if (res) return res;
    throw new Error(`Unknown severity: ${severityValue}`);
};

function SeverityColumn({ value }) {
    const severity = severityLabels[value];
    return <div className={getSeverityClassName(severity)}>{severity}</div>;
}

SeverityColumn.propTypes = {
    value: PropTypes.string.isRequired
};

// Display the categories.
// ///////////////////////
function CategoryColumn({ value }) {
    return value.length > 1 ? (
        <Tooltip
            placement="top"
            mouseLeaveDelay={0}
            overlay={<div>{value.join(' | ')}</div>}
            overlayClassName="pointer-events-none text-white rounded max-w-xs p-2 w-full text-sm text-center"
        >
            <div>Multiple</div>
        </Tooltip>
    ) : (
        value[0]
    );
}

CategoryColumn.propTypes = {
    value: PropTypes.arrayOf(PropTypes.string).isRequired
};

// Display the action buttons when hovered.
// ////////////////////////////////////////
function getActionButtonsColumn(setSelectedAlertId) {
    // eslint-disable-next-line react/prop-types
    return ({ original }) => {
        return (
            <ViolationActionButtons violation={original} setSelectedAlertId={setSelectedAlertId} />
        );
    };
}

// Combine all of the above into a column array.
// /////////////////////////////////////////////
export default function getColumns(setSelectedAlertId) {
    return [
        {
            Header: 'Deployment',
            accessor: 'deployment.name',
            searchField: 'Deployment',
            headerClassName: `w-1/6 sticky-column left-checkbox-offset ${defaultHeaderClassName}`,
            className: `w-1/6 sticky-column left-checkbox-offset ${wrapClassName} ${defaultColumnClassName}`,
            Cell: DeploymentColumn
        },
        {
            Header: 'Cluster',
            accessor: 'deployment.clusterName',
            searchField: 'Cluster',
            headerClassName: `w-1/7  ${defaultHeaderClassName}`,
            className: `w-1/7 ${wrapClassName} ${defaultColumnClassName}`,
            Cell: StringValueColumn
        },
        {
            Header: 'Namespace',
            accessor: 'deployment.namespace',
            searchField: 'Namespace',
            headerClassName: `w-1/7 ${defaultHeaderClassName}`,
            className: `w-1/7 ${wrapClassName} ${defaultColumnClassName}`,
            Cell: StringValueColumn
        },
        {
            Header: 'Policy',
            accessor: 'policy.name',
            searchField: 'Policy',
            headerClassName: `w-1/6 ${defaultHeaderClassName}`,
            className: `w-1/6 ${wrapClassName} ${defaultColumnClassName}`,
            Cell: PolicyColumn
        },
        {
            Header: 'Enforced',
            accessor: 'Enforcement Count',
            searchField: 'Policy',
            headerClassName: `w-1/10  ${defaultHeaderClassName}`,
            className: `w-1/10 ${wrapClassName} ${defaultColumnClassName}`,
            Cell: EnforcementColumn
        },
        {
            Header: 'Severity',
            accessor: 'policy.severity',
            searchField: 'Severity',
            headerClassName: `text-center ${defaultHeaderClassName}`,
            className: `text-center ${wrapClassName} ${defaultColumnClassName}`,
            Cell: SeverityColumn,
            sortMethod: sortSeverity,
            width: 90
        },
        {
            Header: 'Categories',
            accessor: 'policy.categories',
            searchField: 'Category',
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${wrapClassName} ${defaultColumnClassName}`,
            Cell: CategoryColumn
        },
        {
            Header: 'Lifecycle',
            accessor: 'lifecycleStage',
            searchField: 'Lifecycle Stage',
            headerClassName: `${defaultHeaderClassName}`,
            className: `${wrapClassName} ${defaultColumnClassName}`,
            Cell: ({ value }) => lifecycleStageLabels[value]
        },
        {
            Header: 'Time',
            accessor: 'time',
            searchField: 'Violation Time',
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${wrapClassName} ${defaultColumnClassName}`,
            Cell: ({ value }) => dateFns.format(value, dateTimeFormat),
            sortMethod: sortDate
        },
        {
            Header: '',
            accessor: '',
            headerClassName: 'hidden',
            className: rtTrActionsClassName,
            Cell: getActionButtonsColumn(setSelectedAlertId)
        }
    ];
}
