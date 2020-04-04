import React from 'react';
import PropTypes from 'prop-types';
import { useDrag } from 'react-dnd';

import DRAG_DROP_TYPES from 'constants/dragDropTypes';

function PolicyBuilderKey({ label, jsonpath }) {
    // eslint-disable-next-line no-unused-vars
    const [collectedProps, drag] = useDrag({
        item: { id: jsonpath, type: DRAG_DROP_TYPES.KEY }
    });
    return (
        <div
            ref={drag}
            className="cursor-move bg-base-400 border border-base-500 flex font-700 text-sm h-10 items-center pl-1 rounded text-base-700 mb-2"
        >
            <span className="drag-grip border-r border-base-500 mr-3" />
            {label}
        </div>
    );
}

PolicyBuilderKey.propTypes = {
    label: PropTypes.string.isRequired,
    jsonpath: PropTypes.string.isRequired
};

export default PolicyBuilderKey;
