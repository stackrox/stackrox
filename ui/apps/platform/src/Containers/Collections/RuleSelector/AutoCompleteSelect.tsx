import React, { ReactElement, ReactNode, useCallback, useMemo, useState } from 'react';
import { debounce, Select, SelectOption, ValidatedOptions } from '@patternfly/react-core';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import useRestQuery from 'Containers/Dashboard/hooks/useRestQuery';
import { CancellableRequest } from 'services/cancellationUtils';

export type AutoCompleteSelectProps = {
    id: string;
    selectedOption: string;
    className?: string;
    typeAheadAriaLabel?: string;
    onChange: (value: string) => void;
    validated: ValidatedOptions;
    isDisabled: boolean;
    autocompleteProvider?: (search: string) => CancellableRequest<string[]>;
    OptionComponent?: ReactNode;
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
    selectedOption,
    className = '',
    typeAheadAriaLabel,
    onChange,
    validated,
    isDisabled,
    autocompleteProvider,
    OptionComponent = SelectOption,
}: AutoCompleteSelectProps) {
    const { isOpen, onToggle, closeSelect } = useSelectToggle();
    const [typeahead, setTypeahead] = useState(selectedOption);
    // When isTyping is true, autocomplete results will not be displayed. This prevents
    // a clunky UX where the dropdown results and the user text get out of sync.
    const [isTyping, setIsTyping] = useState(false);

    const autocompleteCallback = useCallback(() => {
        const shouldMakeRequest = isOpen && autocompleteProvider;
        if (shouldMakeRequest) {
            const { request, cancel } = autocompleteProvider(typeahead);
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
    }, [isOpen, autocompleteProvider, typeahead]);

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
