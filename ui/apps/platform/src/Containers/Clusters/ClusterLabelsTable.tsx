import React, { ReactElement, useRef, useState } from 'react';
import { Button, Icon, TextInput, Tooltip, ValidatedOptions } from '@patternfly/react-core';
import { PlusCircleIcon, TimesCircleIcon } from '@patternfly/react-icons';
import { Table, Tbody, Td, Thead, Th, Tr } from '@patternfly/react-table';

import { ClusterLabels } from 'services/ClustersService';
import { getIsValidLabelKey, getIsValidLabelValue } from 'utils/labels';

export type ClusterLabelsTableProps = {
    labels: ClusterLabels;
    hasAction: boolean;
    handleChangeLabels: (labels: ClusterLabels) => void;
    isValueRequired?: boolean;
};

/*
 * Render table of cluster labels.
 *
 * If hasAction (always at the moment)
 * render delete buttons at the right of each label row
 * render a row to add a new label or replace an existing label
 */
function ClusterLabelsTable({
    labels,
    hasAction,
    handleChangeLabels,
    isValueRequired,
}: ClusterLabelsTableProps): ReactElement {
    const refKeyInput = useRef<null | HTMLInputElement>(null); // for focus after adding a label
    const [keyInput, setKeyInput] = useState('');
    const [valueInput, setValueInput] = useState('');

    const isValidKey = getIsValidLabelKey(keyInput);
    const isValidValue = getIsValidLabelValue(valueInput, isValueRequired);
    const isValid = isValidKey && isValidValue;

    const isReplace = Object.prototype.hasOwnProperty.call(labels, keyInput); // no-prototype-builtins

    let validatedKey: ValidatedOptions = ValidatedOptions.default;
    if (keyInput) {
        if (isReplace) {
            validatedKey = ValidatedOptions.warning;
        } else {
            validatedKey = isValidKey ? ValidatedOptions.success : ValidatedOptions.error;
        }
    }

    let validatedValue: ValidatedOptions = ValidatedOptions.default;
    if (keyInput || valueInput) {
        validatedValue = isValidValue ? ValidatedOptions.success : ValidatedOptions.error;
    }

    function onAddLabel() {
        handleChangeLabels({ ...labels, [keyInput]: valueInput });
        setKeyInput('');
        setValueInput('');
        if (typeof refKeyInput?.current?.focus === 'function') {
            refKeyInput.current.focus();
        }
    }

    function onKeyPressValue(event) {
        if (event.key === 'Enter' && isValid) {
            onAddLabel();
        }
    }

    function onDeleteLabel(keyDelete: string) {
        const labelsDelete = { ...labels };
        delete labelsDelete[keyDelete];
        handleChangeLabels(labelsDelete);
    }

    return (
        <Table variant="compact">
            <Thead>
                <Tr>
                    <Th>Key</Th>
                    <Th>Value</Th>
                    {hasAction && <Th>Action</Th>}
                </Tr>
            </Thead>
            <Tbody>
                {Object.entries(labels).map(([key, value]) => (
                    <Tr
                        key={key}
                        style={{
                            backgroundColor:
                                key === keyInput
                                    ? 'var(--pf-v5-global--warning-color--100)'
                                    : 'transparent',
                        }}
                    >
                        <Td dataLabel="Key" modifier="breakWord">
                            {key}
                        </Td>
                        <Td dataLabel="Value" modifier="breakWord">
                            {value}
                        </Td>
                        {hasAction && (
                            <Td dataLabel="Action">
                                <Tooltip content="Delete value">
                                    <Button
                                        aria-label="Delete value"
                                        variant="plain"
                                        style={{ padding: 0 }}
                                        onClick={() => onDeleteLabel(key)}
                                    >
                                        <TimesCircleIcon color="var(--pf-v5-global--danger-color--100)" />
                                    </Button>
                                </Tooltip>
                            </Td>
                        )}
                    </Tr>
                ))}
                {hasAction && (
                    <Tr>
                        <Td dataLabel="Key">
                            <TextInput
                                aria-label="Type a label key"
                                value={keyInput}
                                validated={validatedKey}
                                onChange={(_event, val) => setKeyInput(val)}
                                ref={refKeyInput}
                            />
                            {validatedKey === ValidatedOptions.error && (
                                <p className="pf-v5-u-font-size-sm pf-v5-u-danger-color-100">
                                    Invalid label key
                                </p>
                            )}
                            {validatedKey === ValidatedOptions.warning && (
                                <p className="pf-v5-u-font-size-sm pf-v5-u-warning-color-100">
                                    You will replace an existing label which has the same key
                                </p>
                            )}
                        </Td>
                        <Td dataLabel="Value">
                            <TextInput
                                aria-label="Type a label value"
                                value={valueInput}
                                validated={validatedValue}
                                onChange={(_event, val) => setValueInput(val)}
                                onKeyPress={onKeyPressValue}
                            />
                            {validatedValue === ValidatedOptions.error && (
                                <p className="pf-v5-u-font-size-sm pf-v5-u-danger-color-100">
                                    {valueInput.length === 0
                                        ? 'Label value is required'
                                        : 'Invalid label value'}
                                </p>
                            )}
                        </Td>
                        <Td dataLabel="Action">
                            <Tooltip content={isReplace ? 'Replace label' : 'Add label'}>
                                <Button
                                    aria-label={isReplace ? 'Replace label' : 'Add label'}
                                    variant="plain"
                                    style={{ padding: 0 }}
                                    isDisabled={!isValid}
                                    onClick={() => onAddLabel()}
                                >
                                    <Icon>
                                        <PlusCircleIcon
                                            color={
                                                isReplace
                                                    ? 'var(--pf-v5-global--warning-color--100)'
                                                    : 'var(--pf-v5-global--success-color--100)'
                                            }
                                        />
                                    </Icon>
                                </Button>
                            </Tooltip>
                        </Td>
                    </Tr>
                )}
            </Tbody>
        </Table>
    );
}

export default ClusterLabelsTable;
