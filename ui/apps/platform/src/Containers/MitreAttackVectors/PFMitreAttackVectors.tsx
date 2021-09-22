import React, { ReactElement } from 'react';
import { useQuery, gql } from '@apollo/client';

import NoResultsMessage from 'Components/NoResultsMessage';
import Loader from 'Components/Loader';
import {
    Card,
    CardActions,
    CardBody,
    CardFooter,
    CardHeader,
    CardHeaderMain,
    CardTitle,
    Label,
} from '@patternfly/react-core';
import { ExternalLinkAltIcon } from '@patternfly/react-icons';

export const GET_MITRE_ATTACK_VECTORS = gql`
    query getMitreAttackVectors($id: ID!) {
        policy(id: $id) {
            mitreAttackVectors: fullMitreAttackVectors {
                tactic {
                    id
                    name
                    description
                }
                techniques {
                    id
                    name
                    description
                }
            }
        }
    }
`;

export type ExternalMitreAttackLinkProps = {
    type: 'tactic' | 'technique';
    id: string;
    name: string;
};

function ExternalMitreAttackLink({ type, id, name }: ExternalMitreAttackLinkProps): ReactElement {
    const urlType = type === 'tactic' ? 'tactics' : 'techniques';
    const urlID: string = id.includes('.') ? id.replace('.', '/') : id;
    const url = `https://attack.mitre.org/${urlType}/${urlID}/`;

    return (
        <a target="_blank" href={url} rel="noreferrer">
            <ExternalLinkAltIcon alt={`${name} link`} />
        </a>
    );
}

export type PFMitreAttackVectorsProps = {
    policyId: string;
};

function PFMitreAttackVectors({ policyId }: PFMitreAttackVectorsProps): ReactElement {
    const {
        loading: isLoading,
        data,
        error,
    } = useQuery(GET_MITRE_ATTACK_VECTORS, {
        variables: {
            id: policyId,
        },
    });

    if (isLoading) {
        return <Loader message={null} />;
    }

    if (error) {
        let message =
            'An error occurred in retrieving MITRE ATT&CK vectors. Please refresh the page. If this problem continues, please contact support.';
        if (error.message) {
            message = `${message} - ${error.message}`;
        }
        return (
            <Card>
                <CardBody>
                    <NoResultsMessage message={message} icon="warn" />
                </CardBody>
            </Card>
        );
    }

    const mitreAttackVectors = data?.policy?.mitreAttackVectors || [];

    return (
        <div className="gap-4">
            {mitreAttackVectors.length === 0 ? (
                <Card>
                    <CardBody>
                        <NoResultsMessage message="There are no MITRE ATT&CK vectors" />
                    </CardBody>
                </Card>
            ) : (
                mitreAttackVectors.map(({ tactic, techniques }) => {
                    return (
                        <Card>
                            <CardHeader>
                                <CardHeaderMain>
                                    <CardTitle>
                                        <span className="pf-u-mr-sm">
                                            {tactic.name} | {tactic.id}
                                        </span>
                                        <Label>Tactic</Label>
                                    </CardTitle>
                                </CardHeaderMain>
                                <CardActions hasNoOffset>
                                    <ExternalMitreAttackLink
                                        type="tactic"
                                        id={tactic.id}
                                        name={tactic.name}
                                    />
                                </CardActions>
                            </CardHeader>
                            <CardBody>
                                <div>{tactic.description}</div>
                            </CardBody>
                            <CardFooter>
                                {techniques.length !== 0 && (
                                    <div>
                                        {techniques.map((technique) => {
                                            const isSubtechnique = technique.name.includes('.');
                                            return (
                                                <Card>
                                                    <CardHeader>
                                                        <CardHeaderMain>
                                                            <CardTitle>
                                                                <span className="pf-u-mr-sm">
                                                                    {technique.name} |{' '}
                                                                    {technique.id}
                                                                </span>
                                                                <Label>
                                                                    {isSubtechnique
                                                                        ? 'Sub-technique'
                                                                        : 'Technique'}
                                                                </Label>
                                                            </CardTitle>
                                                        </CardHeaderMain>
                                                        <CardActions hasNoOffset>
                                                            <ExternalMitreAttackLink
                                                                type="technique"
                                                                id={technique.id}
                                                                name={technique.name}
                                                            />
                                                        </CardActions>
                                                    </CardHeader>
                                                    <CardBody>
                                                        <div>{technique.description}</div>
                                                    </CardBody>
                                                </Card>
                                            );
                                        })}
                                    </div>
                                )}
                            </CardFooter>
                        </Card>
                    );
                })
            )}
        </div>
    );
}

export default PFMitreAttackVectors;
