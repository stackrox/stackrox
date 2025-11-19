import type { ReactElement } from 'react';
import { DescriptionList, Divider, Flex } from '@patternfly/react-core';

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
        <div>
            <Flex
                justifyContent={{ default: 'justifyContentSpaceBetween' }}
                alignItems={{ default: 'alignItemsFlexStart' }}
            >
                <Divider component="div" />
                <span className="pf-v5-u-background-color-warning pf-v5-u-px-md pf-v5-u-py-sm">
                    {execFilePath}
                </span>
            </Flex>
            <DescriptionList
                columnModifier={{
                    default: '2Col',
                }}
                className="pf-v5-u-my-md"
            >
                <DescriptionListItem term="Container ID" desc={containerId} />
                <DescriptionListItem term="Time" desc={timeFormat} />
                <DescriptionListItem term="User ID" desc={uid} />
            </DescriptionList>
            <DescriptionList className="pf-v5-u-mb-md">
                <DescriptionListItem term="Arguments" desc={args} />
                {Array.isArray(lineage) && lineage.length ? (
                    <DescriptionListItem term="Ancestors" desc={lineage.join(', ')} />
                ) : null}
            </DescriptionList>
        </div>
    );
}

export default ProcessCardContent;
