import React from 'react';
import PropTypes from 'prop-types';
import { ClipLoader as Loader } from 'react-spinners';

const GraphLoader = ({ isLoading }) => {
    if (!isLoading) return null;
    return (
        <div className="flex flex-col items-center text-center">
            <div className="w-10 rounded-full p-2 bg-base-100 shadow-lg mb-4">
                <Loader loading size={20} color="currentColor" />
            </div>
            <div className="uppercase text-sm tracking-widest font-700">Generating Graph...</div>
        </div>
    );
};

GraphLoader.propTypes = {
    isLoading: PropTypes.bool.isRequired
};

export default GraphLoader;
