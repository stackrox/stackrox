import React from 'react';
import { SimpleList, SimpleListItem } from '@patternfly/react-core';

type PolicyCategoriesListProps = {
    // policyCategories: {
    //     id: string;
    //     name: string;
    //     isDefault: boolean;
    // }[];
    policyCategories: string[];
};

function PolicyCategoriesList({ policyCategories }: PolicyCategoriesListProps) {
    return (
        <SimpleList onSelect={() => {}}>
            {policyCategories.map((name, idx) => (
                <SimpleListItem
                    // eslint-disable-next-line react/no-array-index-key
                    key={idx}
                    onClick={() => {}}
                    isActive={false}
                    componentProps={{ isDisabled: false }}
                >
                    {name}
                </SimpleListItem>
            ))}
        </SimpleList>
    );
}

export default PolicyCategoriesList;
