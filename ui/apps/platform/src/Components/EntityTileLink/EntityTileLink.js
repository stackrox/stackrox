import React from 'react';
import PropTypes from 'prop-types';
import pluralize from 'pluralize';
import entityTypes from 'constants/entityTypes';
import { resourceLabels } from 'messages/common';

import TileLink, { POSITION } from 'Components/TileLink';

const EntityTileLink = ({
    count,
    entityType,
    position,
    superText,
    subText,
    icon,
    url,
    loading,
    isError,
    short,
}) => {
    const resourceLabel = resourceLabels[entityType] || '';
    const text = `${count} ${count === 1 ? resourceLabel : `${pluralize(resourceLabel)}`}`;

    return (
        <TileLink
            text={text}
            superText={superText}
            subText={subText}
            position={position}
            icon={icon}
            url={url}
            short={short}
            loading={loading}
            isError={isError}
        />
    );
};

EntityTileLink.propTypes = {
    count: PropTypes.number.isRequired,
    entityType: PropTypes.oneOf(Object.values(entityTypes)).isRequired,
    superText: PropTypes.oneOfType([PropTypes.string, PropTypes.number]),
    subText: PropTypes.string,
    icon: PropTypes.element,
    url: PropTypes.string.isRequired,
    loading: PropTypes.bool,
    isError: PropTypes.bool,
    position: PropTypes.oneOf(Object.values(POSITION)),
    short: PropTypes.bool,
};

EntityTileLink.defaultProps = {
    isError: false,
    position: null,
    loading: false,
    superText: null,
    subText: null,
    icon: null,
    short: false,
};

export default EntityTileLink;
