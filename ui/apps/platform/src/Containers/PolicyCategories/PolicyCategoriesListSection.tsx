import React, { useEffect, useState } from 'react';
import { PageSection, Title, Flex, TextInput } from '@patternfly/react-core';

import PolicyCategoriesList from './PolicyCategoriesList';
import PolicyCategoriesFilterSelect from './PolicyCategoriesFilterSelect';

type PolicyCategoriesListSectionProps = {
    // TODO: change once proto goes in
    // policyCategories: {
    //     id: string;
    //     name: string;
    //     isDefault: boolean;
    // }[];
    policyCategories: string[];
};

function PolicyCategoriesListSection({ policyCategories }: PolicyCategoriesListSectionProps) {
    // const [selectedCategory, setSelectedCategory] = useState('');
    // TODO: to remove once proto is in
    const allPolicyCategories = policyCategories.map((category) => ({
        id: category,
        name: category,
        isDefault: true,
    }));
    const customPolicyCategories = allPolicyCategories.filter(({ isDefault }) => !isDefault);
    const defaultPolicyCategories = allPolicyCategories.filter(({ isDefault }) => isDefault);
    //
    const [currentPolicyCategories, setCurrentPolicyCategories] = useState(allPolicyCategories);
    const [filterTerm, setFilterTerm] = useState('');
    const [filteredCategories, setFilteredCategories] = useState(currentPolicyCategories);

    useEffect(() => {
        setFilteredCategories(
            currentPolicyCategories.filter(({ name }) => name.includes(filterTerm))
        );
    }, [filterTerm, currentPolicyCategories]);

    return (
        <PageSection isFilled id="policy-categories-list-section">
            <PageSection isFilled variant="light">
                <Flex direction={{ default: 'column' }}>
                    <Flex
                        justifyContent={{ default: 'justifyContentSpaceBetween' }}
                        fullWidth={{ default: 'fullWidth' }}
                    >
                        <Title headingLevel="h3">Categories</Title>
                        <Title headingLevel="h3">{filteredCategories.length} results found</Title>
                    </Flex>
                    <Flex
                        justifyContent={{ default: 'justifyContentSpaceBetween' }}
                        fullWidth={{ default: 'fullWidth' }}
                        flexWrap={{ default: 'nowrap' }}
                    >
                        <TextInput
                            onChange={setFilterTerm}
                            type="text"
                            value={filterTerm}
                            placeholder="Filter by category name..."
                        />
                        <PolicyCategoriesFilterSelect
                            setCurrentPolicyCategories={setCurrentPolicyCategories}
                            defaultPolicyCategories={defaultPolicyCategories}
                            customPolicyCategories={customPolicyCategories}
                        />
                    </Flex>
                    {filteredCategories.length > 0 && (
                        <PolicyCategoriesList policyCategories={filteredCategories} />
                    )}
                    {filteredCategories.length === 0 && <div>No policy categories found.</div>}
                </Flex>
            </PageSection>
        </PageSection>
    );
}

export default PolicyCategoriesListSection;
