import React, { useEffect, useState } from 'react';
import { PageSection, Divider, Title, Flex, FlexItem, TextInput } from '@patternfly/react-core';

import { PolicyCategory } from 'types/policy.proto';
import PolicyCategoriesList from './PolicyCategoriesList';
import PolicyCategoriesFilterSelect, { CategoryFilter } from './PolicyCategoriesFilterSelect';
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
    const [selectedFilters, setSelectedFilters] = useState<CategoryFilter[]>([
        'Default categories',
        'Custom categories',
    ]);
    const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false);

    let currentPolicyCategories = policyCategories;
    if (selectedFilters.length === 1) {
        if (selectedFilters[0] === 'Default categories') {
            currentPolicyCategories = defaultPolicyCategories;
        }
        if (selectedFilters[0] === 'Custom categories') {
            currentPolicyCategories = customPolicyCategories;
        }
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
    }, [filterTerm, selectedFilters, policyCategories]);

    return (
        <>
            <PageSection id="policy-categories-list-section">
                <Flex
                    spaceItems={{ default: 'spaceItemsNone' }}
                    alignItems={{ default: 'alignItemsStretch' }}
                    className="pf-u-h-100"
                >
                    <FlexItem flex={{ default: 'flex_1' }}>
                        <PageSection isFilled variant="light" className="pf-u-h-100">
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
                                    justifyContent={{ default: 'justifyContentSpaceBetween' }}
                                    fullWidth={{ default: 'fullWidth' }}
                                    flexWrap={{ default: 'nowrap' }}
                                >
                                    <TextInput
                                        onChange={setFilterTerm}
                                        type="text"
                                        value={filterTerm}
                                        placeholder="Filter by category name..."
                                        id="policy-categories-filter-input"
                                        isDisabled={!!selectedCategory}
                                    />
                                    <PolicyCategoriesFilterSelect
                                        selectedFilters={selectedFilters}
                                        setSelectedFilters={setSelectedFilters}
                                        isDisabled={!!selectedCategory}
                                    />
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
                            <Divider component="div" isVertical />
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
