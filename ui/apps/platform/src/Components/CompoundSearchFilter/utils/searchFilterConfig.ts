import { SearchCategory } from 'services/SearchService';
import { CompoundSearchFilterConfig, SearchFilterAttribute } from '../types';

export function createSearchFilterConfig(
    configs: {
        displayName: string;
        searchCategory: SearchCategory;
        attributes: SearchFilterAttribute[];
    }[]
): CompoundSearchFilterConfig {
    const searchFilterConfig = configs.reduce((acc, config) => {
        acc[config.displayName] = {
            displayName: config.displayName,
            searchCategory: config.searchCategory,
            attributes: config.attributes.reduce((acc, curr) => {
                acc[curr.displayName] = curr;
                return acc;
            }, {}),
        };
        return acc;
    }, {});

    return searchFilterConfig;
}
