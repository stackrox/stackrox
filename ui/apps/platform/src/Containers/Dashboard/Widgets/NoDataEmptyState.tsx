import { EmptyState, EmptyStateFooter } from '@patternfly/react-core';
import { SearchIcon } from '@patternfly/react-icons';

function NoDataEmptyState() {
    return (
        <EmptyState icon={SearchIcon} className="pf-v6-u-h-100" variant="xs">
            <EmptyStateFooter>
                <div>No data was found in the selected resources.</div>
            </EmptyStateFooter>
        </EmptyState>
    );
}

export default NoDataEmptyState;
