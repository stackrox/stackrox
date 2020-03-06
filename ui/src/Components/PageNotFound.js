import React from 'react';
import { withRouter } from 'react-router-dom';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';

import URLService from 'modules/URLService';
import NotFoundMessage from 'Components/NotFoundMessage';

const PageNotFound = ({ match, location, resourceType }) => {
    const homeUrl = URLService.getURL(match, location)
        .base()
        .url();

    const resourceTypeName = (resourceType || 'resource').toLowerCase();
    const message = (
        <>
            <h2 className="text-tertiary-800 mb-2">
                {`Unfortunately, the ${resourceTypeName} you are looking for cannot be found.`}
            </h2>
            <p className="text-tertiary-800 mb-8">
                It may have changed, did not exist, or no longer exists. Try using search from the
                dashboard to find what you&apos;re looking for.
            </p>
        </>
    );
    return <NotFoundMessage message={message} actionText="Go to dashboard" url={homeUrl} />;
};

PageNotFound.propTypes = {
    resourceType: PropTypes.string,
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired
};

PageNotFound.defaultProps = {
    resourceType: null
};

export default withRouter(PageNotFound);
