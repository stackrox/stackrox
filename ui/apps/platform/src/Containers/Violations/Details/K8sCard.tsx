import React, { ReactElement, useState } from 'react';
import capitalize from 'lodash/capitalize';
import {
    Card,
    CardHeader,
    CardTitle,
    CardExpandableContent,
    CardBody,
    DescriptionList,
} from '@patternfly/react-core';

import DescriptionListItem from 'Components/DescriptionListItem';
import { getDateTime } from 'utils/dateUtils';

type K8sCardProps = {
    keyValueAttrs?: {
        attrs: {
            key: string;
            value: string;
        }[];
    };
    message: string;
    time: string;
};

function K8sCard({ message, keyValueAttrs = { attrs: [] }, time }: K8sCardProps): ReactElement {
    const [isExpanded, setIsExpanded] = useState(true);

    function onExpand() {
        setIsExpanded(!isExpanded);
    }

    return (
        <div className="pf-v5-u-pb-md">
            <Card isExpanded={isExpanded} isFlat>
                <CardHeader
                    onExpand={onExpand}
                    toggleButtonProps={{ 'aria-expanded': isExpanded, 'aria-label': 'Details' }}
                >
                    <CardTitle>{message}</CardTitle>
                </CardHeader>
                <CardExpandableContent>
                    <CardBody className="pf-v5-u-mt-lg">
                        <DescriptionList isHorizontal>
                            <DescriptionListItem term="Time" desc={getDateTime(time)} />
                            {keyValueAttrs.attrs.map(({ key, value }) => (
                                <DescriptionListItem
                                    term={capitalize(key)}
                                    desc={value}
                                    key={key}
                                />
                            ))}
                        </DescriptionList>
                    </CardBody>
                </CardExpandableContent>
            </Card>
        </div>
    );
}

export default K8sCard;
