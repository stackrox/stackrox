import { useEffect, useRef, useState } from 'react';
import type { ReactElement } from 'react';
import { Button, Icon, TextInput, Tooltip, ValidatedOptions } from '@patternfly/react-core';
import { ArrowCircleDownIcon, TimesCircleIcon } from '@patternfly/react-icons';
import { Td, Tr } from '@patternfly/react-table';

import { getIsValidLabelKey } from 'utils/labels';

/*
 * Render a temporary row to enter the key for a new requirement.
 */
export type RequirementRowAddKeyProps = {
    handleRequirementKeyOK: (key: string) => void;
    handleRequirementKeyCancel: () => void;
};

function RequirementRowAddKey({
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

    const isInvalidKey = !getIsValidLabelKey(keyInput);
    const isDisabledOK = isInvalidKey;

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
                <div className="pf-v6-u-display-flex">
                    <span className="pf-v6-u-flex-basis-0 pf-v6-u-flex-grow-1 pf-v6-u-flex-shrink-1 pf-v6-u-text-break-word">
                        <TextInput
                            aria-label="Type a key"
                            value={keyInput}
                            validated={validatedKey}
                            onChange={(_event, keyChange: string) => onKeyChange(keyChange)}
                            onKeyDown={onKeyDown}
                            ref={refKeyInput}
                            className="pf-m-small"
                        />
                    </span>
                    <span className="pf-v6-u-flex-shrink-0">
                        <Tooltip content="Requirement key OK (press tab or enter)">
                            <Button
                                icon={
                                    <Icon>
                                        <ArrowCircleDownIcon
                                            color="var(--pf-t--temp--dev--tbd)" /* CODEMODS: original v5 color was --pf-v5-global--primary-color--100 */
                                            style={{ transform: 'rotate(-90deg)' }}
                                        />
                                    </Icon>
                                }
                                aria-label="Requirement key OK (press tab or enter)"
                                variant="plain"
                                className="pf-m-smallest pf-v6-u-ml-sm"
                                isDisabled={isDisabledOK}
                                onClick={onClickRequirementKeyOK}
                            />
                        </Tooltip>
                    </span>
                </div>
                {keyInput.length !== 0 && isInvalidKey && (
                    <p className="pf-v6-u-font-size-sm pf-v6-u-danger-color-100">Invalid key</p>
                )}
            </Td>
            <Td dataLabel="Operator" />
            <Td dataLabel="Values" />
            <Td dataLabel="Action" className="pf-v6-u-text-align-right">
                <Tooltip key="Cancel" content="Cancel">
                    <Button
                        icon={
                            <Icon>
                                <TimesCircleIcon color="var(--pf-v5-global--color--100)" />
                            </Icon>
                        }
                        aria-label="Cancel"
                        variant="plain"
                        className="pf-m-smallest"
                        onClick={handleRequirementKeyCancel}
                    />
                </Tooltip>
            </Td>
        </Tr>
    );
}

export default RequirementRowAddKey;
