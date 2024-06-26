import React, { ReactNode, createContext, useContext, useState } from 'react';
import { SearchFilter } from 'types/search';

export type AutocompleteContextType = {
    autocompleteContext: SearchFilter;
    setAutocompleteContext: React.Dispatch<React.SetStateAction<SearchFilter>>;
};

const AutocompleteContext = createContext<AutocompleteContextType>({
    autocompleteContext: {},
    setAutocompleteContext: () => {},
});

type AutocompleteContextProviderProps = {
    value: SearchFilter;
    children: ReactNode;
};

export function AutocompleteContextProvider({ value, children }: AutocompleteContextProviderProps) {
    const [autocompleteContext, setAutocompleteContext] = useState<SearchFilter>(() => value);

    return (
        <AutocompleteContext.Provider value={{ autocompleteContext, setAutocompleteContext }}>
            {children}
        </AutocompleteContext.Provider>
    );
}

export const useAutocompleteContext = () => useContext(AutocompleteContext);
