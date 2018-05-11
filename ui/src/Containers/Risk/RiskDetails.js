import React from 'react';
import PropTypes from 'prop-types';
import CollapsibleCard from 'Components/CollapsibleCard';
import * as Icon from 'react-feather';

const RiskDetails = ({ risk }) =>
    risk.results.map(result => (
        <div className="px-3 py-4" key={result.name}>
            <div
                className="alert-preview bg-white shadow text-primary-600 tracking-wide"
                key={result.name}
            >
                <CollapsibleCard title={result.name}>
                    {result.factors.map(factor => (
                        <div className="flex h-full p-3 font-500" key={factor}>
                            <div>
                                <Icon.Circle className="h-2 w-2 mr-3" />
                            </div>
                            <div className="pl-1">{factor}</div>
                        </div>
                    ))}
                </CollapsibleCard>
            </div>
        </div>
    ));

RiskDetails.propTypes = {
    risk: PropTypes.shape({
        results: PropTypes.array.isRequired
    }).isRequired
};

export default RiskDetails;
