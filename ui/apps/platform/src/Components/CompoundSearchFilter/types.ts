import { SearchCategory } from 'services/SearchService';

// Compound search filter types

export type BaseInputType = 'autocomplete' | 'text' | 'date-picker' | 'condition-number';
export type InputType = BaseInputType | 'select';
export type SelectSearchFilterOptions = {
    options: { label: string; value: string }[];
};
export type SelectSearchFilterGroupedOptions = {
    groupOptions: { name: string; options: { label: string; value: string }[] }[];
};

type BaseSearchFilterAttribute = {
    displayName: string;
    filterChipLabel: string;
    searchTerm: string;
    inputType: BaseInputType;
};

export type SelectSearchFilterAttribute = {
    displayName: string;
    filterChipLabel: string;
    searchTerm: string;
    inputType: 'select';
    inputProps: SelectSearchFilterOptions | SelectSearchFilterGroupedOptions;
};

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
