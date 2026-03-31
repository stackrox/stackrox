import { SimpleList, SimpleListItem } from '@patternfly/react-core';

import type { PolicyCategory } from 'types/policy.proto';

type PolicyCategoriesListProps = {
    policyCategories: PolicyCategory[];
    setSelectedCategory: (selectedCategory: PolicyCategory) => void;
};

// TODO Evaluate whether or not we should switch from SimpleList here - the disabled style is almost indistinguishable from the default style.
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
