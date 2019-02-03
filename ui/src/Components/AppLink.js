import React from 'react';
import PropTypes from 'prop-types';
import { Link, withRouter } from 'react-router-dom';
import URLService from 'modules/URLService';

const AppLink = ({ context, pageType, params, children, ...rest }) => {
    const { staticContext, entityType, ...linkParams } = rest;
    const to = URLService.getLinkTo(context, pageType, params);

    return (
        <Link to={to} {...linkParams}>
            {children}
        </Link>
    );
};

AppLink.propTypes = {
    context: PropTypes.string.isRequired,
    pageType: PropTypes.string.isRequired,
    params: PropTypes.shape({}).isRequired
};

export default withRouter(AppLink);
