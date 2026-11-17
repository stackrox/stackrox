import type { ReactElement } from 'react';
import { DescriptionList } from '@patternfly/react-core';

import DescriptionListItem from 'Components/DescriptionListItem';
import type { PolicyWorkloadKind } from 'types/policy.proto';

import { formatWorkloadKindList } from '../policies.utils';

type ExclusionWorkloadKindDetailsProps = {
    kinds: PolicyWorkloadKind[];
};

function ExclusionWorkloadKindDetails({ kinds }: ExclusionWorkloadKindDetailsProps): ReactElement {
    return (
        <DescriptionList isCompact isHorizontal horizontalTermWidthModifier={{ default: '16ch' }}>
            <DescriptionListItem term="Workload kind" desc={formatWorkloadKindList(kinds)} />
        </DescriptionList>
    );
}

export default ExclusionWorkloadKindDetails;
