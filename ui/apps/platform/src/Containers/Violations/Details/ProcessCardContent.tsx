import type { ReactElement } from 'react';
import { DescriptionList, Divider, Flex, Title } from '@patternfly/react-core';

import DescriptionListItem from 'Components/DescriptionListItem';
import { getDateTime } from 'utils/dateUtils';
import type { ProcessIndicator } from 'types/processIndicator.proto';

type ProcessCardContentProps = {
    event: ProcessIndicator;
};

function ProcessCardContent({ event }: ProcessCardContentProps): ReactElement {
    const { time, args, execFilePath, containerId, lineage, uid } = event.signal;
    const timeFormat = time ? getDateTime(new Date(time)) : 'N/A';

    return (
        <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsMd' }}>
            <Divider component="div" />
            <Title headingLevel="h2" className="pf-v6-u-pb-sm">
                {execFilePath}
            </Title>
            <DescriptionList
                columnModifier={{
                    default: '2Col',
                }}
            >
                <DescriptionListItem term="Container ID" desc={containerId} />
                <DescriptionListItem term="Time" desc={timeFormat} />
                <DescriptionListItem term="User ID" desc={uid} />
            </DescriptionList>
            <DescriptionList>
                <DescriptionListItem term="Arguments" desc={args} />
                {Array.isArray(lineage) && lineage.length ? (
                    <DescriptionListItem term="Ancestors" desc={lineage.join(', ')} />
                ) : null}
            </DescriptionList>
        </Flex>
    );
}

export default ProcessCardContent;
