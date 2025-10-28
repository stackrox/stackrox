import React from 'react';
import type { ReactElement } from 'react';
import { Title, Grid, GridItem, Card, CardBody, List, ListItem } from '@patternfly/react-core';

import useFetchClustersForPermissions from 'hooks/useFetchClustersForPermissions';
import type { PolicyScope, PolicyExclusion } from 'types/policy.proto';
import Restriction from './Restriction';
import ExcludedDeployment from './ExcludedDeployment';
import { getExcludedDeployments, getExcludedImageNames } from '../policies.utils';

type PolicyScopeSectionProps = {
    scope: PolicyScope[];
    exclusions: PolicyExclusion[];
};

function PolicyScopeSection({ scope, exclusions }: PolicyScopeSectionProps): ReactElement {
    const { clusters } = useFetchClustersForPermissions(['Deployment']);

    const excludedDeploymentScopes = getExcludedDeployments(exclusions);
    const excludedImageNames = getExcludedImageNames(exclusions);
    return (
        <>
            {scope?.length !== 0 && (
                <>
                    <Title headingLevel="h3" className="pf-v5-u-pt-md pf-v5-u-pb-sm">
                        Scope inclusions
                    </Title>
                    <Grid hasGutter md={12} xl={6}>
                        {scope.map((restriction, index) => (
                            // eslint-disable-next-line react/no-array-index-key
                            <GridItem key={index}>
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
            {excludedDeploymentScopes?.length !== 0 && (
                <>
                    <Title headingLevel="h3" className="pf-v5-u-pt-md pf-v5-u-pb-sm">
                        Scope exclusions
                    </Title>
                    <Grid hasGutter md={12} xl={6}>
                        {excludedDeploymentScopes.map((excludedDeployment, index) => (
                            // eslint-disable-next-line react/no-array-index-key
                            <GridItem key={index}>
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
            {excludedImageNames?.length !== 0 && (
                <>
                    <Title headingLevel="h3" className="pf-v5-u-pt-md pf-v5-u-pb-sm">
                        Image exclusions
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
