// HACK: We add some client only fields to the policy objects, since for the whitelist, the layout in the UI
// does not correspond to the layout on the backend. All such field names are explicitly listed below for clarity.
// These fields have to all be derived from the server format in preFormatWhitelistField, and
// translated back into the server format in postFormatWhitelistField.
// eslint-disable-next-line import/prefer-default-export
export const clientOnlyWhitelistFieldNames = {
    WHITELISTED_IMAGE_NAMES: 'whitelistedImageNames',
    WHITELISTED_DEPLOYMENT_SCOPES: 'whitelistedDeploymentScopes'
};
