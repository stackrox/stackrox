import React from 'react';
import { Card, CardBody, CardTitle, EmptyState, List, ListItem } from '@patternfly/react-core';

type ContainerCommandInfoProps = {
    command: string[]; // note: the k8s API, and our data of it, use singular "command" for this array
};

function ContainerCommandInfo({ command }: ContainerCommandInfoProps) {
    return (
        <Card>
            <CardTitle>Commands</CardTitle>
            {command.length > 0 ? (
                <CardBody>
                    <List isPlain>
                        {command.map((cmd) => (
                            <ListItem>{cmd}</ListItem>
                        ))}
                    </List>
                </CardBody>
            ) : (
                <CardBody>
                    <EmptyState>No commands</EmptyState>
                </CardBody>
            )}
        </Card>
    );
}

export default ContainerCommandInfo;
