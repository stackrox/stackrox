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
            <div className="py-3 pb-2 leading-normal tracking-normal border-b border-base-300">
                {renderedMessage}
            </div>
        </div>
    );
};

Factor.propTypes = {
    message: PropTypes.string.isRequired,
    url: PropTypes.string
};

Factor.defaultProps = {
    url: ''
};

const RiskDetails = ({ risk }) => {
    if (!risk) return <NoResultsMessage message="No Risk Details Available" />;

    return risk.results.map(result => (
        <div className="px-3 pt-5" key={result.name}>
            <div
                className="alert-preview bg-base-100 text-primary-600 tracking-wide"
                key={result.name}
            >
                <CollapsibleCard title={result.name}>
                    {result.factors.map((factor, index) => (
                        <Factor
                            key={`factor.message-${index}`}
                            message={factor.message}
                            url={factor.url}
                        />
                    ))}
                </CollapsibleCard>
            </div>
        </div>
    ));
};

RiskDetails.propTypes = {
    risk: PropTypes.shape({
        results: PropTypes.array.isRequired
    })
};

RiskDetails.defaultProps = {
    risk: null
};

export default RiskDetails;
