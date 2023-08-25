import React from 'react';
import { useDrag } from 'react-dnd';
import { Flex } from '@patternfly/react-core';

import './PolicyCriteriaKey.css';

function PolicyCriteriaKey({ fieldKey }) {
    const { name, shortName } = fieldKey;
    const [, drag] = useDrag({
        type: name,
        item: { id: name, type: name, fieldKey },
    });

    return (
        <div ref={drag} className="pf-u-p-sm pf-u-mb-md policy-criteria-key">
            <Flex alignItems={{ default: 'alignItemsCenter' }} flexWrap={{ default: 'nowrap' }}>
                <span className="draggable-grip" />
                {shortName || name}
            </Flex>
        </div>
    );
}

export default PolicyCriteriaKey;
