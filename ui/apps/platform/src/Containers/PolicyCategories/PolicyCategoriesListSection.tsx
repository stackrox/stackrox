import React, { useEffect, useState } from 'react';
import { PageSection, Divider, Title, Flex, FlexItem, TextInput } from '@patternfly/react-core';

import type { PolicyCategory } from 'types/policy.proto';
import PolicyCategoriesList from './PolicyCategoriesList';
import PolicyCategoriesFilterSelect from './PolicyCategoriesFilterSelect';
import type { CategoryFilter } from './PolicyCategoriesFilterSelect';
import PolicyCategorySidePanel from './PolicyCategorySidePanel';
import DeletePolicyCategoryModal from './DeletePolicyCategoryModal';

type PolicyCategoriesListSectionProps = {
    policyCategories: PolicyCategory[];
    addToast: (message) => void;
    selectedCategory: PolicyCategory | undefined;
    setSelectedCategory: (category) => void;
    refreshPolicyCategories: () => void;
};

function PolicyCategoriesListSection({
    policyCategories,
    addToast,
    selectedCategory,
    setSelectedCategory,
    refreshPolicyCategories,
}: PolicyCategoriesListSectionProps) {
    const customPolicyCategories = policyCategories.filter(({ isDefault }) => !isDefault);
    const defaultPolicyCategories = policyCategories.filter(({ isDefault }) => isDefault);
    const [selectedFilter, setSelectedFilter] = useState<CategoryFilter>('All categories');
    const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false);

    let currentPolicyCategories = policyCategories;
    if (selectedFilter === 'Default categories') {
        currentPolicyCategories = defaultPolicyCategories;
    } else if (selectedFilter === 'Custom categories') {
        currentPolicyCategories = customPolicyCategories;
    }
    const [filterTerm, setFilterTerm] = useState('');
    const [filteredCategories, setFilteredCategories] = useState(currentPolicyCategories);

    useEffect(() => {
        setFilteredCategories(
            currentPolicyCategories.filter(({ name }) =>
                name.toLowerCase().includes(filterTerm.toLowerCase())
            )
        );
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [filterTerm, selectedFilter, policyCategories]);

    return (
        <>
            <PageSection id="policy-categories-list-section">
                <Flex
                    spaceItems={{ default: 'spaceItemsNone' }}
                    alignItems={{ default: 'alignItemsStretch' }}
                    className="pf-v5-u-h-100"
                >
                    <FlexItem flex={{ default: 'flex_1' }}>
                        <PageSection isFilled variant="light" className="pf-v5-u-h-100">
                            <Flex direction={{ default: 'column' }}>
                                <Title headingLevel="h2">
                                    <Flex
                                        justifyContent={{ default: 'justifyContentSpaceBetween' }}
                                        fullWidth={{ default: 'fullWidth' }}
                                    >
                                        <span>Categories</span>
                                        <span>{filteredCategories.length} results found</span>
                                    </Flex>
                                </Title>
                                <Flex
                                    spaceItems={{ default: 'spaceItemsSm' }}
                                    fullWidth={{ default: 'fullWidth' }}
                                    flexWrap={{ default: 'nowrap' }}
                                >
                                    <FlexItem flex={{ default: 'flex_1' }}>
                                        <TextInput
                                            onChange={(_event, val) => setFilterTerm(val)}
                                            type="text"
                                            value={filterTerm}
                                            placeholder="Filter by category name..."
                                            id="policy-categories-filter-input"
                                            isDisabled={!!selectedCategory}
                                        />
                                    </FlexItem>
                                    <FlexItem>
                                        <PolicyCategoriesFilterSelect
                                            selectedFilter={selectedFilter}
                                            setSelectedFilter={setSelectedFilter}
                                            isDisabled={!!selectedCategory}
                                        />
                                    </FlexItem>
                                </Flex>
                                {filteredCategories.length > 0 && (
                                    <PolicyCategoriesList
                                        policyCategories={filteredCategories}
                                        setSelectedCategory={setSelectedCategory}
                                    />
                                )}
                                {filteredCategories.length === 0 && (
                                    <div>No policy categories found.</div>
                                )}
                            </Flex>
                        </PageSection>
                    </FlexItem>
                    {selectedCategory && (
                        <>
                            <Divider component="div" orientation={{ default: 'vertical' }} />
                            <FlexItem flex={{ default: 'flex_1' }}>
                                <PolicyCategorySidePanel
                                    selectedCategory={selectedCategory}
                                    setSelectedCategory={setSelectedCategory}
                                    addToast={addToast}
                                    openDeleteModal={() => setIsDeleteModalOpen(true)}
                                    refreshPolicyCategories={refreshPolicyCategories}
                                />
                            </FlexItem>
                        </>
                    )}
                </Flex>
            </PageSection>
            <DeletePolicyCategoryModal
                isOpen={isDeleteModalOpen}
                onClose={() => setIsDeleteModalOpen(false)}
                refreshPolicyCategories={refreshPolicyCategories}
                addToast={addToast}
                selectedCategory={selectedCategory}
                setSelectedCategory={setSelectedCategory}
            />
        </>
    );
}

export default PolicyCategoriesListSection;
