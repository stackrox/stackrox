import React, { useState } from 'react';
import { PageSection, Drawer, DrawerContent, DrawerContentBody } from '@patternfly/react-core';

import PolicyCategoriesList from './PolicyCategoriesList';

type PolicyCategoriesListSectionProps = {
    policyCategories: string[];
};

function PolicyCategoriesListSection({ policyCategories }: PolicyCategoriesListSectionProps) {
    const [selectedCategory, setSelectedCategory] = useState('');
    return (
        <PageSection isFilled id="policy-categories-list-section">
            <Drawer isExpanded={!!selectedCategory} isInline>
                <DrawerContent panelContent="hi">
                    <DrawerContentBody>
                        <PolicyCategoriesList policyCategories={policyCategories} />
                    </DrawerContentBody>
                </DrawerContent>
            </Drawer>
        </PageSection>
    );
}

export default PolicyCategoriesListSection;
