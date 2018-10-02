import React from 'react';
import PropTypes from 'prop-types';
import CollapsibleCard from 'Components/CollapsibleCard';
import * as Icon from 'react-feather';
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
        <div className="flex h-full p-3 font-500">
            <div>
                <Icon.Circle className="h-2 w-2 mr-3" />
            </div>
            <div className="pl-1">{renderedMessage}</div>
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
        <div className="px-3 py-4" key={result.name}>
            <div
                className="alert-preview bg-base-100 shadow text-primary-600 tracking-wide"
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
