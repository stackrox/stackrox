import React from 'react';
import { ClipLoader } from 'react-spinners';
import PropTypes from 'prop-types';

const Loader = ({ message }) => (
    <div className="flex flex-col items-center justify-center h-full w-full">
        <ClipLoader loading size={20} />
        <div className="text-lg font-sans tracking-wide mt-4">{message}</div>
    </div>
);

Loader.propTypes = {
    message: PropTypes.string
};

Loader.defaultProps = {
    message: 'Loading...'
};

export default Loader;
