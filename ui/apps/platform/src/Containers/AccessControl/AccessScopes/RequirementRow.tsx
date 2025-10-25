import { useRef, useState } from 'react';
import type { ReactElement } from 'react';
import { Button, Icon, TextInput, Tooltip, ValidatedOptions } from '@patternfly/react-core';
import {
    CheckCircleIcon,
    MinusCircleIcon,
    PlusCircleIcon,
    PencilAltIcon,
    TimesCircleIcon,
} from '@patternfly/react-icons';
import { Td, Tr } from '@patternfly/react-table';

import type { LabelSelectorRequirement } from 'services/AccessScopesService';
import { getIsValidLabelValue } from 'utils/labels';

import { getIsKeyInSetOperator, getOpText, getValueText } from './accessScopes.utils';
import type { Activity } from './accessScopes.utils';

/*
 * Render a requirement with editing interaction if hasAction.
 */

export type RequirementRowProps = {
    requirement: LabelSelectorRequirement;
    hasAction: boolean;
    activity: Activity;
    handleRequirementDelete: () => void;
    handleRequirementEdit: () => void;
    handleRequirementOK: () => void;
    handleRequirementCancel: () => void;
    handleValueAdd: (value: string) => void;
    handleValueDelete: (indexValue: number) => void;
};

function RequirementRow({
    requirement,
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
    const isEditableOperator = op === 'IN';
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
            <Td dataLabel="Key" modifier="breakWord">
                {key}
            </Td>
            <Td dataLabel="Operator">{getOpText(op, values)}</Td>
            <Td dataLabel="Values" modifier="breakWord">
                {isKeyInSet &&
                    values.map((value, indexValue) => (
                        <div key={value} className="pf-v5-u-display-flex">
                            <span className="pf-v5-u-flex-basis-0 pf-v5-u-flex-grow-1 pf-v5-u-flex-shrink-1 pf-v5-u-text-break-word">
                                {getValueText(value)}
                            </span>
                            <span className="pf-v5-u-flex-shrink-0 pf-v5-u-pl-sm">
                                {isRequirementActive && isEditableOperator && (
                                    <Tooltip content="Delete value">
                                        <Button
                                            aria-label="Delete value"
                                            variant="plain"
                                            className="pf-m-smallest pf-v5-u-mr-sm"
                                            isDisabled={values.length === 1}
                                            onClick={() => handleValueDelete(indexValue)}
                                        >
                                            <Icon>
                                                <MinusCircleIcon color="var(--pf-v5-global--danger-color--100)" />
                                            </Icon>
                                        </Button>
                                    </Tooltip>
                                )}
                            </span>
                        </div>
                    ))}
                {isRequirementActive && (
                    <>
                        <div className="pf-v5-u-display-flex pf-v5-u-align-items-center">
                            <span className="pf-v5-u-flex-basis-0 pf-v5-u-flex-grow-1 pf-v5-u-flex-shrink-1">
                                <TextInput
                                    aria-label="Type a value"
                                    value={valueInput}
                                    validated={validatedValue}
                                    onChange={(_event, val) => setValueInput(val)}
                                    onKeyDown={onKeyDown}
                                    ref={refValueInput}
                                    className="pf-m-small"
                                />
                            </span>
                            <span className="pf-v5-u-flex-shrink-0 pf-v5-u-pl-sm">
                                {isRequirementActive && (
                                    <Tooltip content="Add value (press Enter)">
                                        <Button
                                            aria-label="Add value (press Enter)"
                                            variant="plain"
                                            className="pf-m-smallest pf-v5-u-mr-sm"
                                            isDisabled={isDisabledAddValue}
                                            onClick={onAddValue}
                                        >
                                            <Icon>
                                                <PlusCircleIcon color="var(--pf-v5-global--primary-color--100)" />
                                            </Icon>
                                        </Button>
                                    </Tooltip>
                                )}
                            </span>
                        </div>
                        {isInvalidValue && (
                            <div className="pf-v5-u-font-size-sm pf-v5-u-danger-color-100">
                                Invalid label value
                            </div>
                        )}
                        {isDuplicateValue && (
                            <div className="pf-v5-u-font-size-sm pf-v5-u-danger-color-100">
                                Duplicate label value
                            </div>
                        )}
                    </>
                )}
            </Td>
            {hasAction && (
                <Td dataLabel="Action" className="pf-v5-u-text-align-right">
                    {isRequirementActive ? (
                        <>
                            <Tooltip key="OK" content="OK">
                                <Button
                                    aria-label="OK"
                                    variant="plain"
                                    className="pf-m-smallest pf-v5-u-mr-sm"
                                    isDisabled={values.length === 0 || valueInput.length !== 0}
                                    onClick={handleRequirementOK}
                                >
                                    <Icon>
                                        <CheckCircleIcon color="var(--pf-v5-global--primary-color--100)" />
                                    </Icon>
                                </Button>
                            </Tooltip>
                            <Tooltip key="Cancel" content="Cancel">
                                <Button
                                    aria-label="Cancel"
                                    variant="plain"
                                    className="pf-m-smallest"
                                    onClick={handleRequirementCancel}
                                >
                                    <Icon>
                                        <TimesCircleIcon color="var(--pf-v5-global--color--100)" />
                                    </Icon>
                                </Button>
                            </Tooltip>
                        </>
                    ) : (
                        <>
                            {isEditableOperator && (
                                <Tooltip key="Edit rule" content="Edit rule">
                                    <Button
                                        aria-label="Edit rule"
                                        variant="plain"
                                        className="pf-m-smallest pf-v5-u-mr-sm"
                                        isDisabled={activity === 'DISABLED'}
                                        onClick={handleRequirementEdit}
                                    >
                                        <Icon>
                                            <PencilAltIcon color="var(--pf-v5-global--primary-color--100)" />
                                        </Icon>
                                    </Button>
                                </Tooltip>
                            )}
                            <Tooltip key="Delete rule" content="Delete rule">
                                <Button
                                    aria-label="Delete rule"
                                    variant="plain"
                                    className="pf-m-smallest"
                                    isDisabled={activity === 'DISABLED'}
                                    onClick={handleRequirementDelete}
                                >
                                    <Icon>
                                        <MinusCircleIcon color="var(--pf-v5-global--danger-color--100)" />
                                    </Icon>
                                </Button>
                            </Tooltip>
                        </>
                    )}
                </Td>
            )}
        </Tr>
    );
}

export default RequirementRow;
