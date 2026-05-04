import type { ReactElement } from 'react';
import { Card, CardBody, CardTitle, EmptyState, List, ListItem } from '@patternfly/react-core';

type ContainerArgumentsInfoProps = {
    args: string[];
};

function ContainerArgumentsInfo({ args }: ContainerArgumentsInfoProps): ReactElement {
    return (
        <Card>
            <CardTitle>Arguments</CardTitle>
            {args.length > 0 ? (
                <CardBody>
                    <List isPlain>
                        {args.map((arg) => (
                            <ListItem key={arg}>--{arg}</ListItem>
                        ))}
                    </List>
                </CardBody>
            ) : (
                <CardBody>
                    <EmptyState>No arguments</EmptyState>
                </CardBody>
            )}
        </Card>
    );
}

export default ContainerArgumentsInfo;
