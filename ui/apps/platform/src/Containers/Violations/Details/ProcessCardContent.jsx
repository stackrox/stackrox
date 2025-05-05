import React from 'react';
import PropTypes from 'prop-types';
import { DescriptionList, Flex, Divider } from '@patternfly/react-core';

import DescriptionListItem from 'Components/DescriptionListItem';
import KeyValue from 'Components/KeyValue';
import { getDateTime } from 'utils/dateUtils';

function ProcessCardContent({ process }) {
    const { time, args, execFilePath, containerId, lineage, uid } = process.signal;
    const processTime = new Date(time);
    const timeFormat = getDateTime(processTime);
    let ancestors = null;
    if (Array.isArray(lineage) && lineage.length) {
        ancestors = (
            <div className="flex flex-1 text-base-600 px-4 py-2">
                <KeyValue label="Ancestors:" value={lineage.join(', ')} />
            </div>
        );
    }

    return (
        <div label={process.id}>
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
            </DescriptionList>
            {ancestors}
        </div>
    );
}

ProcessCardContent.propTypes = {
    process: PropTypes.shape({
        id: PropTypes.string.isRequired,
        deploymentId: PropTypes.string.isRequired,
        containerName: PropTypes.string.isRequired,
        signal: PropTypes.shape({
            time: PropTypes.string.isRequired,
            args: PropTypes.string.isRequired,
            execFilePath: PropTypes.string.isRequired,
            containerId: PropTypes.string.isRequired,
            lineage: PropTypes.arrayOf(PropTypes.string).isRequired,
            uid: PropTypes.string.isRequired,
        }),
    }).isRequired,
};

export default ProcessCardContent;
