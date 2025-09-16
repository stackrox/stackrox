import React from 'react';
import PropTypes from 'prop-types';

/*
 * Recommended for optional string value or Redux form element as children.
 * Not recommeded for input element as children because component does not render label with htmlFor prop.
 */
function Labeled({ label, children }) {
    if (!React.Children.count(children)) {
        return null;
    } // don't render w/o children
    return (
        <div className="mb-4" data-testid="labeled-key-value-pair">
            <div className="py-1 text-base-600 font-700" data-testid="labeled-key">
                {label}
            </div>
            <div className="w-full py-1" data-testid="labeled-value">
                {children}
            </div>
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
