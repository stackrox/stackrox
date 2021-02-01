import React from 'react';
import PropTypes from 'prop-types';

import { mainPath } from 'routePaths';
import NotFoundMessage from 'Components/NotFoundMessage';

const PageNotFound = ({ resourceType, useCase }) => {
    const resourceTypeName = (resourceType || 'resource').toLowerCase();
    const url = useCase ? `${mainPath}/${useCase}` : mainPath;
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
    return <NotFoundMessage message={message} actionText="Go to dashboard" url={url} />;
};

PageNotFound.propTypes = {
    resourceType: PropTypes.string,
    useCase: PropTypes.string,
};

PageNotFound.defaultProps = {
    resourceType: '',
    useCase: '',
};

export default PageNotFound;
