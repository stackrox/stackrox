/* eslint-disable react/jsx-no-bind */
/* eslint-disable react/no-array-index-key */
import React, { ReactElement } from 'react';
import { Button, Toolbar, ToolbarContent, ToolbarItem, Tooltip } from '@patternfly/react-core';
import { TimesCircleIcon } from '@patternfly/react-icons';
import { TableComposable, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import {
    LabelSelectorOperator,
    LabelSelectorRequirement,
    LabelSelectorsKey,
} from 'services/RolesService';

import AddRequirementDropdown from './AddRequirementDropdown';

/*
 * For more information about label selectors:
 * https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/
 */

function getOpText(op: LabelSelectorOperator): string {
    switch (op) {
        case 'IN':
            return 'in';
        case 'NOT_IN':
            return 'not in';
        case 'EXISTS':
            return 'exists';
        case 'NOT_EXISTS':
            return 'not exists';
        default:
            return '';
    }
}

function getValueText(value: string): string {
    return value || '""';
}

const infoValues = {
    ariaLabel: 'in: key has one of the values; not in: key does not have any of the values',
    tooltip: (
        <div>
            in: key has one of the values
            <br />
            not in: key does not have any of the values
        </div>
    ),
    tooltipProps: {
        isContentLeftAligned: true,
        maxWidth: '24rem',
    },
};

export function LabelSelectorCreatable(): ReactElement {
    function onAddRequirement(op: LabelSelectorOperator) {
        console.log('onAddRequirement', op); // eslint-disable-line no-console
    }

    function onAddLabelSelector() {
        console.log('onAddLabelSelector'); // eslint-disable-line no-console
    }

    return (
        <>
            <TableComposable variant="compact">
                <Thead>
                    <Tr>
                        <Th modifier="breakWord">Key</Th>
                        <Th modifier="fitContent">Operator</Th>
                        <Th modifier="breakWord" info={infoValues}>
                            Values
                        </Th>
                        <Th modifier="fitContent">Action</Th>
                    </Tr>
                </Thead>
            </TableComposable>
            <Toolbar className="pf-u-pb-0" inset={{ default: 'insetNone' }}>
                <ToolbarContent>
                    <ToolbarItem>
                        <AddRequirementDropdown onAddRequirement={onAddRequirement} isDisabled />
                    </ToolbarItem>
                    <ToolbarItem alignment={{ default: 'alignRight' }}>
                        <Button
                            variant="primary"
                            isSmall
                            isDisabled
                            onClick={() => onAddLabelSelector()}
                        >
                            Add label selector
                        </Button>
                    </ToolbarItem>
                </ToolbarContent>
            </Toolbar>
        </>
    );
}

export type LabelSelectorUpdatableProps = {
    requirements: LabelSelectorRequirement[];
    labelSelectorsKey: LabelSelectorsKey;
    hasAction: boolean;
    handleChangeRequirements: (requirements: LabelSelectorRequirement[]) => void;
};

export function LabelSelectorUpdatable({
    requirements,
    labelSelectorsKey,
    hasAction,
    handleChangeRequirements,
}: LabelSelectorUpdatableProps): ReactElement {
    const classNameRow =
        labelSelectorsKey === 'namespaceLabelSelectors' ? 'pf-u-background-color-200' : '';

    function handleChangeRequirement(
        indexRequirement: number,
        requirementChange: LabelSelectorRequirement
    ) {
        handleChangeRequirements(
            requirements.map((requirement, index) =>
                index === indexRequirement ? requirementChange : requirement
            )
        );
    }

    function onAddRequirement(op: LabelSelectorOperator) {
        console.log('onAddRequirement', op); // eslint-disable-line no-console
    }

    function onDeleteRequirement(indexRequirement: number) {
        console.log('onDeleteRequirement', indexRequirement); // eslint-disable-line no-console
        handleChangeRequirements(requirements.filter((_, index) => index !== indexRequirement));
    }

    function onDeleteValue(indexRequirement: number, indexValue: number) {
        console.log('onDeleteValue', indexRequirement, indexValue); // eslint-disable-line no-console
        const { key, op, values } = requirements[indexRequirement];
        handleChangeRequirement(indexRequirement, {
            key,
            op,
            values: values.filter((_, index) => index !== indexValue),
        });
    }

    return (
        <>
            <TableComposable variant="compact">
                <Thead>
                    <Tr>
                        <Th modifier="breakWord">Key</Th>
                        <Th modifier="fitContent">Operator</Th>
                        <Th modifier="breakWord" info={infoValues}>
                            Values
                        </Th>
                        {hasAction && <Th modifier="fitContent">Action</Th>}
                    </Tr>
                </Thead>
                <Tbody>
                    {requirements.map(({ key, op, values }, indexRequirement) => (
                        <Tr key={key} className={classNameRow}>
                            <Td dataLabel="Key">{key}</Td>
                            <Td dataLabel="Operator">{getOpText(op)}</Td>
                            <Td dataLabel="Values">
                                {values.map((value, indexValue) => (
                                    <div
                                        key={value}
                                        className="pf-u-display-flex pf-u-justify-content-flex-start"
                                    >
                                        <span className="pf-u-flex-basis-0 pf-u-flex-grow-1 pf-u-flex-shrink-1 pf-u-text-break-word">
                                            {getValueText(value)}
                                        </span>
                                        <span className="pf-u-flex-shrink-0 pf-u-pl-sm">
                                            {hasAction && (
                                                <Tooltip content="Delete value">
                                                    <Button
                                                        aria-label="Delete value"
                                                        variant="plain"
                                                        className="pf-m-smallest pf-u-mr-sm"
                                                        isDisabled={values.length === 1}
                                                        onClick={() =>
                                                            onDeleteValue(
                                                                indexRequirement,
                                                                indexValue
                                                            )
                                                        }
                                                    >
                                                        <TimesCircleIcon color="var(--pf-global--danger-color--100)" />
                                                    </Button>
                                                </Tooltip>
                                            )}
                                        </span>
                                    </div>
                                ))}
                            </Td>
                            {hasAction && (
                                <Td dataLabel="Action" className="pf-u-text-align-right">
                                    <Tooltip content="Delete requirement">
                                        <Button
                                            aria-label="Delete requirement"
                                            variant="plain"
                                            className="pf-m-smallest"
                                            isDisabled={requirements.length === 1}
                                            onClick={() => onDeleteRequirement(indexRequirement)}
                                        >
                                            <TimesCircleIcon color="var(--pf-global--danger-color--100)" />
                                        </Button>
                                    </Tooltip>
                                </Td>
                            )}
                        </Tr>
                    ))}
                </Tbody>
            </TableComposable>
            {hasAction && (
                <Toolbar className="pf-u-pb-0" inset={{ default: 'insetNone' }}>
                    <ToolbarContent>
                        <ToolbarItem>
                            <AddRequirementDropdown
                                onAddRequirement={onAddRequirement}
                                isDisabled
                            />
                        </ToolbarItem>
                    </ToolbarContent>
                </Toolbar>
            )}
        </>
    );
}
