import React, {
    ForwardedRef,
    forwardRef,
    ReactElement,
    ReactNode,
    useCallback,
    useMemo,
    useState,
} from 'react';
import { debounce, Select, SelectOption, ValidatedOptions } from '@patternfly/react-core';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import useRestQuery from 'Containers/Dashboard/hooks/useRestQuery';
import { getCollectionAutoComplete } from 'services/CollectionsService';
import ResourceIcon from 'Components/PatternFly/ResourceIcon';
import { generateRequest } from '../converter';
import { ClientCollection, SelectorEntityType, SelectorField } from '../types';

export type AutoCompleteSelectProps = {
    id: string;
    collection: ClientCollection;
    entityType: SelectorEntityType;
    autocompleteField: SelectorField;
    selectedOption: string;
    className?: string;
    typeAheadAriaLabel?: string;
    onChange: (value: string) => void;
    placeholder: string;
    validated: ValidatedOptions;
    isDisabled: boolean;
};

function getOptions(
    OptionComponent: ReactNode,
    data: string[] | undefined
): ReactElement[] | undefined {
    return data?.map((value) => (
        <SelectOption key={value} value={value} component={OptionComponent} />
    ));
}

export function AutoCompleteSelect({
    id,
    collection,
    entityType,
    autocompleteField,
    selectedOption,
    className = '',
    typeAheadAriaLabel,
    onChange,
    placeholder,
    validated,
    isDisabled,
}: AutoCompleteSelectProps) {
    const { isOpen, onToggle, closeSelect } = useSelectToggle();
    const [typeahead, setTypeahead] = useState(selectedOption);

    // When isTyping is true, autocomplete results will not be displayed. This prevents
    // a clunky UX where the dropdown results and the user text get out of sync.
    const [isTyping, setIsTyping] = useState(false);

    // We need to wrap this custom SelectOption component in a forward ref
    // because PatternFly will pass a `ref` to it
    const OptionComponent = forwardRef(
        (
            props: {
                className: string;
                children: ReactNode;
                onClick: (...args: unknown[]) => void;
            },
            ref: ForwardedRef<HTMLButtonElement | null>
        ) => (
            <button className={props.className} onClick={props.onClick} type="button" ref={ref}>
                <ResourceIcon kind={entityType} />
                {props.children}
            </button>
        )
    );

    const autocompleteCallback = useCallback(() => {
        const shouldMakeRequest = isOpen;
        if (shouldMakeRequest) {
            const req = generateRequest(collection);
            const { request, cancel } = getCollectionAutoComplete(
                req.resourceSelectors,
                autocompleteField,
                typeahead
            );
            request.finally(() => setIsTyping(false));
            return { request, cancel };
        }
        return {
            request: new Promise<string[]>((resolve) => {
                setIsTyping(false);
                resolve([]);
            }),
            cancel: () => {},
        };
    }, [autocompleteField, collection, isOpen, typeahead]);

    const { data } = useRestQuery(autocompleteCallback);

    function onSelect(_, value) {
        onChange(value);
        closeSelect();
    }

    // Debounce the autocomplete requests to not overload the backend
    const updateTypeahead = useMemo(
        () => debounce((value: string) => setTypeahead(value), 800),
        []
    );

    return (
        <>
            <Select
                toggleId={id}
                validated={validated}
                typeAheadAriaLabel={typeAheadAriaLabel}
                className={className}
                placeholderText={placeholder}
                variant="typeahead"
                isCreatable
                createText="Add"
                isOpen={isOpen}
                onFilter={() => getOptions(OptionComponent, isTyping ? [] : data)}
                onToggle={onToggle}
                onTypeaheadInputChanged={(val: string) => {
                    setIsTyping(true);
                    updateTypeahead(val);
                }}
                onBlur={() => updateTypeahead(selectedOption)}
                selections={selectedOption}
                onSelect={onSelect}
                isDisabled={isDisabled}
            >
                {getOptions(OptionComponent, isTyping ? [] : data)}
            </Select>
        </>
    );
}
