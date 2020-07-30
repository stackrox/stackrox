import React from 'react';
import PropTypes from 'prop-types';

const InfoList = ({ items, renderItem, extraClassNames }) => {
    return (
        <ul
            className={`bg-base-100 border-2 rounded p-2 border-base-300 w-full font-600 text-base-600 hover:border-base-400 leading-normal last:mb-0 overflow-scroll ${extraClassNames}`}
        >
            {items.map(renderItem)}
        </ul>
    );
};

InfoList.propTypes = {
    extraClassNames: PropTypes.string,
    items: PropTypes.arrayOf(PropTypes.any),
    renderItem: PropTypes.func,
};

InfoList.defaultProps = {
    extraClassNames: '',
    items: [],
    renderItem: (item) => <li key={item.id}>{item.toString()}</li>,
};

export default InfoList;
