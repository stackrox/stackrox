import React from 'react';

import {
    imageCVESearchFilterConfig,
    imageComponentSearchFilterConfig,
} from 'Containers/Vulnerabilities/searchFilterConfig';
import { getSearchFilterConfigWithFeatureFlagDependency } from 'Components/CompoundSearchFilter/utils/utils';
import useFeatureFlags from 'hooks/useFeatureFlags';

import ImagePage from './ImagePage';

function ImagePageRoute() {
    const { isFeatureFlagEnabled } = useFeatureFlags();

    const isEpssProbabilityColumnEnabled = isFeatureFlagEnabled('ROX_SCANNER_V4');
    return <ImagePage showVulnerabilityStateTabs />;
}

export default ImagePageRoute;
