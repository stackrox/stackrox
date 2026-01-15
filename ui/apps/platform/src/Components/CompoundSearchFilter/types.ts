import type { SearchCategory } from 'services/SearchService';
import type { FeatureFlagEnvVar } from 'types/featureFlag';
import type { NonEmptyArray } from 'utils/type.utils';
import type { ConditionTextInputProps } from './components/SearchFilterConditionText';

// Compound search filter types

export type BaseInputType = 'autocomplete' | 'text' | 'date-picker' | 'condition-number';
export type InputType =
    | BaseInputType
    | 'condition-text'
    | 'select'
    | 'select-exclusive-double'
    | 'select-exclusive-single'
    | 'unspecified';

export type SelectSearchFilterOption = {
    label: string;
    value: string;
};
export type SelectSearchFilterOptions = {
    options: SelectSearchFilterOption[];
};

export type SelectSearchFilterGroupedOption = {
    name: string;
} & SelectSearchFilterOptions;
export type SelectSearchFilterGroupedOptions = {
    groupOptions: SelectSearchFilterGroupedOption[];
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

export type SelectExclusiveSingleSearchFilterAttribute = {
    inputType: 'select-exclusive-single';
    inputProps: SelectSearchFilterOptions;
} & BaseSearchFilterAttribute;

export type SelectExclusiveDoubleSearchFilterAttribute = {
    inputType: 'select-exclusive-double';
    inputProps: SelectExclusiveDoubleSearchFilterInputProps;
} & BaseSearchFilterAttribute;

export type SelectExclusiveDoubleSearchFilterInputProps = {
    category2: string;
    options: NonEmptyArray<SelectExclusiveDoubleSearchFilterOption>;
};

export type SelectExclusiveDoubleSearchFilterOption = {
    category: string;
} & SelectSearchFilterOption;

// Only for certain attributes in view-based report.
// For example, Image CVE discovered time: All time
export type UnspecifiedSearchFilterAttribute = {
    inputType: 'unspecified';
    label: string;
} & BaseSearchFilterAttribute;

export type CompoundSearchFilterAttribute =
    | ConditionTextFilterAttribute
    | GenericSearchFilterAttribute
    | SelectSearchFilterAttribute
    | SelectExclusiveDoubleSearchFilterAttribute
    | SelectExclusiveSingleSearchFilterAttribute
    | UnspecifiedSearchFilterAttribute;

export type CompoundSearchFilterEntity = {
    displayName: string;
    searchCategory: SearchCategory;
    attributes: CompoundSearchFilterAttribute[];
};

export type CompoundSearchFilterConfig = CompoundSearchFilterEntity[];

// Compound search filter interaction types

export type OnSearchCallback = (payload: OnSearchPayload) => void;

export type OnSearchPayload = NonEmptyArray<OnSearchPayloadItem>;

export function isOnSearchPayload(payload: OnSearchPayloadItem[]): payload is OnSearchPayload {
    return payload.length !== 0;
}

export type OnSearchPayloadItem =
    | OnSearchPayloadItemAdd
    | OnSearchPayloadItemDelete
    | OnSearchPayloadItemRemove;

export type OnSearchPayloadItemAdd =
    | OnSearchPayloadItemAppend
    | OnSearchPayloadItemSelectInclusive
    | OnSearchPayloadItemSelectExclusive;

export type OnSearchPayloadItemAppend = {
    action: 'APPEND'; // inputType: autocomplete, and so on
} & OnSearchPayloadItemWithValue;

export type OnSearchPayloadItemSelectInclusive = {
    action: 'SELECT_INCLUSIVE'; // inputType: select
} & OnSearchPayloadItemWithValue;

export type OnSearchPayloadItemSelectExclusive = {
    action: 'SELECT_EXCLUSIVE'; // inputType: select-single
} & OnSearchPayloadItemWithValue;

export type OnSearchPayloadItemDelete = {
    action: 'DELETE';
} & OnSearchPayloadItemWithoutValue;

export type OnSearchPayloadItemRemove = {
    action: 'REMOVE';
} & OnSearchPayloadItemWithValue;

export type OnSearchPayloadItemWithValue = {
    value: string;
} & OnSearchPayloadItemWithoutValue;

export type OnSearchPayloadItemWithoutValue = {
    action: string;
    category: string;
};
