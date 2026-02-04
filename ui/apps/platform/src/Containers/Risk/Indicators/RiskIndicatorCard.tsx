import { useState } from 'react';
import {
    Card,
    CardBody,
    CardExpandableContent,
    CardHeader,
    CardTitle,
    List,
    ListItem,
} from '@patternfly/react-core';

import type { RiskResult } from 'services/DeploymentsService';

type RiskIndicatorCardProps = {
    result: RiskResult;
};

function RiskIndicatorCard({ result }: RiskIndicatorCardProps) {
    const [isExpanded, setIsExpanded] = useState(true);

    function onExpand() {
        setIsExpanded((prev) => !prev);
    }

    return (
        <Card isExpanded={isExpanded}>
            <CardHeader
                onExpand={onExpand}
                toggleButtonProps={{ 'aria-expanded': isExpanded, 'aria-label': 'Details' }}
            >
                <CardTitle>{result.name}</CardTitle>
            </CardHeader>
            <CardExpandableContent>
                <CardBody>
                    <List isPlain isBordered>
                        {result.factors.map(({ message, url }, index) => (
                            // TODO is the link external or internal?
                            /* eslint-disable generic/ExternalLink-anchor */
                            // eslint-disable-next-line react/no-array-index-key
                            <ListItem key={index}>
                                {url ? (
                                    <a href={url} target="_blank" rel="noopener noreferrer">
                                        {message}
                                    </a>
                                ) : (
                                    message
                                )}
                            </ListItem>
                            /* eslint-enable generic/ExternalLink-anchor */
                        ))}
                    </List>
                </CardBody>
            </CardExpandableContent>
        </Card>
    );
}

export default RiskIndicatorCard;
