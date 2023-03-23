import React from 'react';
import { Card, CardBody, CardTitle, EmptyState, List, ListItem } from '@patternfly/react-core';

type ContainerCommandsInfoProps = {
    command: string[]; // note: the k8s API, and our data of it, use singular "command" for this array
};

function ContainerCommandsInfo({ command }: ContainerCommandsInfoProps) {
    return (
        <Card>
            <CardTitle>Commands</CardTitle>
            {command.length > 0 ? (
                <CardBody className="">
                    <List isPlain>
                        {command.map((arg) => (
                            <ListItem>{arg}</ListItem>
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

export default ContainerCommandsInfo;
