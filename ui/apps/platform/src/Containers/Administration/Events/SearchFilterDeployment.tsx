import { useState } from 'react';
import { SearchInput } from '@patternfly/react-core';

type SearchFilterDeploymentProps = {
    isDisabled: boolean;
    deployment: string | undefined;
    setDeployment: (deployment: string | undefined) => void;
};

function SearchFilterDeployment({
    isDisabled,
    deployment,
    setDeployment,
}: SearchFilterDeploymentProps) {
    const [inputValue, setInputValue] = useState(deployment ?? '');

    return (
        <SearchInput
            aria-label="Deployment filter"
            placeholder="Filter by deployment"
            value={inputValue}
            onChange={(_event, value) => setInputValue(value)}
            onSearch={() => setDeployment(inputValue || undefined)}
            onClear={() => {
                setInputValue('');
                setDeployment(undefined);
            }}
            isDisabled={isDisabled}
        />
    );
}

export default SearchFilterDeployment;
