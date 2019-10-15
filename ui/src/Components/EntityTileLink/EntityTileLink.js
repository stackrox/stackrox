import React from 'react';
import PropTypes from 'prop-types';
import pluralize from 'pluralize';
import entityTypes from 'constants/entityTypes';

import TileLink, { POSITION } from 'Components/TileLink';

const EntityTileLink = ({ count, entityType, position, subText, icon, url, loading, isError }) => {
    const text = `${count} ${count === 1 ? entityType : `${pluralize(entityType)}`}`;

    return (
        <TileLink
            text={text}
            subText={subText}
            position={position}
            icon={icon}
            url={url}
            loading={loading}
            isError={isError}
        />
    );
};

EntityTileLink.propTypes = {
    count: PropTypes.number.isRequired,
    entityType: PropTypes.oneOf(Object.values(entityTypes)).isRequired,
    subText: PropTypes.string,
    icon: PropTypes.element,
    url: PropTypes.string.isRequired,
    loading: PropTypes.bool,
    isError: PropTypes.bool,
    position: PropTypes.oneOf(Object.values(POSITION))
};

EntityTileLink.defaultProps = {
    isError: false,
    position: null,
    loading: false,
    subText: null,
    icon: null
};

export default EntityTileLink;
