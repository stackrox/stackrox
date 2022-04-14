import React, { ReactElement } from 'react';
import pluralize from 'pluralize';
import dateFns from 'date-fns';
import { Button, ButtonVariant, Flex, FlexItem, Tooltip, Label } from '@patternfly/react-core';

import LinkShim from 'Components/PatternFly/LinkShim';
import dateTimeFormat from 'constants/dateTimeFormat';
import { severityColorMapPF } from 'constants/severityColors';
import { severityLabels, lifecycleStageLabels } from 'messages/common';
import {
    BLOCKING_ENFORCEMENT_ACTIONS,
    ENFORCEMENT_ACTIONS_AS_PAST_TENSE,
} from 'constants/enforcementActions';
import LIFECYCLE_STAGES from 'constants/lifecycleStages';
import { violationsBasePath } from 'routePaths';
import { ListAlert } from './types/violationTypes';

type EntityTableCellProps = {
    // original: ListAlert;
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
        <Flex direction={{ default: 'column' }}>
            <FlexItem className="pf-u-mb-0">{name}</FlexItem>
            <FlexItem className="pf-u-color-400 pf-u-font-size-xs">
                {`in "${clusterName}/${namespace}"`}
            </FlexItem>
        </Flex>
    );
}

type CategoriesTableCellProps = {
    value: string[];
};

function CategoriesTableCell({ value }: CategoriesTableCellProps): ReactElement {
    return value.length > 1 ? (
        <Tooltip content={value.join(' | ')}>
            <span>Multiple</span>
        </Tooltip>
    ) : (
        <span>{value[0]}</span>
    );
}

type EnforcementTableCellProps = {
    original: ListAlert;
};

// Display the enforcement.
// ////////////////////////
function EnforcementColumn({ original }: EnforcementTableCellProps): ReactElement {
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

const tableColumnDescriptor = [
    {
        Header: 'Policy',
        accessor: 'policy.name',
        sortField: 'Policy',
        Cell: ({ original }) => {
            const url = `${violationsBasePath}/${original.id as string}`;
            return (
                <Tooltip content={original?.policy?.description || 'No description available'}>
                    <Button variant={ButtonVariant.link} isInline component={LinkShim} href={url}>
                        {original?.policy?.name}
                    </Button>
                </Tooltip>
            );
        },
    },
    {
        Header: 'Entity',
        accessor: 'resource.name',
        Cell: EntityTableCell,
    },
    {
        Header: 'Type',
        accessor: 'commonEntityInfo.resourceType',
        Cell: ({ value }): string => value.toLowerCase() as string,
    },
    {
        Header: 'Enforced',
        accessor: 'enforcementCount',
        sortField: 'Enforcement',
        Cell: EnforcementColumn,
    },
    {
        Header: 'Severity',
        accessor: 'policy.severity',
        sortField: 'Severity',
        Cell: ({ value }) => {
            const severity = severityLabels[value];
            return <Label color={severityColorMapPF[severity]}>{severity}</Label>;
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
        Cell: ({ value }): string => dateFns.format(value, dateTimeFormat),
    },
];

export default tableColumnDescriptor;
