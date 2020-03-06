import { isBackendFeatureFlagEnabled } from 'utils/featureFlags';

export function filterLinksByFeatureFlag(flagsToUse, navLinks, defaultVal = false) {
    return navLinks.filter(navLink => {
        if (!navLink.featureFlag) return true;
        return isBackendFeatureFlagEnabled(flagsToUse, navLink.featureFlag, defaultVal);
    });
}

export const getDarkModeLinkClassName = isDarkMode =>
    isDarkMode ? 'hover:bg-base-200 border-base-400' : 'border-primary-900 hover:bg-base-700';

export default {
    filterLinksByFeatureFlag,
    getDarkModeLinkClassName
};
