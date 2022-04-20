import React from 'react';
import { ExpandableSection } from '@patternfly/react-core';

import { Descriptor } from './policyCriteriaDescriptors';
import PolicyCriteriaKey from './PolicyCriteriaKey';

type PolicyCriteriaCategoryProps = {
    category: string;
    keys: Descriptor[];
    isOpenDefault?: boolean;
};

function PolicyCriteriaCategory({
    category,
    keys,
    isOpenDefault = false,
}: PolicyCriteriaCategoryProps) {
    const [isExpanded, setIsExpanded] = React.useState(isOpenDefault);

    function onToggle(expanded: boolean) {
        setIsExpanded(expanded);
    }

    return (
        <ExpandableSection
            isExpanded={isExpanded}
            onToggle={onToggle}
            toggleText={category}
            data-testid="policy-criteria-key-group"
        >
            {keys.map((key) => (
                <PolicyCriteriaKey fieldKey={key} key={key.name} />
            ))}
        </ExpandableSection>
    );
}

export default PolicyCriteriaCategory;
