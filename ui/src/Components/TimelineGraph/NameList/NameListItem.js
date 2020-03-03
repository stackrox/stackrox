import React from 'react';
import PropTypes from 'prop-types';

import { ChevronRight } from 'react-feather';
import Button from 'Components/Button';

const NameListItem = ({ id, type, name, subText, hasChildren, onClick }) => {
    function onClickHandler() {
        onClick(type, id);
    }
    return (
        <li className="flex flex-col justify-center p-3 leading-normal relative h-12 first:border-t first:border-base-300">
            <div className="font-700 text-base-600">{name}</div>
            <div className="text-base-500 text-xs font-700">{subText}</div>
            {hasChildren && (
                <Button
                    className="absolute bg-base-100 border border-primary-300 center-y py-1 right-0 rounded transform translate-x-1/2 hover:bg-primary-200"
                    onClick={onClickHandler}
                    icon={<ChevronRight className="h-4 w-4 text-base-700" />}
                />
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
    onClick: PropTypes.func
};

NameListItem.defaultProps = {
    onClick: () => {}
};

export default NameListItem;
