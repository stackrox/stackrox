import { SearchCategory } from 'services/SearchService';
import { FeatureFlagEnvVar } from 'types/featureFlag';
import { ConditionTextInputProps } from './components/ConditionText';

// Compound search filter types

export type BaseInputType = 'autocomplete' | 'text' | 'date-picker' | 'condition-number';
export type InputType = BaseInputType | 'condition-text' | 'select';
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
    inputType: InputType;
    featureFlagDependency?: FeatureFlagEnvVar[];
};

export type GenericSearchFilterAttribute = {
    inputType: BaseInputType;
} & BaseSearchFilterAttribute;

export type ConditionTextFilterAttribute = {
    inputType: 'condition-text';
    inputProps: ConditionTextInputProps;
} & BaseSearchFilterAttribute;

export type SelectSearchFilterAttribute = {
    inputType: 'select';
    inputProps: SelectSearchFilterOptions | SelectSearchFilterGroupedOptions;
} & BaseSearchFilterAttribute;

export type CompoundSearchFilterAttribute =
    | ConditionTextFilterAttribute
    | GenericSearchFilterAttribute
    | SelectSearchFilterAttribute;

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
