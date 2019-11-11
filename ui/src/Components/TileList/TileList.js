import React from 'react';
import PropTypes from 'prop-types';

import TileLink from 'Components/TileLink';

const TileList = ({ items, title }) => {
    return (
        <div className="bg-primary-200 text-base-600 rounded border border-primary-400 m-2">
            {title !== '' && (
                <h3 className="border-b border-base-400 text-xs text-base-600 uppercase text-center tracking-wide p-1 leading-normal font-700">
                    {title}
                </h3>
            )}
            <ul className="list-reset">
                {items.map(item => (
                    <li className="p-2" key={item.label}>
                        <TileLink superText={item.count} text={item.label} url={item.url} />
                    </li>
                ))}
            </ul>
        </div>
    );
};

TileList.propTypes = {
    items: PropTypes.arrayOf(
        PropTypes.shape({
            label: PropTypes.string,
            count: PropTypes.oneOfType([PropTypes.number, PropTypes.string]),
            url: PropTypes.string
        })
    ).isRequired,
    title: PropTypes.string
};

TileList.defaultProps = {
    title: ''
};

export default TileList;
