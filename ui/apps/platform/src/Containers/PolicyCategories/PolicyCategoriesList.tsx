import React from 'react';
import { SimpleList, SimpleListItem } from '@patternfly/react-core';

type PolicyCategoriesListProps = {
    policyCategories: {
        id: string;
        name: string;
        isDefault: boolean;
    }[];
};

function PolicyCategoriesList({ policyCategories }: PolicyCategoriesListProps) {
    return (
        <SimpleList onSelect={() => {}}>
            {policyCategories.map(({ id, name, isDefault }) => (
                <SimpleListItem
                    key={id}
                    onClick={() => {}}
                    isActive={false}
                    componentProps={{ disabled: isDefault }}
                >
                    {name}
                </SimpleListItem>
            ))}
        </SimpleList>
    );
}

export default PolicyCategoriesList;
