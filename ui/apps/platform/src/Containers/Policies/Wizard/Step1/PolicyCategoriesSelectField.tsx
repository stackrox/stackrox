import React, { useState, useEffect, ReactElement } from 'react';
import { FormGroup, Select, SelectOption, SelectVariant } from '@patternfly/react-core';
import { useField } from 'formik';

import useFeatureFlags from 'hooks/useFeatureFlags';
import { getPolicyCategories as getPolicyCategoriesNonPostgres } from 'services/PoliciesService';
import { getPolicyCategories } from 'services/PolicyCategoriesService';
import { PolicyCategory } from 'types/policy.proto';

function PolicyCategoriesSelectField(): ReactElement {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isPolicyCategoriesEnabled = isFeatureFlagEnabled('ROX_POSTGRES_DATASTORE');

    const [policyCategories, setPolicyCategories] = useState<PolicyCategory[]>([]);
    // manage state for Categories select below
    const [isCategoriesOpen, setIsCategoriesOpen] = useState(false);

    const [field, , helpers] = useField('categories');

    function onCategoriesToggle(isOpen) {
        setIsCategoriesOpen(isOpen);
    }

    function onCreateCategory(newCategory) {
        setPolicyCategories([...policyCategories, newCategory]);
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
        if (isPolicyCategoriesEnabled) {
            getPolicyCategories()
                .then((data) => {
                    setPolicyCategories(data);
                })
                .catch(() => {});
        } else {
            getPolicyCategoriesNonPostgres()
                .then((data) => {
                    setPolicyCategories(data.map((name) => ({ id: name, name, isDefault: false })));
                })
                .catch(() => {});
        }

        return () => {
            setPolicyCategories([]);
        };
    }, [isPolicyCategoriesEnabled]);

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
                onCreateOption={
                    isPolicyCategoriesEnabled ? () => undefined as void : onCreateCategory
                }
                onClear={clearSelection}
                isCreatable={!isPolicyCategoriesEnabled}
            >
                {policyCategories.map(({ id, name }) => (
                    <SelectOption key={id} value={name} />
                ))}
            </Select>
        </FormGroup>
    );
}

export default PolicyCategoriesSelectField;
