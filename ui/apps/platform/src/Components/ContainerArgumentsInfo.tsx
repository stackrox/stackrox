import React, { CSSProperties } from 'react';
import { Card, CardBody, CardTitle, EmptyState, List, ListItem } from '@patternfly/react-core';

type ContainerArgumentsInfoProps = {
    args: string[];
};

const styleConstant = {
    overflow: 'scroll',
    '--pf-u-max-height--MaxHeight': '12ch',
} as CSSProperties;

function ContainerArgumentsInfo({ args }: ContainerArgumentsInfoProps) {
    return (
        <Card>
            <CardTitle>Arguments</CardTitle>
            {args.length > 0 ? (
                <CardBody className="pf-u-background-color-200 pf-u-pt-lg pf-u-mx-lg pf-u-mb-lg">
                    <List isPlain className="pf-u-max-height" style={styleConstant}>
                        {args.map((arg) => (
                            <ListItem>--{arg}</ListItem>
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
