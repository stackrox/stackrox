import React from 'react';
import PropTypes from 'prop-types';

const RiskScore = ({ score }) => {
    return (
        <div className="flex justify-center items-center">
            <span className="pr-1 text-xl">Risk priority:</span>
            <span className="pl-1 text-3xl">{score}</span>
        </div>
    );
};

RiskScore.propTypes = {
    score: PropTypes.oneOfType([PropTypes.string, PropTypes.number]).isRequired,
};

export default RiskScore;
