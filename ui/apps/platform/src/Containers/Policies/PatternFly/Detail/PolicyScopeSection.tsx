import React from 'react';
import { Title, Grid, GridItem, Card, CardBody, List, ListItem } from '@patternfly/react-core';

import { Cluster } from 'types/cluster.proto';
import { PolicyScope, PolicyExcludedDeployment } from 'types/policy.proto';
import Restriction from './Restriction';
import ExcludedDeployment from './ExcludedDeployment';

type PolicyScopeSectionProps = {
    scope: PolicyScope[];
    excludedDeployments: PolicyExcludedDeployment[];
    excludedImageNames: string[];
    clusters: Cluster[];
};

function PolicyScopeSection({
    scope,
    excludedDeployments,
    excludedImageNames,
    clusters,
}: PolicyScopeSectionProps): React.ReactElement {
    return (
        <>
            {scope.length !== 0 && (
                <>
                    <Title headingLevel="h3" className="pf-u-pt-md pf-u-pb-sm">
                        Restrict to scopes
                    </Title>
                    <Grid hasGutter>
                        {scope.map((restriction, index) => (
                            // eslint-disable-next-line react/no-array-index-key
                            <GridItem key={index} span={4}>
                                <Card isFlat>
                                    <CardBody>
                                        <Restriction
                                            clusters={clusters}
                                            restriction={restriction}
                                        />
                                    </CardBody>
                                </Card>
                            </GridItem>
                        ))}
                    </Grid>
                </>
            )}
            {excludedDeployments.length !== 0 && (
                <>
                    <Title headingLevel="h3" className="pf-u-pt-md pf-u-pb-sm">
                        Excluded deployments
                    </Title>
                    <Grid hasGutter>
                        {excludedDeployments.map((excludedDeployment, index) => (
                            // eslint-disable-next-line react/no-array-index-key
                            <GridItem key={index} span={4}>
                                <Card isFlat>
                                    <CardBody>
                                        <ExcludedDeployment
                                            clusters={clusters}
                                            excludedDeployment={excludedDeployment}
                                        />
                                    </CardBody>
                                </Card>
                            </GridItem>
                        ))}
                    </Grid>
                </>
            )}
            {excludedImageNames.length !== 0 && (
                <>
                    <Title headingLevel="h3" className="pf-u-pt-md pf-u-pb-sm">
                        Excluded images
                    </Title>
                    <List isPlain>
                        {excludedImageNames.map((name) => (
                            <ListItem key={name}>{name}</ListItem>
                        ))}
                    </List>
                </>
            )}
        </>
    );
}

export default PolicyScopeSection;
