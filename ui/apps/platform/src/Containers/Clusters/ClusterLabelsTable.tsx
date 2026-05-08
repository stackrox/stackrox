import { useRef, useState } from 'react';
import type { ReactElement } from 'react';
import { Button, Icon, TextInput, Tooltip, ValidatedOptions } from '@patternfly/react-core';
import { PlusCircleIcon, TimesCircleIcon } from '@patternfly/react-icons';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import type { ClusterLabels } from 'services/ClustersService';
import { getIsValidLabelKey, getIsValidLabelValue } from 'utils/labels';

export type ClusterLabelsTableProps = {
    labels: ClusterLabels;
    handleChangeLabels: (labels: ClusterLabels) => void;
};

/*
 * Editable table of cluster labels: add, replace, and delete rows.
 */
function ClusterLabelsTable({ labels, handleChangeLabels }: ClusterLabelsTableProps): ReactElement {
    const refKeyInput = useRef<null | HTMLInputElement>(null); // for focus after adding a label
    const [keyInput, setKeyInput] = useState('');
    const [valueInput, setValueInput] = useState('');

    const isValidKey = getIsValidLabelKey(keyInput);
    const isValidValue = getIsValidLabelValue(valueInput, true);
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
        <Table variant="compact" aria-label="Cluster labels">
            <Thead>
                <Tr>
                    <Th>Key</Th>
                    <Th>Value</Th>
                    <Th>Action</Th>
                </Tr>
            </Thead>
            <Tbody>
                {Object.entries(labels).map(([key, value]) => (
                    <Tr
                        key={key}
                        style={{
                            backgroundColor:
                                key === keyInput
                                    ? 'var(--pf-t--global--color--status--warning--default)'
                                    : 'transparent',
                        }}
                    >
                        <Td dataLabel="Key" modifier="breakWord">
                            {key}
                        </Td>
                        <Td dataLabel="Value" modifier="breakWord">
                            {value}
                        </Td>
                        <Td dataLabel="Action">
                            <Tooltip content="Delete value">
                                <Button
                                    icon={
                                        <TimesCircleIcon color="var(--pf-t--global--icon--color--status--danger--default)" />
                                    }
                                    aria-label="Delete value"
                                    variant="plain"
                                    style={{ padding: 0 }}
                                    onClick={() => onDeleteLabel(key)}
                                />
                            </Tooltip>
                        </Td>
                    </Tr>
                ))}
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
                            <p className="pf-v6-u-font-size-sm pf-v6-u-text-color-status-danger">
                                Invalid label key
                            </p>
                        )}
                        {validatedKey === ValidatedOptions.warning && (
                            <p className="pf-v6-u-font-size-sm pf-v6-u-text-color-status-warning">
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
                            <p className="pf-v6-u-font-size-sm pf-v6-u-text-color-status-danger">
                                {valueInput.length === 0
                                    ? 'Label value is required'
                                    : 'Invalid label value'}
                            </p>
                        )}
                    </Td>
                    <Td dataLabel="Action">
                        <Tooltip content={isReplace ? 'Replace label' : 'Add label'}>
                            <Button
                                icon={
                                    <Icon>
                                        <PlusCircleIcon
                                            color={
                                                isReplace
                                                    ? 'var(--pf-t--global--icon--color--status--warning--default)'
                                                    : 'var(--pf-t--global--icon--color--status--success--default)'
                                            }
                                        />
                                    </Icon>
                                }
                                aria-label={isReplace ? 'Replace label' : 'Add label'}
                                variant="plain"
                                style={{ padding: 0 }}
                                isDisabled={!isValid}
                                onClick={() => onAddLabel()}
                            />
                        </Tooltip>
                    </Td>
                </Tr>
            </Tbody>
        </Table>
    );
}

export default ClusterLabelsTable;
