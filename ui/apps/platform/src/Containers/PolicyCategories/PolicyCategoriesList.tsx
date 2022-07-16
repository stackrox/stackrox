import React from 'react';
import { SimpleList, SimpleListItem } from '@patternfly/react-core';

type PolicyCategoriesListProps = {
    policyCategories: {
        id: string;
        name: string;
        isDefault: boolean;
    }[];
    setSelectedCategory: (selectedCategory: string) => void;
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
                        isActive={false}
                        // componentProps={{ disabled: isDefault }}
                    >
                        {name}
                    </SimpleListItem>
                );
            })}
        </SimpleList>
    );
}

export default PolicyCategoriesList;
