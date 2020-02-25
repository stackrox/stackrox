import React from 'react';
import PropTypes from 'prop-types';

import TooltipOverlay from 'Components/TooltipOverlay';

const HoverHint = ({ top, left, title, body, subtitle, footer }) => (
    <TooltipOverlay
        top={top}
        left={left}
        title={title}
        body={body}
        subtitle={subtitle}
        footer={footer}
        className="visible absolute"
    />
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
