import React from 'react';
import PropTypes from 'prop-types';

const HoverHint = ({ top, left, title, body, footer }) => (
    <div
        className="graph-hint visible text-xs text-base-600 absolute p-2 pb-1 pt-1 border z-10 border-tertiary-400 bg-tertiary-200 rounded min-w-32"
        style={{ top, left }}
    >
        <h1 className="graph-hint-title border-b border-grey-light text-sm leading-loose mb-1 py-1">
            {title}
        </h1>
        <div className="graph-hint-body py-2">{body}</div>
        {!!footer && (
            <footer className="font-700 border-t border-grey-light text-sm leading-loose mt-1 py-1">
                {footer}
            </footer>
        )}
    </div>
);

HoverHint.propTypes = {
    top: PropTypes.number,
    left: PropTypes.number,
    title: PropTypes.string.isRequired,
    body: PropTypes.node.isRequired,
    footer: PropTypes.node
};

HoverHint.defaultProps = {
    top: 0,
    left: 0,
    footer: ''
};

export default HoverHint;
