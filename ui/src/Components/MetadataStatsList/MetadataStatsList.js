import React from 'react';
import PropTypes from 'prop-types';

const MetadataStatsList = ({ statTiles }) => {
    return (
        <div className="border-b border-base-300 text-base-500 flex justify-between items-center">
            {statTiles.map((stat, i, arr) => (
                <div
                    className={`flex flex-col p-4 flex-grow justify-center text-center ${
                        i !== arr.length - 1 ? 'border-r-2 border-base-300 border-dotted' : ''
                    }`}
                >
                    {stat}
                </div>
            ))}
        </div>
    );
};

MetadataStatsList.propTypes = {
    statTiles: PropTypes.arrayOf(PropTypes.node)
};

MetadataStatsList.defaultProps = {
    statTiles: null
};

export default MetadataStatsList;
