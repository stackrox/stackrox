import React from 'react';
import PropTypes from 'prop-types';

/*
 * Render auxiliary information (following a label chip) as small text.
 *
 * A space precedes children for inline layout (especially in PDF Export).
 * The space is ignored in Web UI for flex-col layout.
 */
const LabelChipSubtext = ({ children }) => (
    <span className="pt-1 text-base-500 text-xs font-700 text-center"> {children}</span>
);

LabelChipSubtext.propTypes = {
    children: PropTypes.node.isRequired,
};

export default LabelChipSubtext;
