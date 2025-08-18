import React, { useState, useEffect, useMemo, ReactElement } from 'react';
import {
    FormGroup,
    FormHelperText,
    HelperText,
    HelperTextItem,
    Select,
    SelectOption,
    SelectList,
    MenuToggle,
    MenuToggleElement,
    TextInputGroup,
    TextInputGroupMain,
    TextInputGroupUtilities,
    ChipGroup,
    Chip,
} from '@patternfly/react-core';
import { useField } from 'formik';

import { getPolicyCategories } from 'services/PolicyCategoriesService';
import { PolicyCategory } from 'types/policy.proto';

function PolicyCategoriesSelectField(): ReactElement {
    const [policyCategories, setPolicyCategories] = useState<PolicyCategory[]>([]);
    const [isOpen, setIsOpen] = useState(false);
    const [inputValue, setInputValue] = useState('');
    const [shouldStayOpen, setShouldStayOpen] = useState(false);
    const [field, , helpers] = useField('categories');

    const selectedCategories: string[] = useMemo(
        () => (field.value as string[]) || [],
        [field.value]
    );

    const onToggle = () => {
        setIsOpen(!isOpen);
    };

    const onSelect = (
        _event: React.MouseEvent<Element, MouseEvent> | undefined,
        value: string | number | undefined
    ) => {
        if (typeof value === 'string' && !selectedCategories.includes(value)) {
            helpers.setValue([...selectedCategories, value]);
            setShouldStayOpen(true);
        }
        setInputValue('');
    };

    const onRemoveChip = (categoryToRemove: string) => {
        helpers.setValue(selectedCategories.filter((category) => category !== categoryToRemove));
    };

    const onClearAll = () => {
        helpers.setValue([]);
        setInputValue('');
    };

    const onInputChange = (value: string) => {
        setInputValue(value);
    };

    useEffect(() => {
        getPolicyCategories()
            .then((data) => {
                setPolicyCategories(data);
            })
            .catch(() => {});
    }, []);

    // Filter available options based on input and already selected items
    const filteredOptions = useMemo(
        () =>
            policyCategories
                .filter(
                    ({ name }) =>
                        name.toLowerCase().includes(inputValue.toLowerCase()) &&
                        !selectedCategories.includes(name)
                )
                .map(({ id, name }) => (
                    <SelectOption key={id} value={name}>
                        {name}
                    </SelectOption>
                )),
        [policyCategories, inputValue, selectedCategories]
    );

    const toggle = (toggleRef: React.Ref<MenuToggleElement>) => (
        <MenuToggle
            variant="typeahead"
            onClick={onToggle}
            innerRef={toggleRef}
            isExpanded={isOpen}
            className="pf-v5-u-w-100"
        >
            <TextInputGroup isPlain>
                <TextInputGroupMain
                    value={inputValue}
                    onClick={onToggle}
                    onChange={(_event, value) => onInputChange(value)}
                    autoComplete="off"
                    placeholder="Select categories"
                    role="combobox"
                    isExpanded={isOpen}
                    aria-controls="select-typeahead-listbox"
                >
                    <ChipGroup>
                        {selectedCategories.map((category) => (
                            <Chip
                                key={category}
                                onClick={(event) => {
                                    event.stopPropagation();
                                    onRemoveChip(category);
                                }}
                            >
                                {category}
                            </Chip>
                        ))}
                    </ChipGroup>
                </TextInputGroupMain>
                <TextInputGroupUtilities>
                    {selectedCategories.length > 0 && (
                        <button
                            className="pf-v5-c-button pf-m-plain"
                            type="button"
                            onClick={onClearAll}
                            aria-label="Clear all"
                        >
                            ×
                        </button>
                    )}
                </TextInputGroupUtilities>
            </TextInputGroup>
        </MenuToggle>
    );

    return (
        <FormGroup fieldId="policy-categories" label="Categories" isRequired>
            <Select
                id="policy-categories-select"
                isOpen={isOpen}
                selected={selectedCategories}
                onSelect={onSelect}
                onOpenChange={(nextOpen: boolean) => {
                    if (shouldStayOpen && !nextOpen) {
                        // If we want to stay open but PatternFly wants to close, keep it open
                        setShouldStayOpen(false);
                        return;
                    }
                    setIsOpen(nextOpen);
                }}
                toggle={toggle}
            >
                <SelectList
                    id="select-typeahead-listbox"
                    style={{ maxHeight: '300px', overflowY: 'auto' }}
                >
                    {filteredOptions.length > 0 ? (
                        filteredOptions
                    ) : (
                        <SelectOption isDisabled key="no-results">
                            No categories found
                        </SelectOption>
                    )}
                </SelectList>
            </Select>
            <FormHelperText>
                <HelperText>
                    <HelperTextItem>
                        Select policy categories you want to apply to this policy
                    </HelperTextItem>
                </HelperText>
            </FormHelperText>
        </FormGroup>
    );
}

export default PolicyCategoriesSelectField;
