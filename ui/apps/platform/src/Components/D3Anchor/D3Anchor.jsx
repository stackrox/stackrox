import React, { useRef, useEffect } from 'react';
import PropTypes from 'prop-types';
import { select } from 'd3-selection';

// This will be a reusable Component that we can use to render the container for D3 elements
// The "children" prop will render any other nested elements
// The "onUpdate" prop will be run on every update and pass the ref for the container as a parameter
const D3Anchor = ({ dataTestId, translateX, translateY, onUpdate, children }) => {
    const refAnchor = useRef(null);

    useEffect(() => {
        onUpdate(select(refAnchor.current));
    });

    return (
        <g
            data-testid={dataTestId}
            ref={refAnchor}
            transform={`translate(${translateX}, ${translateY})`}
        >
            {children}
        </g>
    );
};

D3Anchor.propTypes = {
    dataTestId: PropTypes.string.isRequired,
    translateX: PropTypes.number,
    translateY: PropTypes.number,
    onUpdate: PropTypes.func.isRequired,
    children: PropTypes.oneOfType([PropTypes.arrayOf(PropTypes.node), PropTypes.node]),
};

D3Anchor.defaultProps = {
    translateX: 0,
    translateY: 0,
    children: null,
};

export default D3Anchor;
