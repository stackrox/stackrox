import React, { ReactElement } from 'react';
import pluralize from 'pluralize';
import dateFns from 'date-fns';
import startCase from 'lodash/startCase';
import { Button, ButtonVariant, Flex, FlexItem, Tooltip } from '@patternfly/react-core';

import PolicySeverityIconText from 'Components/PatternFly/IconText/PolicySeverityIconText';
import LinkShim from 'Components/PatternFly/LinkShim';
import dateTimeFormat from 'constants/dateTimeFormat';
import { lifecycleStageLabels } from 'messages/common';
import {
    BLOCKING_ENFORCEMENT_ACTIONS,
    ENFORCEMENT_ACTIONS_AS_PAST_TENSE,
} from 'constants/enforcementActions';
import LIFECYCLE_STAGES from 'constants/lifecycleStages';
import { violationsBasePath } from 'routePaths';
import { ListAlert } from 'types/alert.proto';

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

    const entityPath = namespace ? `${clusterName}/${namespace}` : clusterName;

    return (
        <Flex direction={{ default: 'column' }}>
            <FlexItem className="pf-u-mb-0">{name}</FlexItem>
            <FlexItem className="pf-u-color-200 pf-u-font-size-xs">{`in "${entityPath}"`}</FlexItem>
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
        Cell: ({ value }): string => startCase(value.toLowerCase()),
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
            return <PolicySeverityIconText severity={value} />;
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
