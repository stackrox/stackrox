import type { ReactElement } from 'react';
import {
    Card,
    CardBody,
    Content,
    Grid,
    GridItem,
    List,
    ListItem,
    Title,
} from '@patternfly/react-core';

import useFetchClustersForPermissions from 'hooks/useFetchClustersForPermissions';
import type { PolicyExclusion, PolicyScope } from 'types/policy.proto';
import InclusionScopeDetails from './InclusionScopeDetails';
import ExclusionDeploymentDetails from './ExclusionDeploymentDetails';
import { getExcludedDeployments, getExcludedImageNames } from '../policies.utils';

type PolicyScopeSectionProps = {
    scope: PolicyScope[];
    exclusions: PolicyExclusion[];
};

function PolicyScopeSection({ scope, exclusions }: PolicyScopeSectionProps): ReactElement {
    const { clusters } = useFetchClustersForPermissions(['Deployment']);
    const excludedDeployments = getExcludedDeployments(exclusions);
    const imageExclusionNames = getExcludedImageNames(exclusions);

    const hasIncludedScope = scope?.length > 0;
    const hasExcludedDeployments = excludedDeployments.length > 0;
    const hasImageExclusions = imageExclusionNames.length > 0;
    const hasAnyResources = hasIncludedScope || hasExcludedDeployments || hasImageExclusions;

    return (
        <>
            {!hasAnyResources && <Content component="p">No policy resources.</Content>}
            {hasIncludedScope && (
                <>
                    <Title headingLevel="h3">Included resources</Title>
                    <Grid hasGutter md={12} xl={6}>
                        {scope.map((scopeItem, index) => (
                            // eslint-disable-next-line react/no-array-index-key
                            <GridItem key={index}>
                                <Card>
                                    <CardBody>
                                        <InclusionScopeDetails
                                            clusters={clusters}
                                            scope={scopeItem}
                                        />
                                    </CardBody>
                                </Card>
                            </GridItem>
                        ))}
                    </Grid>
                </>
            )}
            {hasExcludedDeployments && (
                <>
                    <Title headingLevel="h3">Excluded resources</Title>
                    <Grid hasGutter md={12} xl={6}>
                        {excludedDeployments.map((excludedDeployment, index) => (
                            // eslint-disable-next-line react/no-array-index-key
                            <GridItem key={index}>
                                <Card>
                                    <CardBody>
                                        <ExclusionDeploymentDetails
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
            {hasImageExclusions && (
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
