import React from 'react';
import PropTypes from 'prop-types';

import NameListItem from './NameListItem';

const NameList = ({ names, onClick }) => {
    return (
        <ul className="h-full mt-3 border-t border-base-300" data-testid="timeline-names-list">
            {names.map(
                ({ type, id, name, subText, hasChildren = false, drillDownButtonTooltip }) => {
                    return (
                        <NameListItem
                            key={id}
                            id={id}
                            type={type}
                            name={name}
                            subText={subText}
                            hasChildren={hasChildren}
                            onClick={onClick}
                            drillDownButtonTooltip={drillDownButtonTooltip}
                        />
                    );
                }
            )}
        </ul>
    );
};

NameList.propTypes = {
    names: PropTypes.arrayOf(
        PropTypes.shape({
            type: PropTypes.string.isRequired,
            id: PropTypes.string.isRequired,
            name: PropTypes.string.isRequired,
            subText: PropTypes.string.isRequired,
            hasChildren: PropTypes.bool.isRequired,
        })
    ),
    onClick: PropTypes.func, // @TODO: Make this required when we start working with changing views
};

NameList.defaultProps = {
    names: [],
    onClick: () => {},
};

export default NameList;
