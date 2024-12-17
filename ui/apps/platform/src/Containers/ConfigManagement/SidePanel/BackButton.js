import React from 'react';
import PropTypes from 'prop-types';
import { Link, useLocation, useMatch } from 'react-router-dom';
import URLService from 'utils/URLService';

import { ArrowLeft } from 'react-feather';
import EntityIcon from 'Components/EntityIcon';
import { workflowPaths } from 'routePaths';

const BackButton = ({ entityType1, entityListType2, entityId2 }) => {
    const location = useLocation();
    const match = useMatch(workflowPaths.LIST);

    if (entityListType2 || entityId2) {
        const link = URLService.getURL(match, location).pop().url();
        return (
            <Link
                className="flex items-center justify-center text-base-600 border-r border-base-300 px-4 mr-4 h-full hover:bg-primary-200 w-16"
                to={link}
                aria-label="Go to preceding breadcrumb"
            >
                <ArrowLeft className="h-6 w-6 text-600" />
            </Link>
        );
    }
    return (
        <EntityIcon
            className="flex items-center justify-center border-r border-base-300 px-4 mr-4 h-full w-16"
            entityType={entityType1}
        />
    );
};

BackButton.propTypes = {
    entityType1: PropTypes.string,
    entityListType2: PropTypes.string,
    entityId2: PropTypes.string,
};

BackButton.defaultProps = {
    entityType1: null,
    entityListType2: null,
    entityId2: null,
};

export default BackButton;
