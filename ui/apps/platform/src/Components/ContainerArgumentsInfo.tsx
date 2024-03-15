import React, { CSSProperties } from 'react';
import { Card, CardBody, CardTitle, EmptyState, List, ListItem } from '@patternfly/react-core';

type ContainerArgumentsInfoProps = {
    args: string[];
};

const styleConstant = {
    overflow: 'scroll',
    '--pf-v5-u-max-height--MaxHeight': '12ch',
} as CSSProperties;

function ContainerArgumentsInfo({ args }: ContainerArgumentsInfoProps) {
    return (
        <Card>
            <CardTitle>Arguments</CardTitle>
            {args.length > 0 ? (
                <CardBody className="pf-v5-u-background-color-200 pf-v5-u-pt-lg pf-v5-u-mx-lg pf-v5-u-mb-lg">
                    <List isPlain className="pf-v5-u-max-height" style={styleConstant}>
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
