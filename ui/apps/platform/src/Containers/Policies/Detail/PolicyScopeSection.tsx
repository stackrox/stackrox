import type { ReactElement } from 'react';
import { Card, CardBody, Grid, GridItem, List, ListItem, Title } from '@patternfly/react-core';

import useFetchClustersForPermissions from 'hooks/useFetchClustersForPermissions';
import type { PolicyExcludedDeployment, PolicyExclusion, PolicyScope } from 'types/policy.proto';
import Restriction from './Restriction';
import ExcludedDeployment from './ExcludedDeployment';
import { getExcludedDeployments, getExcludedImageNames } from '../policies.utils';

type PolicyScopeSectionProps = {
    scope: PolicyScope[];
    exclusions: PolicyExclusion[];
    excludedDeploymentScopes?: PolicyExcludedDeployment[];
    excludedImageNames?: string[];
};

function PolicyScopeSection({
    scope,
    exclusions,
    excludedDeploymentScopes = [],
    excludedImageNames = [],
}: PolicyScopeSectionProps): ReactElement {
    const { clusters } = useFetchClustersForPermissions(['Deployment']);

    const fromExclusionsDeployments = getExcludedDeployments(exclusions);
    const excludedDeployments =
        fromExclusionsDeployments.length !== 0
            ? fromExclusionsDeployments
            : excludedDeploymentScopes.filter((d) => d.name || d.scope);

    const fromExclusionsImageNames = getExcludedImageNames(exclusions);
    const imageExclusionNames =
        fromExclusionsImageNames.length !== 0
            ? fromExclusionsImageNames
            : excludedImageNames.filter((name) => name !== '');

    return (
        <>
            {scope?.length !== 0 && (
                <>
                    <Title headingLevel="h3">Included resources</Title>
                    <Grid hasGutter md={12} xl={6}>
                        {scope.map((restriction, index) => (
                            // eslint-disable-next-line react/no-array-index-key
                            <GridItem key={index}>
                                <Card>
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
            {excludedDeployments?.length !== 0 && (
                <>
                    <Title headingLevel="h3">Excluded resources</Title>
                    <Grid hasGutter md={12} xl={6}>
                        {excludedDeployments.map((excludedDeployment, index) => (
                            // eslint-disable-next-line react/no-array-index-key
                            <GridItem key={index}>
                                <Card>
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
            {imageExclusionNames?.length !== 0 && (
                <>
                    <Title headingLevel="h3">Image exclusions</Title>
                    <List isPlain>
                        {imageExclusionNames.map((name) => (
                            <ListItem key={name}>{name}</ListItem>
                        ))}
                    </List>
                </>
            )}
        </>
    );
}

export default PolicyScopeSection;
