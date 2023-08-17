import React from 'react';
import { Link } from 'react-router-dom';
import { Alert } from '@patternfly/react-core';

export type TechPreviewBannerProps = {
    featureURL: string;
    featureName: string;
};

function TechPreviewBanner({ featureURL, featureName }: TechPreviewBannerProps) {
    return (
        <Alert
            variant="warning"
            isInline
            title={
                <span>
                    This feature is in its Technology Preview phase. For optimal production
                    performance, please rely on <Link to={featureURL}>{featureName}</Link>.
                </span>
            }
        />
    );
}

export default TechPreviewBanner;
