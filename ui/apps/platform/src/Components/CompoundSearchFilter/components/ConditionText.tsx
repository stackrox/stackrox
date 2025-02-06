import React, { useState } from 'react';
import {
    Button,
    MenuToggle,
    MenuToggleElement,
    Select,
    SelectList,
    SelectOption,
    TextInput,
} from '@patternfly/react-core';
import { ArrowRightIcon } from '@patternfly/react-icons';

import { NonEmptyArray } from 'utils/type.utils';

import './ConditionText.css';

// Potentially reusable for condition-number and date-picker components.
export type ConditionEntry = [conditionKey: string, conditionText: string];
export type ConditionEntries = NonEmptyArray<ConditionEntry>; // first item has default conditionKey
export type ConditionInputProps = {
    conditionEntries: ConditionEntries;
};

type TextInputProps = {
    convertFromExternalToInternalText: (externalText: string) => string; // from TextInput element to query string
    convertFromInternalToExternalText: (internalText: string) => string; // from query string to FilterChip element
    externalTextDefault: string; // initial value for TextInput element
    validateExternalText: (externalText: string) => boolean; // for search Button element
    validateInternalText: (internalText: string) => boolean; // for FilterChip element
};

export type ConditionTextInputProps = {
    conditionProps: ConditionInputProps;
    textProps: TextInputProps;
};

function joinConditionText(conditionKey: string, text: string) {
    return `${conditionKey}${text}`;
}

// For filter chip: split, convert, and then join.
export function convertFromInternalToExternalConditionText(
    inputProps: ConditionTextInputProps,
    internalConditionText: string
) {
    const {
        conditionProps: { conditionEntries },
        textProps: { convertFromInternalToExternalText, validateInternalText },
    } = inputProps;

    // Find the longest prefix, because > is shorter than >= for example.
    let length = 0;
    let conditionFound;
    conditionEntries.forEach((condition) => {
        const conditionKey = condition[0];
        if (internalConditionText.startsWith(conditionKey) && conditionKey.length > length) {
            length = conditionKey.length;
            conditionFound = condition;
        }
    });

    if (conditionFound) {
        const conditionKey = conditionFound[0];
        const internalText = internalConditionText.slice(conditionKey.length);
        if (validateInternalText(internalText)) {
            const externalText = convertFromInternalToExternalText(internalText);
            return joinConditionText(conditionKey, externalText);
        }
    }

    // Enclose text in typographical quotes in case it is clearer.
    return `“${internalConditionText}” is not valid`; // query string in page address
}

export type ConditionTextProps = {
    inputProps: ConditionTextInputProps;
    onSearch: (internalConditionText: string) => void;
};

function ConditionText({ inputProps, onSearch }: ConditionTextProps) {
    const {
        conditionProps: { conditionEntries },
        textProps: { convertFromExternalToInternalText, externalTextDefault, validateExternalText },
    } = inputProps;

    const [conditionKey, setConditionKey] = useState(conditionEntries[0][0]);
    const [externalText, setExternalText] = useState(externalTextDefault);

    // Adapt SimpleSelect because its MenuToggle renders conditionKey instead of conditionText.
    const [isOpen, setIsOpen] = React.useState(false);

    const onToggleClick = () => {
        setIsOpen(!isOpen);
    };

    const onSelect = (
        _event: React.MouseEvent<Element, MouseEvent> | undefined,
        value: string | number | undefined
    ) => {
        setConditionKey(value as string);
        setIsOpen(false);
    };

    // Interesting dilemma that map might be more convenient here,
    // but less convenient for initial state and se;ect list.
    const conditionSelected =
        conditionEntries.find((condition) => condition[0] === conditionKey) ?? conditionEntries[0];

    const toggle = (toggleRef: React.Ref<MenuToggleElement>) => (
        <MenuToggle
            className="pf-v5-u-flex-shrink-0"
            aria-label="Condition selector toggle"
            ref={toggleRef}
            onClick={onToggleClick}
            isExpanded={isOpen}
        >
            {conditionSelected[1]}
        </MenuToggle>
    );

    return (
        <>
            <Select
                isOpen={isOpen}
                selected={conditionKey}
                onSelect={onSelect}
                onOpenChange={(isOpen) => setIsOpen(isOpen)}
                toggle={toggle}
                shouldFocusToggleOnSelect
            >
                <SelectList aria-label="Condition selector menu">
                    {conditionEntries.map(([conditionKey, conditionText]) => (
                        <SelectOption key={conditionKey} value={conditionKey}>
                            {conditionText}
                        </SelectOption>
                    ))}
                </SelectList>
            </Select>
            <TextInput
                aria-label="Condition value input"
                className="ConditionTextInput"
                onChange={(event: React.FormEvent<HTMLInputElement>) => {
                    const { value: changedText } = event.target as HTMLInputElement;
                    setExternalText(changedText);
                }}
                validated={validateExternalText(externalText) ? 'success' : 'error'}
                value={externalText}
            />
            <Button
                aria-label="Apply condition and number input to search"
                isDisabled={!validateExternalText(externalText)}
                onClick={() => {
                    const internalText = convertFromExternalToInternalText(externalText);
                    onSearch(joinConditionText(conditionKey, internalText));
                }}
                variant="control"
            >
                <ArrowRightIcon />
            </Button>
        </>
    );
}

export default ConditionText;
