import React from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router-dom';
import Loader from 'Components/Loader';

const TileLink = ({ value, caption, to, loading }) => {
    const content = loading ? (
        <Loader message="" />
    ) : (
        <>
            <div className="text-3xl tracking-widest">{value}</div>
            <div className="text-sm pt-1 tracking-wide uppercase font-condensed">
                {value === 1 ? caption : `${caption}s`}
            </div>
        </>
    );
    return (
        <Link to={to} className="no-underline">
            <div className="flex flex-col border border-base-400 text-base-500 items-center justify-center font-600 p-2 px-4 min-w-24 rounded-sm hover:bg-base-200">
                {content}
            </div>
        </Link>
    );
};

TileLink.propTypes = {
    value: PropTypes.number.isRequired,
    caption: PropTypes.string.isRequired,
    to: PropTypes.string.isRequired,
    loading: PropTypes.bool.isRequired
};

export default TileLink;
