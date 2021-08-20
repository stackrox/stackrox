import React, { ReactElement } from 'react';
import { useQuery, gql } from '@apollo/client';

import NoResultsMessage from 'Components/NoResultsMessage';
import MitreAttackVectorContainer from 'Components/MitreAttackVectorContainer';
import LoadingSection from 'Components/LoadingSection';

export type MitreAttackVectorsProps = {
    policyId: string;
};

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

function MitreAttackVectors({ policyId }: MitreAttackVectorsProps): ReactElement {
    const { loading: isLoading, data, error } = useQuery(GET_MITRE_ATTACK_VECTORS, {
        variables: {
            id: policyId,
        },
    });

    if (isLoading) {
        return <LoadingSection />;
    }

    if (error) {
        const message =
            'An error occurred in retrieving MITRE ATT&CK vectors. Please refresh the page. If this problem continues, please contact support.';
        return <NoResultsMessage message={message} icon="warn" />;
    }

    const mitreAttackVectors = data?.policy?.mitreAttackVectors || [];

    return (
        <div className="gap-4">
            {mitreAttackVectors.length === 0 ? (
                <NoResultsMessage message="There are no MITRE ATT&CK vectors" />
            ) : (
                mitreAttackVectors.map(({ tactic, techniques }) => {
                    return (
                        <MitreAttackVectorContainer headerText="Tactic">
                            <div className="p-3 space-y-3">
                                <div>
                                    <span className="text-base-600 font-700 mr-1">Name:</span>
                                    {tactic.name} | {tactic.id}
                                </div>
                                <div>{tactic.description}</div>
                            </div>
                            {techniques.length !== 0 && (
                                <div className="border-t border-base-300 p-4">
                                    {techniques.map((technique) => {
                                        const isSubtechnique = technique.name.includes('.');
                                        return (
                                            <MitreAttackVectorContainer
                                                headerText={
                                                    isSubtechnique ? 'Subtechnique' : 'Technique'
                                                }
                                                isLight
                                            >
                                                <div className="p-3 space-y-3">
                                                    <div>
                                                        <span className="text-base-600 font-700 mr-1">
                                                            Name:
                                                        </span>
                                                        {technique.name} | {technique.id}
                                                    </div>
                                                    <div>{technique.description}</div>
                                                </div>
                                            </MitreAttackVectorContainer>
                                        );
                                    })}
                                </div>
                            )}
                        </MitreAttackVectorContainer>
                    );
                })
            )}
        </div>
    );
}

export default MitreAttackVectors;
