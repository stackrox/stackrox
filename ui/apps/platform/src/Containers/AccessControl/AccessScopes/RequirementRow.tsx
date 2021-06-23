/* eslint-disable react/jsx-no-bind */
import React, { ReactElement, useEffect, useRef, useState } from 'react';
import { Button, TextInput, Tooltip, ValidatedOptions } from '@patternfly/react-core';
import {
    ArrowCircleDownIcon,
    CheckCircleIcon,
    MinusCircleIcon,
    PlusCircleIcon,
    PencilAltIcon,
    TimesCircleIcon,
} from '@patternfly/react-icons';
import { Td, Tr } from '@patternfly/react-table';

import {
    LabelSelectorOperator,
    LabelSelectorRequirement,
    getIsKeyExistsOperator,
    getIsKeyInSetOperator,
} from 'services/RolesService';
import { getIsValidLabelKey, getIsValidLabelValue } from 'utils/labels'; // getIsValidLabelValue

import { Activity } from './accessScopes.utils';

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
            return 'unknown';
    }
}

function getValueText(value: string): string {
    return value || '""';
}

function getIsOverlappingRequirement(
    key: string,
    op: LabelSelectorOperator,
    requirements: LabelSelectorRequirement[]
) {
    return requirements.some((requirement) => {
        if (key === requirement.key) {
            if (op === requirement.op) {
                // Prevent same key and op because:
                // redundant for exists
                // confusing for set, because effective requirement is intersection of values
                // however, "key in (...)" and "key not in (...)" are possible
                return true;
            }

            if (getIsKeyExistsOperator(op) && getIsKeyExistsOperator(requirement.op)) {
                // Prevent "key exists" and "key not exists" because they contradict each other.
                return true;
            }
        }

        return false;
    });
}

/*
 * Render a temporary row to enter the key for a new requirement.
 */

export type RequirementRowAddKeyProps = {
    op: LabelSelectorOperator;
    requirements: LabelSelectorRequirement[];
    handleRequirementKeyOK: (key: string) => void;
    handleRequirementKeyCancel: () => void;
};

export function RequirementRowAddKey({
    op,
    requirements,
    handleRequirementKeyOK,
    handleRequirementKeyCancel,
}: RequirementRowAddKeyProps): ReactElement {
    const refKeyInput = useRef<null | HTMLInputElement>(null); // for focus after initial rendering
    const [keyInput, setKeyInput] = useState('');

    useEffect(() => {
        if (typeof refKeyInput?.current?.focus === 'function') {
            refKeyInput.current.focus();
        }
    }, []);

    const isKeyInSet = getIsKeyInSetOperator(op);

    const isOverlappingRequirement = getIsOverlappingRequirement(keyInput, op, requirements);
    const isInvalidKey = !getIsValidLabelKey(keyInput);
    const isDisabledOK = isInvalidKey || isOverlappingRequirement;

    let validatedKey: ValidatedOptions = ValidatedOptions.default;
    if (keyInput) {
        validatedKey = isDisabledOK ? ValidatedOptions.error : ValidatedOptions.success;
    }

    function onKeyChange(keyChange: string) {
        setKeyInput(keyChange);
    }

    function onClickRequirementKeyOK() {
        handleRequirementKeyOK(keyInput);
    }

    function onKeyDown(event) {
        if (event.code === 'Escape') {
            handleRequirementKeyCancel();
        } else if (event.code === 'Enter' || event.code === 'Tab') {
            if (!isDisabledOK) {
                onClickRequirementKeyOK();
            }
        }
    }

    return (
        <Tr>
            <Td dataLabel="Key">
                <div className="pf-u-display-flex">
                    <span className="pf-u-flex-basis-0 pf-u-flex-grow-1 pf-u-flex-shrink-1 pf-u-text-break-word">
                        <TextInput
                            aria-label="Type a key"
                            value={keyInput}
                            validated={validatedKey}
                            onChange={onKeyChange}
                            onKeyDown={onKeyDown}
                            ref={refKeyInput}
                            className="pf-m-small"
                        />
                    </span>
                    {isKeyInSet && (
                        <span className="pf-u-flex-shrink-0">
                            <Tooltip content="Requirement key OK">
                                <Button
                                    aria-label="Requirement key OK"
                                    variant="plain"
                                    className="pf-m-smallest pf-u-ml-sm"
                                    isDisabled={isDisabledOK}
                                    onClick={onClickRequirementKeyOK}
                                >
                                    <ArrowCircleDownIcon
                                        color="var(--pf-global--primary-color--100)"
                                        style={{ transform: 'rotate(-90deg)' }}
                                    />
                                </Button>
                            </Tooltip>
                        </span>
                    )}
                </div>
                {keyInput.length !== 0 && isInvalidKey && (
                    <p className="pf-u-font-size-sm pf-u-danger-color-100">Invalid key</p>
                )}
                {isOverlappingRequirement && (
                    <p className="pf-u-font-size-sm pf-u--color-100">
                        A requirement overlaps with this key and operator
                    </p>
                )}
            </Td>
            <Td dataLabel="Operator">{getOpText(op)}</Td>
            <Td dataLabel="Values" />
            <Td dataLabel="Action" className="pf-u-text-align-right">
                {!isKeyInSet && (
                    <Tooltip key="OK" content="OK">
                        <Button
                            aria-label="OK"
                            variant="plain"
                            className="pf-m-smallest pf-u-mr-sm"
                            isDisabled={isKeyInSet || isDisabledOK}
                            onClick={onClickRequirementKeyOK}
                        >
                            <CheckCircleIcon color="var(--pf-global--primary-color--100)" />
                        </Button>
                    </Tooltip>
                )}
                <Tooltip key="Cancel" content="Cancel">
                    <Button
                        aria-label="Cancel"
                        variant="plain"
                        className="pf-m-smallest"
                        onClick={handleRequirementKeyCancel}
                    >
                        <TimesCircleIcon color="var(--pf-global--color--100)" />
                    </Button>
                </Tooltip>
            </Td>
        </Tr>
    );
}

/*
 * Render a requirement with editing interaction if hasAction.
 */

export type RequirementRowProps = {
    requirement: LabelSelectorRequirement;
    isOnlyRequirement: boolean;
    hasAction: boolean;
    activity: Activity;
    handleRequirementDelete: () => void;
    handleRequirementEdit: () => void;
    handleRequirementOK: () => void;
    handleRequirementCancel: () => void;
    handleValueAdd: (value: string) => void;
    handleValueDelete: (indexValue: number) => void;
};

export function RequirementRow({
    requirement,
    isOnlyRequirement,
    hasAction,
    activity,
    handleRequirementDelete,
    handleRequirementEdit,
    handleRequirementOK,
    handleRequirementCancel,
    handleValueAdd,
    handleValueDelete,
}: RequirementRowProps): ReactElement {
    const refValueInput = useRef<null | HTMLInputElement>(null); // for focus after adding a label
    const [valueInput, setValueInput] = useState('');

    const { key, op, values } = requirement;
    const isKeyInSet = getIsKeyInSetOperator(op);
    const isRequirementActive = activity === 'ACTIVE';

    const isInvalidValue = !getIsValidLabelValue(valueInput);
    const isDuplicateValue = values.includes(valueInput);
    const isDisabledAddValue = isInvalidValue || isDuplicateValue;

    let validatedValue: ValidatedOptions = ValidatedOptions.default;
    if (valueInput) {
        validatedValue =
            isInvalidValue || isDuplicateValue ? ValidatedOptions.error : ValidatedOptions.success;
    }

    function onAddValue() {
        handleValueAdd(valueInput);
        setValueInput('');
        if (typeof refValueInput?.current?.focus === 'function') {
            refValueInput.current.focus();
        }
    }

    function onKeyDown(event) {
        if (event.code === 'Enter' && !isDisabledAddValue) {
            onAddValue();
        }
    }

    return (
        <Tr>
            <Td dataLabel="Key">{key}</Td>
            <Td dataLabel="Operator">{getOpText(op)}</Td>
            <Td dataLabel="Values">
                {isKeyInSet &&
                    values.map((value, indexValue) => (
                        <div key={value} className="pf-u-display-flex">
                            <span className="pf-u-flex-basis-0 pf-u-flex-grow-1 pf-u-flex-shrink-1 pf-u-text-break-word">
                                {getValueText(value)}
                            </span>
                            <span className="pf-u-flex-shrink-0 pf-u-pl-sm">
                                {isRequirementActive && (
                                    <Tooltip content="Delete value">
                                        <Button
                                            aria-label="Delete value"
                                            variant="plain"
                                            className="pf-m-smallest pf-u-mr-sm"
                                            isDisabled={values.length === 1}
                                            onClick={() => handleValueDelete(indexValue)}
                                        >
                                            <MinusCircleIcon color="var(--pf-global--danger-color--100)" />
                                        </Button>
                                    </Tooltip>
                                )}
                            </span>
                        </div>
                    ))}
                {isRequirementActive && (
                    <>
                        {/* values.length === 1 && (
                            <div className="pf-u-font-size-sm pf-u-info-color-100">
                                If you need to replace the last value, first add its replacement
                            </div>
                        ) */}
                        <div className="pf-u-display-flex pf-u-align-items-center">
                            <span className="pf-u-flex-basis-0 pf-u-flex-grow-1 pf-u-flex-shrink-1">
                                <TextInput
                                    aria-label="Type a value"
                                    value={valueInput}
                                    validated={validatedValue}
                                    onChange={setValueInput}
                                    onKeyDown={onKeyDown}
                                    ref={refValueInput}
                                    className="pf-m-small"
                                />
                            </span>
                            <span className="pf-u-flex-shrink-0 pf-u-pl-sm">
                                {isRequirementActive && (
                                    <Tooltip content="Add value">
                                        <Button
                                            aria-label="Add value"
                                            variant="plain"
                                            className="pf-m-smallest pf-u-mr-sm"
                                            isDisabled={isDisabledAddValue}
                                            onClick={onAddValue}
                                        >
                                            <PlusCircleIcon color="var(--pf-global--primary-color--100)" />
                                        </Button>
                                    </Tooltip>
                                )}
                            </span>
                        </div>
                        {isInvalidValue && (
                            <div className="pf-u-font-size-sm pf-u-danger-color-100">
                                Invalid label value
                            </div>
                        )}
                        {isDuplicateValue && (
                            <div className="pf-u-font-size-sm pf-u-danger-color-100">
                                Duplicate label value
                            </div>
                        )}
                    </>
                )}
            </Td>
            {hasAction && (
                <Td dataLabel="Action" className="pf-u-text-align-right">
                    {isRequirementActive ? (
                        <>
                            <Tooltip key="OK" content="OK">
                                <Button
                                    aria-label="OK"
                                    variant="plain"
                                    className="pf-m-smallest pf-u-mr-sm"
                                    isDisabled={values.length === 0 || valueInput.length !== 0}
                                    onClick={handleRequirementOK}
                                >
                                    <CheckCircleIcon color="var(--pf-global--primary-color--100)" />
                                </Button>
                            </Tooltip>
                            <Tooltip key="Cancel" content="Cancel">
                                <Button
                                    aria-label="Cancel"
                                    variant="plain"
                                    className="pf-m-smallest"
                                    onClick={handleRequirementCancel}
                                >
                                    <TimesCircleIcon color="var(--pf-global--color--100)" />
                                </Button>
                            </Tooltip>
                        </>
                    ) : (
                        <>
                            {isKeyInSet && (
                                <Tooltip key="Edit requirement" content="Edit requirement">
                                    <Button
                                        aria-label="Edit requirement"
                                        variant="plain"
                                        className="pf-m-smallest pf-u-mr-sm"
                                        isDisabled={activity === 'DISABLED'}
                                        onClick={handleRequirementEdit}
                                    >
                                        <PencilAltIcon color="var(--pf-global--primary-color--100)" />
                                    </Button>
                                </Tooltip>
                            )}
                            <Tooltip key="Delete requirement" content="Delete requirement">
                                <Button
                                    aria-label="Delete requirement"
                                    variant="plain"
                                    className="pf-m-smallest"
                                    isDisabled={activity === 'DISABLED' || isOnlyRequirement}
                                    onClick={handleRequirementDelete}
                                >
                                    <MinusCircleIcon color="var(--pf-global--danger-color--100)" />
                                </Button>
                            </Tooltip>
                        </>
                    )}
                </Td>
            )}
        </Tr>
    );
}
