import React from 'react';
import { Link } from 'react-router-dom-v5-compat';
import { Alert } from '@patternfly/react-core';

import CollapsibleCard from 'Components/CollapsibleCard';
import { getURLLinkToDeployment } from 'Containers/NetworkGraph/utils/networkGraphURLUtils';
import useIsRouteEnabled from 'hooks/useIsRouteEnabled';
import type { Deployment } from 'types/deployment.proto';
import type { Risk, RiskFactor } from 'types/risk.proto';

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
    deployment: Deployment;
    risk: Risk | null | undefined;
};

function RiskIndicators({ deployment, risk }: RiskIndicatorsProps) {
    const isRouteEnabled = useIsRouteEnabled();
    const isRouteEnabledForNetworkGraph = isRouteEnabled('network-graph');

    return (
        <>
            {isRouteEnabledForNetworkGraph && (
                <Link
                    className="btn btn-base h-10 no-underline mt-4 ml-3 mr-3"
                    to={getURLLinkToDeployment({
                        cluster: deployment.clusterName,
                        namespace: deployment.namespace,
                        deploymentId: deployment.id,
                    })}
                >
                    View Deployment in Network Graph
                </Link>
            )}
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
