import React from 'react';
import PropTypes from 'prop-types';

const HoverHint = props => (
    <div
        className="graph-hint visible text-xs absolute p-2 pb-1 pt-1 border border-primary-500 bg-primary-300 opacity-75 rounded min-w-32"
        style={{ top: props.top, left: props.left }}
    >
        <h1 className="graph-hint-title text-uppercase border-b border-grey-light leading-loose text-xs mb-1">
            {props.title}
        </h1>
        <div className="graph-hint-body text-xs">{props.body}</div>
    </div>
);

HoverHint.propTypes = {
    top: PropTypes.number,
    left: PropTypes.number,
    title: PropTypes.string.isRequired,
    body: PropTypes.node.isRequired
};

HoverHint.defaultProps = {
    top: 0,
    left: 0
};

export default HoverHint;
