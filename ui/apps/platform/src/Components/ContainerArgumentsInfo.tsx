import type { CSSProperties, ReactElement } from 'react';
import { Card, CardBody, CardTitle, EmptyState, List, ListItem } from '@patternfly/react-core';

type ContainerArgumentsInfoProps = {
    args: string[];
};

const styleConstant = {
    overflow: 'scroll',
    '--pf-v5-u-max-height--MaxHeight': '12ch',
} as CSSProperties;

function ContainerArgumentsInfo({ args }: ContainerArgumentsInfoProps): ReactElement {
    return (
        <Card>
            <CardTitle>Arguments</CardTitle>
            {args.length > 0 ? (
                <CardBody className="pf-v6-u-background-color-200 pf-v6-u-pt-lg pf-v6-u-mx-lg pf-v6-u-mb-lg">
                    <List isPlain className="pf-v6-u-max-height" style={styleConstant}>
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
