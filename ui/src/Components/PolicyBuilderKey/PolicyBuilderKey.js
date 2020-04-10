import React from 'react';
import PropTypes from 'prop-types';
import { useDrag } from 'react-dnd';

import DRAG_DROP_TYPES from 'constants/dragDropTypes';

function PolicyBuilderKey({ fieldKey }) {
    const { name } = fieldKey;
    // eslint-disable-next-line no-unused-vars
    const [collectedProps, drag] = useDrag({
        item: { id: name, type: DRAG_DROP_TYPES.KEY, fieldKey }
    });
    return (
        <div
            ref={drag}
            className="cursor-move bg-base-400 border border-base-500 flex font-700 text-sm h-10 items-center pl-1 rounded text-base-700 mb-2"
            data-testid="draggable-policy-key"
        >
            <span className="drag-grip min-w-4 border-r border-base-500 mr-3" />
            {name}
        </div>
    );
}

PolicyBuilderKey.propTypes = {
    fieldKey: PropTypes.shape({
        name: PropTypes.string.isRequired
    }).isRequired
};

export default PolicyBuilderKey;
