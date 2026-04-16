import { hideColumnIf } from 'hooks/useManagedColumns';
import useIsScannerV4Enabled from 'hooks/useIsScannerV4Enabled';
import useFeatureFlags from 'hooks/useFeatureFlags';

import {
    clusterSearchFilterConfig,
    deploymentSearchFilterConfig,
    imageComponentSearchFilterConfig,
    imageSearchFilterConfig,
    namespaceSearchFilterConfig,
} from '../../searchFilterConfig';
import { getSearchFilterConfigWithFeatureFlagDependency } from 'Components/CompoundSearchFilter/utils/utils';
import ImageCvePage from './ImageCvePage';
import useVulnerabilityState from '../hooks/useVulnerabilityState';

function ImageCvePageRoute() {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const searchFilterConfig = getSearchFilterConfigWithFeatureFlagDependency(
        isFeatureFlagEnabled,
        [
            clusterSearchFilterConfig,
            deploymentSearchFilterConfig,
            imageSearchFilterConfig,
            imageComponentSearchFilterConfig,
            namespaceSearchFilterConfig,
        ]
    );

    const vulnerabilityState = useVulnerabilityState();
    const isScannerV4Enabled = useIsScannerV4Enabled();

    const imageTableColumnOverrides = {
        nvdCvss: hideColumnIf(!isScannerV4Enabled),
    };
    const deploymentTableColumnOverrides = {};

    return (
        <ImageCvePage
            searchFilterConfig={searchFilterConfig}
            showVulnerabilityStateTabs
            vulnerabilityState={vulnerabilityState}
            imageTableColumnOverrides={imageTableColumnOverrides}
            deploymentTableColumnOverrides={deploymentTableColumnOverrides}
        />
    );
}

export default ImageCvePageRoute;
