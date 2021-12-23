import React from 'react';
import { useField } from 'formik';
import {
    TextInput,
    ToggleGroup,
    ToggleGroupItem,
    FormGroup,
    Select,
    SelectOption,
    SelectVariant,
} from '@patternfly/react-core';

import { Descriptor, SubComponent } from 'Containers/Policies/Wizard/Form/descriptors';

type FieldProps = {
    descriptor: Descriptor;
    readOnly: boolean;
    name: string;
};

function Field({ descriptor, readOnly, name }: FieldProps) {
    const [field, , helper] = useField(name);
    const [isSelectOpen, setIsSelectOpen] = React.useState(false);
    const { value } = field;
    const { setValue } = helper;

    // TODO: Add group/nested fields
    // this is to accomodate for recursive Fields (when type is 'group')
    // const path = descriptor.subpath ? name : `${name}.value`;

    function handleChangeValue(val) {
        setValue({ value: val });
    }

    function handleChangeSelectedValue(selectedVal) {
        return () => handleChangeValue(selectedVal);
    }

    function handleChangeSelect(e, val) {
        setIsSelectOpen(false);
        handleChangeValue(val);
    }

    function handleChangeSelectMultiple(e, selection) {
        if (value.value?.includes(selection)) {
            handleChangeValue(value.value.filter((item) => item !== selection));
        } else {
            handleChangeValue([...value.value, selection]);
        }
        setIsSelectOpen(false);
    }

    function handleOnToggleSelect() {
        setIsSelectOpen(!isSelectOpen);
    }

    switch (descriptor.type) {
        case 'text':
            return (
                <TextInput
                    value={value.value}
                    type="text"
                    isDisabled={readOnly}
                    onChange={handleChangeValue}
                />
            );
        case 'radioGroup':
            return (
                <ToggleGroup>
                    {descriptor?.radioButtons?.map(({ text, value: radioValue }) => {
                        return (
                            <React.Fragment key={text}>
                                <ToggleGroupItem
                                    text={text}
                                    buttonId={text}
                                    isDisabled={readOnly}
                                    isSelected={value.value === radioValue}
                                    onChange={handleChangeSelectedValue(radioValue)}
                                />
                            </React.Fragment>
                        );
                    })}
                </ToggleGroup>
            );
        case 'number':
            return (
                <TextInput
                    value={value.value}
                    type="number"
                    isDisabled={readOnly}
                    onChange={handleChangeValue}
                    placeholder={descriptor.placeholder}
                />
            );
        case 'select':
            return (
                <FormGroup
                    label={descriptor.label}
                    fieldId={descriptor.name}
                    className="pf-u-flex-1"
                >
                    <Select
                        onToggle={handleOnToggleSelect}
                        onSelect={handleChangeSelect}
                        isOpen={isSelectOpen}
                        isDisabled={readOnly}
                        selections={value.value}
                        placeholderText={descriptor.placeholder}
                    >
                        {descriptor?.options?.map((option) => {
                            return <SelectOption value={option.value}>{option.label}</SelectOption>;
                        })}
                    </Select>
                </FormGroup>
            );
        case 'multiselect':
            return (
                <FormGroup
                    label={descriptor.label}
                    fieldId={descriptor.name}
                    className="pf-u-flex-1"
                >
                    <Select
                        onToggle={handleOnToggleSelect}
                        onSelect={handleChangeSelectMultiple}
                        isOpen={isSelectOpen}
                        isDisabled={readOnly}
                        selections={value.value}
                        onClear={handleChangeSelectedValue([])}
                        placeholderText={descriptor.placeholder}
                        variant={SelectVariant.typeaheadMulti}
                    >
                        {descriptor?.options?.map((option) => {
                            return <SelectOption value={option.value}>{option.label}</SelectOption>;
                        })}
                    </Select>
                </FormGroup>
            );
        default:
            throw new Error(`Unknown field type: ${descriptor.type}`);
    }
}

export default Field;
