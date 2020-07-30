import scopeSelectors from '../helpers/scopeSelectors';
import tab from '../selectors/tab';
import search from '../selectors/search';

const viewOnLabelChip = '[data-testid="view-on-label-chip"]';
const filterOnLabelChip = '[data-testid="filter-on-label-chip"]';

export const selectors = {
    globalSearchButton: 'button:contains("Search")',
    pageSearchSuggestions: 'div.Select-menu-outer',
    pageSearch: scopeSelectors('[data-testid="page-header"]', {
        input: search.input,
        options: search.input,
    }),
    globalSearch: scopeSelectors('.search-modal', {
        input: search.input,
        options: search.input,
    }),
    allTab: `${tab.tabs}:contains("All")`,
    violationsTab: `${tab.tabs}:contains("Violations")`,
    policiesTab: `${tab.tabs}:contains("Policies")`,
    deploymentsTab: `${tab.tabs}:contains("Deployments")`,
    imagesTab: `${tab.tabs}:contains("Images")`,
    secretsTab: `${tab.tabs}:contains("Secrets")`,
    globalSearchResults: scopeSelectors('[data-testid="global-search-results"]', {
        header: 'h1',
    }),
    viewOnRiskLabelChip: `${viewOnLabelChip}:contains("RISK")`,
    viewOnViolationsLabelChip: `${viewOnLabelChip}:contains("VIOLATIONS")`,
    viewOnPoliciesLabelChip: `${viewOnLabelChip}:contains("POLICIES")`,
    viewOnImagesLabelChip: `${viewOnLabelChip}:contains("IMAGES")`,
    viewOnSecretsLabelChip: `${viewOnLabelChip}:contains("SECRETS")`,
    filterOnRiskLabelChip: `${filterOnLabelChip}:contains("RISK")`,
    filterOnViolationsLabelChip: `${filterOnLabelChip}:contains("VIOLATIONS")`,
    filterOnNetworkLabelChip: `${filterOnLabelChip}:contains("NETWORK")`,
};

export default selectors;
