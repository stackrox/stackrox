import React from 'react';
import PropTypes from 'prop-types';

import HeaderWithSubText from 'Components/HeaderWithSubText';
import DrillDownButton from './DrillDownButton';

const NameListItem = ({
    id,
    type,
    name,
    subText,
    hasChildren,
    onClick,
    drillDownButtonTooltip,
}) => {
    function onClickHandler() {
        onClick(type, id);
    }

    return (
        <li className="flex flex-col justify-center leading-normal relative h-12 border-b border-base-300">
            <HeaderWithSubText header={name} subText={subText} />
            {hasChildren && (
                <DrillDownButton tooltip={drillDownButtonTooltip} onClick={onClickHandler} />
            )}
        </li>
    );
};

NameListItem.propTypes = {
    id: PropTypes.string.isRequired,
    type: PropTypes.string.isRequired,
    name: PropTypes.string.isRequired,
    subText: PropTypes.string.isRequired,
    hasChildren: PropTypes.bool.isRequired,
    onClick: PropTypes.func,
    drillDownButtonTooltip: PropTypes.string,
};

NameListItem.defaultProps = {
    onClick: () => {},
    drillDownButtonTooltip: null,
};

export default NameListItem;
