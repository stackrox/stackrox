import React from 'react';
import PropTypes from 'prop-types';
import { useDrag } from 'react-dnd';

function PolicyBuilderKey({ fieldKey }) {
    const { name } = fieldKey;
    const [, drag] = useDrag({
        item: { id: name, type: name, fieldKey },
    });
    return (
        <div
            ref={drag}
            className="cursor-move bg-base-400 border border-base-500 flex font-700 text-sm leading-tight h-10 items-center pl-1 rounded text-base-700 mb-2"
            data-testid="draggable-policy-key"
        >
            <span className="drag-grip min-w-4 border-r border-base-500 mr-2" />
            {name}
        </div>
    );
}

PolicyBuilderKey.propTypes = {
    fieldKey: PropTypes.shape({
        name: PropTypes.string.isRequired,
    }).isRequired,
};

export default PolicyBuilderKey;
