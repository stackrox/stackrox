import { Alert } from '@patternfly/react-core';

import CollapsibleCard from 'Components/CollapsibleCard';
import type { Risk, RiskFactor } from 'services/DeploymentsService';

type FactorProps = {
    factor: RiskFactor;
};

function Factor({ factor: { message, url } }: FactorProps) {
    // TODO is the link external or internal?
    /* eslint-disable generic/ExternalLink-anchor */
    const renderedMessage = url ? (
        <a href={url} target="_blank" rel="noopener noreferrer">
            {message}
        </a>
    ) : (
        message
    );
    /* eslint-enable generic/ExternalLink-anchor */

    return (
        <div className="px-3">
            <div className="py-3 pb-2 leading-normal border-b border-base-300">
                {renderedMessage}
            </div>
        </div>
    );
}

export type RiskIndicatorsProps = {
    risk: Risk | null | undefined;
};

function RiskIndicators({ risk }: RiskIndicatorsProps) {
    return (
        <>
            {Array.isArray(risk?.results) ? (
                risk.results.map((result) => (
                    <div className="px-3 pt-5" key={result.name}>
                        <div className="alert-preview bg-base-100 text-primary-600">
                            <CollapsibleCard title={result.name}>
                                {result.factors.map((factor, index) => (
                                    // eslint-disable-next-line react/no-array-index-key
                                    <Factor key={index} factor={factor} />
                                ))}
                            </CollapsibleCard>
                        </div>
                    </div>
                ))
            ) : (
                <Alert variant="warning" isInline title="Risk not found" component="p">
                    Risk for selected deployment may not have been processed.
                </Alert>
            )}
        </>
    );
}

export default RiskIndicators;
