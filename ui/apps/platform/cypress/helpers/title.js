/*
 * Return regular expression to assert page title independent of product branding in testing environment.
 */
export function getRegExpForTitleWithBranding(title) {
    return new RegExp(`^${title} | (Red Hat Advanced Cluster Security|StackRox)$`);
}
