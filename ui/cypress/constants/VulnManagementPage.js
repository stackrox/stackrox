export const baseURL = '/main/vulnerability-management';

export const url = {
    dashboard: baseURL,
    list: {
        policies: `${baseURL}/policies`,
        clusters: `${baseURL}/clusters`,
        namespaces: `${baseURL}/namespaces`,
        deployments: `${baseURL}/deployments`,
        images: `${baseURL}/images`,
        components: `${baseURL}/components`
    }
};
export const dashboardSelectors = {
    widgets: "[data-test-id='widget']",
    tileLinks: "[data-test-id='tile-link']",
    tileLinkValue: "[data-test-id='tile-link-value']",
    applicationAndInfrastructureDropdown: 'button:contains("Application & Infrastructure")',
    getMenuListItem: name => {
        return `[data-test-id="menu-list"] [data-test-id="${name}"]`;
    }
};
export const selectors = {
    ...dashboardSelectors
};
