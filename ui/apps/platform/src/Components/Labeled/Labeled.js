import React from 'react';
import PropTypes from 'prop-types';

function Labeled({ label, children }) {
    if (!React.Children.count(children)) return null; // don't render w/o children
    return (
        <div className="mb-4">
            <div className="py-1 text-base-600 font-700">{label}</div>
            <div className="w-full py-1">{children}</div>
        </div>
    );
}

Labeled.propTypes = {
    label: PropTypes.node.isRequired,
    children: PropTypes.node,
};

Labeled.defaultProps = {
    children: null,
};

export default Labeled;
