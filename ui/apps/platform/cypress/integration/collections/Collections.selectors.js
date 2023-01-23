const resultsPanel = `.pf-c-drawer__panel:has(h2:contains("Collection results"))`;
const deploymentIcon = `*[title="Deployment"]`;
const deploymentResults = `${resultsPanel} ${deploymentIcon} `;
const deploymentResult = (deploymentName) =>
    `${resultsPanel} *:has(${deploymentIcon}) *:contains("${deploymentName}")`;
const viewMoreResultsButton = `${resultsPanel} button:contains("View more")`;

/**
 * @param {'Attached'|'Available'} type
 * @param {string} collectionName
 */
const embeddedCollectionRow = (type, collectionName) =>
    `*[aria-label="${type} collections"]  tr:has(button:contains("${collectionName}"))`;
const viewEmbeddedCollectionButton = (type, collectionName) =>
    `*[aria-label="${type} collections"]  button:contains("${collectionName}")`;
const attachCollectionButton = (collectionName) =>
    `${embeddedCollectionRow('Available', collectionName)} button:contains("Attach")`;
const detachCollectionButton = (collectionName) =>
    `${embeddedCollectionRow('Attached', collectionName)} button:contains("Detach")`;

const resultsPanelFilterEntitySelect = `${resultsPanel} button[aria-label="Select an entity type to filter the results by"]`;
/**
 * @param {'Deployment'|'Namespace'|'Cluster'} entity
 */
const resultsPanelFilterEntitySelectOption = (entity) =>
    `${resultsPanel} *[role="listbox"] button:contains("${entity}")`;
const resultsPanelFilterInput = `${resultsPanel} input[aria-label="Filter by name"]`;
const resultsPanelFilterSearch = `${resultsPanel} button[aria-label="Search"]`;

export const collectionSelectors = {
    tableLinkByName: (name) => `td[data-label="Collection"] a:contains("${name}")`,
    modal: '*[role="dialog"]',
    modalClose: '*[role="dialog"] button[aria-label="Close"]',
    resultsPanel,
    deploymentResults,
    deploymentResult,
    viewMoreResultsButton,
    embeddedCollectionRow,
    viewEmbeddedCollectionButton,
    attachCollectionButton,
    detachCollectionButton,
    resultsPanelFilterEntitySelect,
    resultsPanelFilterEntitySelectOption,
    resultsPanelFilterInput,
    resultsPanelFilterSearch,
};
