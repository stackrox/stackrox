import React from 'react';
import PropTypes from 'prop-types';
import pluralize from 'pluralize';
import dateFns from 'date-fns';
import dateTimeFormat from 'constants/dateTimeFormat';
import { Tooltip, TooltipOverlay } from '@stackrox/ui-components';

import { sortDate, sortSeverity } from 'sorters/sorters';
import { severityLabels, lifecycleStageLabels } from 'messages/common';
import {
    defaultHeaderClassName,
    defaultColumnClassName,
    rtTrActionsClassName,
} from 'Components/Table';
import {
    BLOCKING_ENFORCEMENT_ACTIONS,
    ENFORCEMENT_ACTIONS_AS_PAST_TENSE,
} from 'constants/enforcementActions';
import LIFECYCLE_STAGES from 'constants/lifecycleStages';
import ViolationActionButtons from './ViolationActionButtons';

// Display the enforcement.
// ////////////////////////
function EnforcementColumn({ original }) {
    if (BLOCKING_ENFORCEMENT_ACTIONS.has(original.enforcementAction)) {
        const message = `${ENFORCEMENT_ACTIONS_AS_PAST_TENSE[original?.enforcementAction]}`;
        return <div className="text-alert-700">{message}</div>;
    }

    const count = original?.enforcementCount;
    if (original?.lifecycleStage === LIFECYCLE_STAGES.DEPLOY) {
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
        enforcementCount: PropTypes.number.isRequired,
        enforcementAction: PropTypes.string.isRequired,
        state: PropTypes.string.isRequired,
    }).isRequired,
};

// Display the severity.
// /////////////////////
const getSeverityClassName = (severityValue) => {
    const severityClassMapping = {
        Low: 'px-2 rounded-full bg-base-200 border-2 border-base-300 text-base-600',
        Medium: 'px-2 rounded-full bg-warning-200 border-2 border-warning-300 text-warning-800',
        High: 'px-2 rounded-full bg-caution-200 border-2 border-caution-300 text-caution-800',
        Critical: 'px-2 rounded-full bg-alert-200 border-2 border-alert-300 text-alert-800',
    };
    const res = severityClassMapping[severityValue];
    if (res) {
        return res;
    }
    throw new Error(`Unknown severity: ${severityValue}`);
};

// Because of fixed checkbox width, total of column ratios must be less than 100%
// 2 * 1/7 + 7 * 1/10 = 98.6%
export default function getColumns(setSelectedAlertId) {
    return [
        {
            Header: 'Entity',
            accessor: 'resource.name',
            searchField: 'Entity',
            headerClassName: `w-1/7 ${defaultHeaderClassName}`,
            className: `w-1/7 ${defaultColumnClassName}`,
            Cell: ({ original }) => {
                const { commonEntityInfo, resource, deployment } = original;
                const { name } = resource || deployment;
                const { namespace, clusterName } = commonEntityInfo;
                return (
                    <div className="flex flex-col">
                        <div className="flex items-center">{name}</div>
                        <div className="text-base-500 text-sm">{`in "${clusterName}/${namespace}"`}</div>
                    </div>
                );
            },
            sortable: false,
        },
        {
            Header: 'Type',
            accessor: 'commonEntityInfo.resourceType',
            searchField: 'Entity Type',
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            Cell: ({ value }) => <span className="capitalize">{value.toLowerCase()}</span>,
            sortable: false,
        },
        {
            Header: 'Policy',
            accessor: 'policy.name',
            searchField: 'Policy',
            headerClassName: `w-1/7 ${defaultHeaderClassName}`,
            className: `w-1/7 ${defaultColumnClassName}`,
            Cell: ({ original }) => (
                <Tooltip
                    content={
                        <TooltipOverlay>
                            {original?.policy?.description || 'No description available'}
                        </TooltipOverlay>
                    }
                >
                    <div className="inline-block hover:text-primary-700 underline">
                        {original?.policy?.name}
                    </div>
                </Tooltip>
            ),
        },
        {
            Header: 'Enforced',
            accessor: 'Enforcement Count',
            searchField: 'Enforcement',
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            Cell: EnforcementColumn,
        },
        {
            Header: 'Severity',
            accessor: 'policy.severity',
            searchField: 'Severity',
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            Cell: ({ value }) => {
                const severity = severityLabels[value];
                return <div className={getSeverityClassName(severity)}>{severity}</div>;
            },
            sortMethod: sortSeverity,
        },
        {
            Header: 'Categories',
            accessor: 'policy.categories',
            searchField: 'Category',
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            Cell: ({ value }) => {
                return value.length > 1 ? (
                    <Tooltip content={<TooltipOverlay>{value.join(' | ')}</TooltipOverlay>}>
                        <div>Multiple</div>
                    </Tooltip>
                ) : (
                    value[0]
                );
            },
        },
        {
            Header: 'Lifecycle',
            accessor: 'lifecycleStage',
            searchField: 'Lifecycle Stage',
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            Cell: ({ value }) => lifecycleStageLabels[value],
        },
        {
            Header: 'Time',
            accessor: 'time',
            searchField: 'Violation Time',
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            Cell: ({ value }) => dateFns.format(value, dateTimeFormat),
            sortMethod: sortDate,
        },
        {
            Header: '',
            accessor: '',
            headerClassName: 'hidden',
            className: rtTrActionsClassName,
            Cell: ({ original }) => (
                <ViolationActionButtons
                    violation={original}
                    setSelectedAlertId={setSelectedAlertId}
                />
            ),
        },
    ];
}
