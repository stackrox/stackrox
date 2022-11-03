import React, { ReactElement, useCallback, useMemo, useState } from 'react';
import {
    debounce,
    Select,
    SelectOption,
    SelectOptionProps,
    ValidatedOptions,
} from '@patternfly/react-core';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import useRestQuery from 'Containers/Dashboard/hooks/useRestQuery';
import { CancellableRequest } from 'services/cancellationUtils';
import ResourceIcon from 'Components/PatternFly/ResourceIcon';
import { SelectorEntityType } from '../types';

export type AutoCompleteSelectProps = {
    id: string;
    selectedOption: string;
    className?: string;
    typeAheadAriaLabel?: string;
    onChange: (value: string) => void;
    validated: ValidatedOptions;
    isDisabled: boolean;
    autocompleteProvider?: (search: string) => CancellableRequest<string[]>;
};

function ResourceSelectOption({ option, entityType }) {
    return (
        <SelectOption
            value={option}
            component={(props) => (
                <div {...props}>
                    <ResourceIcon kind={entityType} />
                    {option}
                </div>
            )}
        />
    );
}

function getOptions(data: string[] | undefined): ReactElement[] | undefined {
    return data?.map((option) => (
        <ResourceSelectOption key={option} option={option} entityType="Deployment" />
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
}: AutoCompleteSelectProps) {
    const { isOpen, onToggle, closeSelect } = useSelectToggle();
    const [typeahead, setTypeahead] = useState(selectedOption);

    const autocompleteCallback = useCallback(() => {
        const shouldMakeRequest = isOpen && autocompleteProvider;
        if (shouldMakeRequest) {
            return autocompleteProvider(typeahead);
        }
        return {
            request: Promise.resolve([]),
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
                isOpen={isOpen}
                onFilter={() => getOptions(data)}
                onToggle={onToggle}
                onTypeaheadInputChanged={updateTypeahead}
                selections={selectedOption}
                onSelect={onSelect}
                isDisabled={isDisabled}
            >
                {getOptions(data)}
            </Select>
        </>
    );
}
