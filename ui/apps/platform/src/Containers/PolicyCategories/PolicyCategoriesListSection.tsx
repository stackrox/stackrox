import React from 'react';
import { PageSection, Title, Flex } from '@patternfly/react-core';

import PolicyCategoriesList from './PolicyCategoriesList';

type PolicyCategoriesListSectionProps = {
    policyCategories: string[];
};

function PolicyCategoriesListSection({ policyCategories }: PolicyCategoriesListSectionProps) {
    // const [selectedCategory, setSelectedCategory] = useState('');
    return (
        <PageSection isFilled id="policy-categories-list-section">
            <PageSection isFilled variant="light">
                <Flex
                    justifyContent={{ default: 'justifyContentSpaceBetween' }}
                    fullWidth={{ default: 'fullWidth' }}
                >
                    <Title headingLevel="h3">Categories</Title>
                    <Title headingLevel="h3">{policyCategories.length} results found</Title>
                </Flex>
                <PolicyCategoriesList policyCategories={policyCategories} />
            </PageSection>
        </PageSection>
    );
}

export default PolicyCategoriesListSection;
