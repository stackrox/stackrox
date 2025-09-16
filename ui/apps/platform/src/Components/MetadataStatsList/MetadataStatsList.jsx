import React from 'react';
import PropTypes from 'prop-types';

const MetadataStatsList = ({ statTiles }) => {
    return (
        <div className="border-b border-base-300 text flex justify-between items-center">
            {statTiles.map((stat, i, arr) => {
                return (
                    <div
                        key={stat.key}
                        className={`flex flex-col p-4 flex-grow justify-center text-center items-center ${
                            i === arr.length - 1 ? 'border-l-2 border-base-300 border-dotted' : ''
                        }`}
                    >
                        {stat}
                    </div>
                );
            })}
        </div>
    );
};

MetadataStatsList.propTypes = {
    statTiles: PropTypes.arrayOf(PropTypes.node),
};

MetadataStatsList.defaultProps = {
    statTiles: null,
};

export default MetadataStatsList;
