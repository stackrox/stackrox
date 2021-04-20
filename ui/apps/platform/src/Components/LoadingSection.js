import React from 'react';
import PropTypes from 'prop-types';
import { ClipLoader } from 'react-spinners';

const LoadingSection = ({ message }) => (
    <div className="flex flex-col items-center justify-center h-full">
        <ClipLoader color="white" loading size={20} />
        <div className="mt-4">{message}</div>
    </div>
);

LoadingSection.propTypes = {
    message: PropTypes.string,
};

LoadingSection.defaultProps = {
    message: 'Loading...',
};

export default LoadingSection;
