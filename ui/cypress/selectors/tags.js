import selectSelectors from './select';
import scopeSelectors from '../helpers/scopeSelectors';

export const violationTagsSelectors = scopeSelectors(
    '[data-testid="violation-tags"]',
    selectSelectors.multiSelect
);

export const processTagsSelectors = scopeSelectors(
    '[data-testid="process-tags"]',
    selectSelectors.multiSelect
);
