import React from 'react';
import { Title, Divider, Flex } from '@patternfly/react-core';

import PolicyCriteriaCategory from './PolicyCriteriaCategory';

function getKeysByCategory(keys) {
    const categories = {};
    keys.forEach((key) => {
        const { category } = key;
        if (categories[category]) {
            categories[category].push(key);
        } else {
            categories[category] = [key];
        }
    });
    return categories;
}

function PolicyCriteriaKeys({ keys }) {
    const [categories, setCategories] = React.useState(getKeysByCategory(keys));

    React.useEffect(() => {
        setCategories(getKeysByCategory(keys));
    }, [keys]);

    return (
        <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsNone' }}>
            <Title headingLevel="h2">Drag out policy fields</Title>
            <Divider component="div" className="pf-u-mb-sm pf-u-mt-md" />
            {Object.keys(categories).map((category, idx) => (
                <div key={category}>
                    <PolicyCriteriaCategory
                        category={category}
                        keys={categories[category]}
                        isOpenDefault={idx === 0}
                    />
                    <Divider component="div" className="pf-u-mb-sm pf-u-mt-sm" />
                </div>
            ))}
        </Flex>
    );
}

export default PolicyCriteriaKeys;
