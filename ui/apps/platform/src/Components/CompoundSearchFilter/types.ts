/* eslint-disable @typescript-eslint/no-duplicate-type-constituents */
import { SearchCategory } from 'services/SearchService';

// Compound search filter types

export type InputType = 'autocomplete' | 'text' | 'date-picker' | 'condition-number' | 'select';

export type BaseSearchFilterAttribute = {
    displayName: string;
    filterChipLabel: string;
    searchTerm: string;
    inputType: InputType;
};

export interface SelectSearchFilterAttribute extends BaseSearchFilterAttribute {
    inputType: 'select';
    inputProps: {
        options: { label: string; value: string }[];
    };
}

export type SearchFilterAttribute = BaseSearchFilterAttribute | SelectSearchFilterAttribute;

export type SearchFilterConfig = {
    displayName: string;
    searchCategory: SearchCategory;
    attributes: Record<string, SearchFilterAttribute>;
};

export type CompoundSearchFilterConfig = Record<string, SearchFilterConfig>;

// Misc

export type OnSearchPayload = {
    action: 'ADD' | 'REMOVE';
    category: string;
    value: string;
};
