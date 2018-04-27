import React from 'react';
import PropTypes from 'prop-types';
import { ClipLoader } from 'react-spinners';

const LoadingSection = ({ message }) => (
    <section className="flex flex-col items-center justify-center h-full bg-primary-600">
        <ClipLoader color="white" loading size={20} />
        <div className="text-lg font-sans text-white tracking-wide mt-4">{message}</div>
    </section>
);

LoadingSection.propTypes = {
    message: PropTypes.string
};

LoadingSection.defaultProps = {
    message: 'Loading...'
};

export default LoadingSection;
