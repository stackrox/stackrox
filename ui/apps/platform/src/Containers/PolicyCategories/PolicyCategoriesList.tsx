import React from 'react';
import { SimpleList, SimpleListItem } from '@patternfly/react-core';

import { PolicyCategory } from 'types/policy.proto';

type PolicyCategoriesListProps = {
    policyCategories: PolicyCategory[];
    setSelectedCategory: (selectedCategory: PolicyCategory) => void;
};

function PolicyCategoriesList({
    policyCategories,
    setSelectedCategory,
}: PolicyCategoriesListProps) {
    return (
        <SimpleList onSelect={() => {}}>
            {policyCategories.map((category) => {
                const { id, name, isDefault } = category;
                return (
                    <SimpleListItem
                        key={id}
                        onClick={() => {
                            setSelectedCategory(category);
                        }}
                        componentProps={{ disabled: isDefault }}
                    >
                        {name}
                    </SimpleListItem>
                );
            })}
        </SimpleList>
    );
}

export default PolicyCategoriesList;
