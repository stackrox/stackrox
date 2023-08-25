import React from 'react';
import PropTypes from 'prop-types';
import CollapsibleCard from 'Components/CollapsibleCard';
import NoResultsMessage from 'Components/NoResultsMessage';

const Factor = ({ message, url }) => {
    const renderedMessage = url ? (
        <a href={url} target="_blank" rel="noopener noreferrer">
            {message}
        </a>
    ) : (
        message
    );

    return (
        <div className="px-3">
            <div className="py-3 pb-2 leading-normal border-b border-base-300">
                {renderedMessage}
            </div>
        </div>
    );
};

Factor.propTypes = {
    message: PropTypes.string.isRequired,
    url: PropTypes.string,
};

Factor.defaultProps = {
    url: '',
};

const RiskDetails = ({ risk }) => {
    if (!risk) {
        return (
            <NoResultsMessage message="Risk details are being calculated. Please check back shortly." />
        );
    }

    return risk.results.map((result) => (
        <div className="px-3 pt-5" key={result.name}>
            <div className="alert-preview bg-base-100 text-primary-600">
                <CollapsibleCard title={result.name}>
                    {result.factors.map((factor, index) => (
                        // eslint-disable-next-line react/no-array-index-key
                        <Factor key={index} message={factor.message} url={factor.url} />
                    ))}
                </CollapsibleCard>
            </div>
        </div>
    ));
};

RiskDetails.propTypes = {
    risk: PropTypes.shape({
        results: PropTypes.arrayOf(
            PropTypes.shape({
                name: PropTypes.string,
                factors: PropTypes.arrayOf(
                    PropTypes.shape({
                        message: PropTypes.string,
                        url: PropTypes.string,
                    })
                ),
            })
        ).isRequired,
    }),
};

RiskDetails.defaultProps = {
    risk: null,
};

export default RiskDetails;
