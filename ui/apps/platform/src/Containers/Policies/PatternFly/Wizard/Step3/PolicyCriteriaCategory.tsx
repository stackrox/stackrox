import React from 'react';
import { ExpandableSection, Draggable, Droppable, Divider } from '@patternfly/react-core';

import { Descriptor } from 'Containers/Policies/Wizard/Form/descriptors';
import './PolicyCriteriaCategory.css';

type PolicyCriteriaCategoryProps = {
    category: string;
    keys: Descriptor[];
};

function PolicyCriteriaCategory({ category, keys }: PolicyCriteriaCategoryProps) {
    const [isExpanded, setIsExpanded] = React.useState(false);

    function onToggle(expanded: boolean) {
        setIsExpanded(expanded);
    }

    return (
        <ExpandableSection isExpanded={isExpanded} onToggle={onToggle} toggleText={category}>
            <Droppable>
                {keys.map(({ name, shortName }) => (
                    <Draggable
                        key={name}
                        className="pf-u-p-sm pf-u-mb-md pf-u-display-flex policy-criteria-key"
                    >
                        <span className="draggable-grip" />
                        {shortName || name}
                    </Draggable>
                ))}
            </Droppable>
        </ExpandableSection>
    );
}

export default PolicyCriteriaCategory;
