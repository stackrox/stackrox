import React from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router-dom';

const ButtonLink = ({ children, dataTestId, extraClassNames, icon, linkTo }) => {
    return (
        <Link
            to={linkTo}
            className={`no-underline btn btn-base ${extraClassNames}`}
            data-testid={dataTestId}
        >
            {icon}
            {children}
        </Link>
    );
};

ButtonLink.propTypes = {
    children: PropTypes.node.isRequired,
    dataTestId: PropTypes.string,
    extraClassNames: PropTypes.string,
    icon: PropTypes.element,
    linkTo: PropTypes.string.isRequired,
};

ButtonLink.defaultProps = {
    dataTestId: 'button-link',
    extraClassNames: '',
    icon: null,
};

export default ButtonLink;
