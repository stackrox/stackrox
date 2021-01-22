// HACK: We add some client only fields to the policy objects, since for the whitelist, the layout in the UI
// does not correspond to the layout on the backend. All such field names are explicitly listed below for clarity.
// These fields have to all be derived from the server format in preFormatExclusionField, and
// translated back into the server format in postFormatExclusionField.
// eslint-disable-next-line import/prefer-default-export
export const clientOnlyExclusionFieldNames = {
    EXCLUDED_IMAGE_NAMES: 'whitelistedImageNames',
    EXCLUDED_DEPLOYMENT_SCOPES: 'whitelistedDeploymentScopes',
};
