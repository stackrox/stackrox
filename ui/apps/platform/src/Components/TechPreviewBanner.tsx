import React from 'react';
import { Link } from 'react-router-dom';
import { Alert } from '@patternfly/react-core';

import useIsRouteEnabled from 'hooks/useIsRouteEnabled';
import { RouteKey } from 'routePaths';

export type TechPreviewBannerProps = {
    featureURL: string;
    featureName: string;
    routeKey: RouteKey;
};

function TechPreviewBanner({ featureURL, featureName, routeKey }: TechPreviewBannerProps) {
    const isRouteEnabled = useIsRouteEnabled();
    const hasLink = isRouteEnabled(routeKey);

    return (
        <Alert
            variant="warning"
            isInline
            component="p"
            title={
                <span>
                    This feature is in its Technology Preview phase.{' '}
                    {hasLink && (
                        <>
                            For optimal production performance, please rely on{' '}
                            <Link to={featureURL}>{featureName}</Link>.
                        </>
                    )}
                </span>
            }
        />
    );
}

export default TechPreviewBanner;
