import { useState } from 'react';
import type { FormEvent, MouseEvent as ReactMouseEvent, Ref } from 'react';
import {
    Button,
    MenuToggle,
    Select,
    SelectList,
    SelectOption,
    TextInput,
} from '@patternfly/react-core';
import type { MenuToggleElement } from '@patternfly/react-core';
import { ArrowRightIcon } from '@patternfly/react-icons';

import type { NonEmptyArray } from 'utils/type.utils';

import type { ConditionTextFilterAttribute, OnSearchCallback } from '../types';

import './SearchFilterConditionText.css';

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

export type SearchFilterConditionTextProps = {
    attribute: ConditionTextFilterAttribute;
    onSearch: OnSearchCallback;
    // does not depend on searchFilter
};

function SearchFilterConditionText({ attribute, onSearch }: SearchFilterConditionTextProps) {
    const { inputProps, searchTerm: category } = attribute;
    const {
        conditionProps: { conditionEntries },
        textProps: { convertFromExternalToInternalText, externalTextDefault, validateExternalText },
    } = inputProps;

    const [conditionKey, setConditionKey] = useState(conditionEntries[0][0]);
    const [externalText, setExternalText] = useState(externalTextDefault);

    // Adapt SimpleSelect because its MenuToggle renders conditionKey instead of conditionText.
    const [isOpen, setIsOpen] = useState(false);

    const onToggleClick = () => {
        setIsOpen((prev) => !prev);
    };

    const onSelect = (
        _event: ReactMouseEvent<Element, MouseEvent> | undefined,
        value: string | number | undefined
    ) => {
        setConditionKey(value as string);
        setIsOpen(false);
    };

    // Interesting dilemma that map might be more convenient here,
    // but less convenient for initial state and se;ect list.
    const conditionSelected =
        conditionEntries.find((condition) => condition[0] === conditionKey) ?? conditionEntries[0];

    const toggle = (toggleRef: Ref<MenuToggleElement>) => (
        <MenuToggle
            className="pf-v6-u-flex-shrink-0"
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
                onChange={(event: FormEvent<HTMLInputElement>) => {
                    const { value: changedText } = event.target as HTMLInputElement;
                    setExternalText(changedText);
                }}
                validated={validateExternalText(externalText) ? 'success' : 'error'}
                value={externalText}
            />
            <Button
                icon={<ArrowRightIcon />}
                aria-label="Apply condition and number input to search"
                isDisabled={!validateExternalText(externalText)}
                onClick={() => {
                    const internalText = convertFromExternalToInternalText(externalText);
                    onSearch([
                        {
                            action: 'APPEND',
                            category,
                            value: joinConditionText(conditionKey, internalText),
                        },
                    ]);
                }}
                variant="control"
            ></Button>
        </>
    );
}

export default SearchFilterConditionText;
