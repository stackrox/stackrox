import React from 'react';
import PropTypes from 'prop-types';
import { Link, withRouter } from 'react-router-dom';
import URLService from 'modules/URLService';
import * as Icon from 'react-feather';

const AppLink = ({ context, pageType, params, children, externalLink, ...rest }) => {
    const { staticContext, entityType, ...linkParams } = rest;
    const to = URLService.getLinkTo(context, pageType, params);

    return (
        <div className="flex items-center">
            <Link to={to} {...linkParams}>
                {children}
            </Link>
            {externalLink && (
                <Link
                    rel="noopener noreferrer"
                    className="mx-2 text-primary-700 hover:text-primary-800 p-1 bg-primary-300 rounded"
                    target="_blank"
                    to={to}
                >
                    <Icon.ExternalLink size="14" />
                </Link>
            )}
        </div>
    );
};

AppLink.propTypes = {
    context: PropTypes.string.isRequired,
    pageType: PropTypes.string.isRequired,
    externalLink: PropTypes.bool,
    params: PropTypes.shape({}).isRequired
};

AppLink.defaultProps = {
    externalLink: false
};

export default withRouter(AppLink);
