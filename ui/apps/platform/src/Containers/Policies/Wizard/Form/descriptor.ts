// A form descriptor for every option (key) on the policy criteria form page.
/*
    e.g.
    {
        label: 'Image Tag',
        name: 'Image Tag',
        negatedName: `Image tag doesn't match`,
        category: policyCriteriaCategories.IMAGE_REGISTRY,
        type: 'text',
        placeholder: 'latest',
        canBooleanLogic: true,
    },

    label: for legacy policy alert labels
    name: the string used to display UI and send to backend
    negatedName: string used to display UI when negated
        (if this does not exist, the UI assumes that the field cannot be negated)
    longName: string displayed in the UI in the Policy Field Card (not in draggable key)
    category: the category grouping for the policy criteria (collapsible group in keys)
    type: the type of form field to render when dragged to the Policy Field Card
    subComponents: subfields the field renders when dragged to Policy Field Card if 'group' type
    radioButtons: button options if 'radio' type
    options: options if 'select' or 'multiselect' or 'multiselect-creatable' type
    placeholder: string to display as placeholder if applicable
    canBooleanLogic: indicates whether the field supports the AND/OR boolean operator
        (UI assumes false by default)
    defaultValue: the default value to set, if provided
    disabled: disables the field entirely
    reverse: will reverse boolean value on store
 */

export type DescriptorOption = {
    label: string;
    value: string;
};

export type SubComponent = {
    type: 'number' | 'select' | 'text'; // add more if needed
    options?: DescriptorOption[];
    subpath: string;
    placeholder?: string;
    label?: string;
    min?: number;
    max?: number;
    step?: number;
};

export type BaseDescriptor = {
    label?: string;
    name: string;
    longName?: string;
    shortName?: string;
    negatedName?: string;
    category: string;
    type: DescriptorType;
    canBooleanLogic?: boolean;
    disabled?: boolean;
};

export type DescriptorType = 'group' | 'multiselect' | 'number' | 'radioGroup' | 'select' | 'text';

export type Descriptor =
    | GroupDescriptor
    | NumberDescriptor
    | RadioGroupDescriptor
    | SelectDescriptor
    | TextDescriptor;

export type GroupDescriptor = {
    type: 'group';
    subComponents: SubComponent[];
    default?: boolean;
} & BaseDescriptor;

export type NumberDescriptor = {
    type: 'number';
    placeholder?: string;
} & BaseDescriptor;

export type RadioGroupDescriptor = {
    type: 'radioGroup';
    radioButtons: { text: string; value: string | boolean }[];
    defaultValue?: string | boolean;
    reverse?: boolean;
} & BaseDescriptor;

export type SelectDescriptor = {
    type: 'multiselect' | 'select';
    options: DescriptorOption[];
    placeholder?: string;
    reverse?: boolean;
} & BaseDescriptor;

export type TextDescriptor = {
    type: 'text';
    placeholder?: string;
} & BaseDescriptor;
