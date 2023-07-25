import React, { ReactElement, useState } from 'react';
import { format } from 'date-fns';
import capitalize from 'lodash/capitalize';
import {
    Card,
    CardHeader,
    CardTitle,
    CardExpandableContent,
    CardBody,
    DescriptionList,
    Divider,
} from '@patternfly/react-core';

import DescriptionListItem from 'Components/DescriptionListItem';
import dateTimeFormat from 'constants/dateTimeFormat';

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
        <div className="pf-u-pb-md" key={message}>
            <Card isExpanded={isExpanded} id={message} isFlat>
                <CardHeader onExpand={onExpand}>
                    <CardTitle>{message}</CardTitle>
                </CardHeader>
                <CardExpandableContent>
                    <CardBody className="pf-u-mt-lg">
                        <DescriptionList isHorizontal>
                            <DescriptionListItem term="Time" desc={format(time, dateTimeFormat)} />
                            {keyValueAttrs.attrs?.length > 0 && <Divider component="div" />}
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
