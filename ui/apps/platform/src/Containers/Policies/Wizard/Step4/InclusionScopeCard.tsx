import type { ReactElement } from 'react';

import PolicyScopeCardBase from './PolicyScopeCardBase';

type InclusionScopeCardProps = {
    onDelete: () => void;
};

function InclusionScopeCard({ onDelete }: InclusionScopeCardProps): ReactElement {
    return (
        <PolicyScopeCardBase title="Inclusion scope" onDelete={onDelete}>
            placeholder for inclusion scope form fields
        </PolicyScopeCardBase>
    );
}

export default InclusionScopeCard;
