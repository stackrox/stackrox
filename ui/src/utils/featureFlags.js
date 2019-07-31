export const types = {
    SHOW_DISALLOWED_CONNECTIONS: 'SHOW_DISALLOWED_CONNECTIONS',
    SHOW_CONFIG_MANAGEMENT: 'SHOW_CONFIG_MANAGEMENT'
};

const featureFlags = {
    [types.SHOW_DISALLOWED_CONNECTIONS]: false,
    [types.SHOW_CONFIG_MANAGEMENT]: process.env.REACT_APP_CONFIG_MANAGEMENT_ENABLED === 'true'
};

export default featureFlags;
