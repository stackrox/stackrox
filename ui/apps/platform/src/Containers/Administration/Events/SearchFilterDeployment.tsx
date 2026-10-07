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
    return (
        <SearchInput
            aria-label="Deployment filter"
            placeholder="Filter by deployment"
            value={deployment ?? ''}
            onChange={(_event, value) => setDeployment(value || undefined)}
            onClear={() => setDeployment(undefined)}
            isDisabled={isDisabled}
        />
    );
}

export default SearchFilterDeployment;
