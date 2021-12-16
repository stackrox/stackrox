import React from 'react';
import { useDrag } from 'react-dnd';

import './PolicyCriteriaKey.css';

function PolicyCriteriaKey({ fieldKey }) {
    const { name, shortName } = fieldKey;
    const [, drag] = useDrag({
        type: name,
        item: { id: name, type: name, fieldKey },
    });

    return (
        <div
            key={name}
            ref={drag}
            className="pf-u-p-sm pf-u-mb-md pf-u-display-flex policy-criteria-key"
        >
            <span className="draggable-grip" />
            {shortName || name}
        </div>
    );
}

export default PolicyCriteriaKey;
