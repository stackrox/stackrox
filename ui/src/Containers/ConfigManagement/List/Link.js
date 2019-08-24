import React from 'react';
import PropTypes from 'prop-types';
import { Link as RouterLink } from 'react-router-dom';

const Link = ({ url, text, component, pdf }) => {
    function onClick(e) {
        e.stopPropagation();
    }
    // This field is necessary to exclude rendering the Link during PDF generation. It causes an error where the Link can't be rendered outside a Router
    if (pdf) return text;
    return (
        <RouterLink
            to={url}
            className="underline h-full text-left items-center flex text-base-700 hover:text-primary-700"
            onClick={onClick}
        >
            {component || text}
        </RouterLink>
    );
};

Link.propTypes = {
    component: PropTypes.element,
    text: PropTypes.string,
    url: PropTypes.string.isRequired,
    pdf: PropTypes.bool
};

Link.defaultProps = {
    component: null,
    text: null,
    pdf: false
};

export default Link;
