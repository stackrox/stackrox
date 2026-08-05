import { useState } from 'react';
import type { ReactElement } from 'react';
import {
    Card,
    CardBody,
    CardExpandableContent,
    CardHeader,
    CardTitle,
    DescriptionList,
    Label,
} from '@patternfly/react-core';

import DescriptionListItem from 'Components/DescriptionListItem';

type SecurityEventCardProps = {
    keyValueAttrs: {
        attrs: {
            key: string;
            value: string;
        }[];
    };
    message: string;
};

const severityColorMap: Record<string, 'red' | 'orange' | 'blue' | 'grey'> = {
    critical: 'red',
    high: 'red',
    medium: 'orange',
    low: 'blue',
};

function SecurityEventCard({ message, keyValueAttrs }: SecurityEventCardProps): ReactElement {
    const [isExpanded, setIsExpanded] = useState(true);

    function onExpand() {
        setIsExpanded((prev) => !prev);
    }

    const attrMap = new Map(keyValueAttrs.attrs.map(({ key, value }) => [key, value]));

    const source = attrMap.get('Source') ?? 'Unknown';
    const reportedPolicy = attrMap.get('Reported Policy') ?? '';
    const reportedRule = attrMap.get('Reported Rule') ?? '';
    const severity = attrMap.get('Severity') ?? '';
    const result = attrMap.get('Result') ?? '';

    const severityColor = severityColorMap[severity.toLowerCase()] ?? 'grey';

    return (
        <div className="pf-v6-u-pb-md">
            <Card isExpanded={isExpanded}>
                <CardHeader
                    onExpand={onExpand}
                    toggleButtonProps={{ 'aria-expanded': isExpanded, 'aria-label': 'Details' }}
                    actions={{
                        actions: (
                            <>
                                <Label color={severityColor} isCompact>
                                    {severity || 'unknown'}
                                </Label>
                                <Label color="purple" isCompact className="pf-v6-u-ml-sm">
                                    {source}
                                </Label>
                            </>
                        ),
                        hasNoOffset: true,
                    }}
                >
                    <CardTitle>{message}</CardTitle>
                </CardHeader>
                <CardExpandableContent>
                    <CardBody className="pf-v6-u-mt-lg">
                        <DescriptionList isHorizontal>
                            <DescriptionListItem term="Source" desc={source} />
                            <DescriptionListItem term="Reported policy" desc={reportedPolicy} />
                            {reportedRule && (
                                <DescriptionListItem term="Reported rule" desc={reportedRule} />
                            )}
                            <DescriptionListItem term="Result" desc={result} />
                            <DescriptionListItem term="Severity" desc={severity} />
                        </DescriptionList>
                    </CardBody>
                </CardExpandableContent>
            </Card>
        </div>
    );
}

export default SecurityEventCard;
