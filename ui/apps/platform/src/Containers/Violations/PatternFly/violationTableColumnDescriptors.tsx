import React, { ReactElement } from 'react';
import pluralize from 'pluralize';
import dateFns from 'date-fns';
import { Flex, FlexItem } from '@patternfly/react-core';
import { Tooltip, TooltipOverlay } from '@stackrox/ui-components';

import dateTimeFormat from 'constants/dateTimeFormat';
import { severityLabels, lifecycleStageLabels } from 'messages/common';
import {
    BLOCKING_ENFORCEMENT_ACTIONS,
    ENFORCEMENT_ACTIONS_AS_PAST_TENSE,
} from 'constants/enforcementActions';
import LIFECYCLE_STAGES from 'constants/lifecycleStages';

type EntityTableCellProps = {
    original: {
        commonEntityInfo: {
            namespace: string;
            clusterName: string;
        };
        resource?: { name: string };
        deployment: { name: string };
    };
};

function EntityTableCell({ original }: EntityTableCellProps): ReactElement {
    const { commonEntityInfo, resource, deployment } = original;
    const { name } = resource || deployment;
    const { namespace, clusterName } = commonEntityInfo;
    return (
        <Flex
            direction={{ default: 'column' }}
            // justifyContent={{ default: 'justifyContentSpaceAround' }}
        >
            <FlexItem>{name}</FlexItem>
            <FlexItem>{`in "${clusterName}/${namespace}"`}</FlexItem>
        </Flex>
    );
}

type CategoriesTableCellProps = {
    value: string[];
};

function CategoriesTableCell({ value }: CategoriesTableCellProps): ReactElement {
    return value.length > 1 ? (
        <Tooltip content={<TooltipOverlay>{value.join(' | ')}</TooltipOverlay>}>
            <div>Multiple</div>
        </Tooltip>
    ) : (
        <span>{value[0]}</span>
    );
}

type EnforcementTableCellProps = {
    original: {
        lifecycleStage: string;
        enforcementCount: number;
        enforcementAction: string;
        state: string;
    };
};

// Display the enforcement.
// ////////////////////////
function EnforcementColumn({ original }: EnforcementTableCellProps): ReactElement {
    if (BLOCKING_ENFORCEMENT_ACTIONS.has(original.enforcementAction)) {
        const message = `${
            ENFORCEMENT_ACTIONS_AS_PAST_TENSE[original?.enforcementAction] as string
        }`;
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

// Display the severity.
// /////////////////////
const getSeverityClassName = (severityValue: string): string => {
    const severityClassMapping = {
        Low: 'px-2 rounded-full bg-base-200 border-2 border-base-300 text-base-600',
        Medium: 'px-2 rounded-full bg-warning-200 border-2 border-warning-300 text-warning-800',
        High: 'px-2 rounded-full bg-caution-200 border-2 border-caution-300 text-caution-800',
        Critical: 'px-2 rounded-full bg-alert-200 border-2 border-alert-300 text-alert-800',
    };
    const res = severityClassMapping[severityValue] as string;
    if (res) {
        return res;
    }
    throw new Error(`Unknown severity: ${severityValue}`);
};

const tableColumnDescriptor = [
    {
        Header: 'Entity',
        accessor: 'resource.name',
        Cell: EntityTableCell,
    },
    {
        Header: 'Type',
        accessor: 'commonEntityInfo.resourceType',
        Cell: ({ value }) => <span className="capitalize">{value.toLowerCase()}</span>,
    },
    {
        Header: 'Policy',
        accessor: 'policy.name',
        sortField: 'Policy',
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
        accessor: 'enforcementCount',
        sortField: 'Policy',
        Cell: EnforcementColumn,
    },
    {
        Header: 'Severity',
        accessor: 'policy.severity',
        sortField: 'Severity',
        Cell: ({ value }) => {
            const severity = severityLabels[value];
            return <div className={getSeverityClassName(severity)}>{severity}</div>;
        },
    },
    {
        Header: 'Categories',
        accessor: 'policy.categories',
        sortField: 'Category',
        Cell: CategoriesTableCell,
    },
    {
        Header: 'Lifecycle',
        accessor: 'lifecycleStage',
        sortField: 'Lifecycle Stage',
        Cell: ({ value }): ReactElement => <span>{lifecycleStageLabels[value]}</span>,
    },
    {
        Header: 'Time',
        accessor: 'time',
        sortField: 'Violation Time',
        Cell: ({ value }) => dateFns.format(value, dateTimeFormat),
    },
];

export default tableColumnDescriptor;
