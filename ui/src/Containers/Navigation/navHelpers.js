import { isBackendFeatureFlagEnabled } from 'utils/featureFlags';

export function filterLinksByFeatureFlag(flagsToUse, navLinks, defaultVal = false) {
    return navLinks.filter(navLink => {
        if (!navLink.featureFlag) return true;
        return isBackendFeatureFlagEnabled(flagsToUse, navLink.featureFlag, defaultVal);
    });
}

export const getDarkModeLinkClassName = isDarkMode =>
    isDarkMode ? 'hover:bg-primary-100' : 'border-primary-900 hover:bg-base-700';

export default {
    filterLinksByFeatureFlag,
    getDarkModeLinkClassName
};
