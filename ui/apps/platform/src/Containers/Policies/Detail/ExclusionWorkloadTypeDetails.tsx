import type { ReactElement } from 'react';
import { DescriptionList } from '@patternfly/react-core';

import DescriptionListItem from 'Components/DescriptionListItem';
import type { PolicyWorkloadType } from 'types/policy.proto';

import { formatWorkloadTypeList } from '../policies.utils';

type ExclusionWorkloadTypeDetailsProps = {
    types: PolicyWorkloadType[];
};

function ExclusionWorkloadTypeDetails({ types }: ExclusionWorkloadTypeDetailsProps): ReactElement {
    return (
        <DescriptionList isCompact isHorizontal horizontalTermWidthModifier={{ default: '16ch' }}>
            <DescriptionListItem term="Workload type" desc={formatWorkloadTypeList(types)} />
        </DescriptionList>
    );
}

export default ExclusionWorkloadTypeDetails;
