import React from 'react';
import PropTypes from 'prop-types';

const HoverHint = ({ top, left, title, body, subtitle, footer }) => (
    <div
        className="graph-hint visible text-xs text-base-600 absolute border z-10 border-tertiary-400 bg-tertiary-200 rounded min-w-32"
        style={{ top, left }}
    >
        <div className="flex flex-col border-b border-primary-400 mb-1 py-1 px-2 leading-loose">
            <h1 className="graph-hint-title text-sm">{title}</h1>
            {subtitle && <span>{subtitle}</span>}
        </div>
        <div className="graph-hint-body px-2 pt-1 pb-2">{body}</div>
        {!!footer && <footer className="font-700 text-sm leading-loose px-2 pb-1">{footer}</footer>}
    </div>
);

HoverHint.propTypes = {
    top: PropTypes.number,
    left: PropTypes.number,
    title: PropTypes.string.isRequired,
    body: PropTypes.node.isRequired,
    subtitle: PropTypes.string,
    footer: PropTypes.node
};

HoverHint.defaultProps = {
    top: 0,
    left: 0,
    subtitle: '',
    footer: ''
};

export default HoverHint;
