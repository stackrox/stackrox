import React, { useState, useEffect, ReactElement } from 'react';
import { FormGroup, Select, SelectOption, SelectVariant } from '@patternfly/react-core';
import { useField } from 'formik';

import { getPolicyCategories } from 'services/PolicyCategoriesService';
import { PolicyCategory } from 'types/policy.proto';

function PolicyCategoriesSelectField(): ReactElement {
    const [policyCategories, setPolicyCategories] = useState<PolicyCategory[]>([]);
    // manage state for Categories select below
    const [isCategoriesOpen, setIsCategoriesOpen] = useState(false);

    const [field, , helpers] = useField('categories');

    function onCategoriesToggle(isOpen) {
        setIsCategoriesOpen(isOpen);
    }

    function onSelectHandler(selectedCategories) {
        return (event, selection) => {
            const newSelectedCategories = selectedCategories.includes(selection)
                ? selectedCategories.filter((item) => item !== selection)
                : [...selectedCategories, selection];
            helpers.setValue(newSelectedCategories);
        };
    }

    function clearSelection() {
        setIsCategoriesOpen(false);
        helpers.setValue([]);
    }

    useEffect(() => {
        getPolicyCategories()
            .then((data) => {
                setPolicyCategories(data);
            })
            .catch(() => {});
    }, []);

    return (
        <FormGroup
            helperText="Select policy categories you want to apply to this policy"
            fieldId="policy-categories"
            label="Categories"
            isRequired
        >
            <Select
                variant={SelectVariant.typeaheadMulti}
                name={field.name}
                value={field.value}
                isOpen={isCategoriesOpen}
                selections={field.value}
                onSelect={onSelectHandler(field.value)}
                onToggle={onCategoriesToggle}
                onClear={clearSelection}
                isCreatable={false}
            >
                {policyCategories.map(({ id, name }) => (
                    <SelectOption key={id} value={name} />
                ))}
            </Select>
        </FormGroup>
    );
}

export default PolicyCategoriesSelectField;
