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
    Button,
} from '@patternfly/react-core';
import { useField } from 'formik';

import { getPolicyCategories } from 'services/PolicyCategoriesService';
import { PolicyCategory } from 'types/policy.proto';
import { TimesIcon } from '@patternfly/react-icons';

function PolicyCategoriesSelectField(): ReactElement {
    const [policyCategories, setPolicyCategories] = useState<PolicyCategory[]>([]);
    const [isOpen, setIsOpen] = useState(false);
    const [inputValue, setInputValue] = useState('');
    const [field, , helpers] = useField('categories');
    const [preventClose, setPreventClose] = useState(false);

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
            setPreventClose(true);
            // Reset the flag after a short delay
            setTimeout(() => setPreventClose(false), 100);
        }
        setInputValue('');
        // Don't call setIsOpen(false) to keep dropdown open
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
                    onChange={(_event: React.FormEvent<HTMLInputElement>, value: string) =>
                        onInputChange(value)
                    }
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
                                onClick={(event: React.MouseEvent) => {
                                    event.stopPropagation();
                                    onRemoveChip(category);
                                }}
                                aria-label={`Remove ${category} category`}
                            >
                                {category}
                            </Chip>
                        ))}
                    </ChipGroup>
                </TextInputGroupMain>
                <TextInputGroupUtilities>
                    {selectedCategories.length > 0 && (
                        <Button
                            variant="plain"
                            onClick={onClearAll}
                            aria-label="Clear all selected categories"
                            type="button"
                        >
                            <TimesIcon aria-hidden />
                        </Button>
                    )}
                </TextInputGroupUtilities>
            </TextInputGroup>
        </MenuToggle>
    );

    // @TODO: Look into creating a custom component for this, as it's a bit complex and could be reused in other places
    return (
        <FormGroup fieldId="policy-categories" label="Categories" isRequired>
            <Select
                id="policy-categories-select"
                isOpen={isOpen}
                selected={selectedCategories}
                onSelect={onSelect}
                onOpenChange={(nextOpen: boolean) => {
                    // If we just selected an item, keep the dropdown open
                    if (!nextOpen && preventClose) {
                        return;
                    }
                    setIsOpen(nextOpen);
                }}
                toggle={toggle}
                aria-label="Policy categories multi-select"
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
