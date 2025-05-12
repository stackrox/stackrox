import React, { useState, useEffect, ReactElement } from 'react';
import { FormGroup, FormHelperText, HelperText, HelperTextItem } from '@patternfly/react-core';
import { Select, SelectOption } from '@patternfly/react-core/deprecated';
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
        <FormGroup fieldId="policy-categories" label="Categories" isRequired>
            <Select
                variant="typeaheadmulti"
                name={field.name}
                value={field.value}
                isOpen={isCategoriesOpen}
                selections={field.value}
                onSelect={onSelectHandler(field.value)}
                onToggle={(_event, isOpen) => onCategoriesToggle(isOpen)}
                onClear={clearSelection}
                isCreatable={false}
                maxHeight="300px"
            >
                {policyCategories.map(({ id, name }) => (
                    <SelectOption key={id} value={name} />
                ))}
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
