/* eslint-disable @typescript-eslint/no-duplicate-type-constituents */
import { SearchCategory } from 'services/SearchService';

// Compound search filter types

export type InputType = 'autocomplete' | 'text' | 'date-picker' | 'condition-number' | 'select';

type BaseSearchFilterAttribute = {
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

export type CompoundSearchFilterAttribute = BaseSearchFilterAttribute | SelectSearchFilterAttribute;

export type CompoundSearchFilterEntity = {
    displayName: string;
    searchCategory: SearchCategory;
    attributes: CompoundSearchFilterAttribute[];
};

export type CompoundSearchFilterConfig = CompoundSearchFilterEntity[];

// Misc

export type OnSearchCallback = (payload: OnSearchPayload) => void;

export type OnSearchPayload = {
    action: 'ADD' | 'REMOVE';
    category: string;
    value: string;
};
