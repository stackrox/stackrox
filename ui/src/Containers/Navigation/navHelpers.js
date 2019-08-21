import { isBackendFeatureFlagEnabled } from 'utils/featureFlags';

export function filterLinksByFeatureFlag(flagsToUse, navLinks, defaultVal = false) {
    return navLinks.filter(navLink => {
        if (!navLink.featureFlag) return true;
        return isBackendFeatureFlagEnabled(flagsToUse, navLink.featureFlag, defaultVal);
    });
}

export default {
    filterLinksByFeatureFlag
};
