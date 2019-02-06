import React from 'react';
import PropTypes from 'prop-types';

const Labels = ({ list }) => (
    <ul className={` ${list.length > 4 ? `columns-2` : ``} list-reset p-3 w-full leading-normal`}>
        {list.map(label => (
            <li
                key={label}
                className="border-b border-base-300 p-2 truncate"
                style={{
                    'column-break-inside': 'avoid',
                    'page-break-inside': 'avoid'
                }}
            >
                {label}
            </li>
        ))}
    </ul>
);

Labels.propTypes = {
    list: PropTypes.string.isRequired
};

export default Labels;
