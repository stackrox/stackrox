import React, { ReactElement } from 'react';
import { gql, useQuery } from '@apollo/client';
import { Alert, Flex, FlexItem, Spinner, TreeView, TreeViewDataItem } from '@patternfly/react-core';

import { MitreAttackVector } from 'types/mitre.proto';

import MitreAttackLink from './MitreAttackLink';
import { getMitreTacticUrl, getMitreTechniqueUrl } from './MitreAttackVectors.utils';

const GET_MITRE_ATTACK_VECTORS = gql`
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

type GetMitreAttackVectorsData = {
    policy: {
        mitreAttackVectors: MitreAttackVector[];
    };
};

type GetMitreAttackVectorsVars = {
    id: string;
};

type MitreAttackVectorsViewProps = {
    policyId: string;
};

function MitreAttackVectorsView({ policyId }: MitreAttackVectorsViewProps): ReactElement {
    const {
        loading: isLoading,
        data,
        error,
    } = useQuery<GetMitreAttackVectorsData, GetMitreAttackVectorsVars>(GET_MITRE_ATTACK_VECTORS, {
        variables: {
            id: policyId,
        },
    });

    if (isLoading) {
        return (
            <Flex className="pf-u-my-md" justifyContent={{ default: 'justifyContentCenter' }}>
                <FlexItem>
                    <Spinner isSVG />
                </FlexItem>
            </Flex>
        );
    }

    if (error) {
        return (
            <Alert className="pf-u-my-md" title="Request failed" variant="warning" isInline>
                {error.message}
            </Alert>
        );
    }

    const mitreAttackVectors = data?.policy?.mitreAttackVectors || [];

    if (mitreAttackVectors.length === 0) {
        return <div className="pf-u-my-md">Policy has no MITRE ATT&CK vectors</div>;
    }

    return <TreeView data={getData(mitreAttackVectors)} variant="compactNoBackground" />;
}

function getData(mitreAttackVectors: MitreAttackVector[]): TreeViewDataItem[] {
    return mitreAttackVectors.map(({ tactic, techniques }) => ({
        title: (
            <Flex>
                <FlexItem>{tactic.name}</FlexItem>
                <FlexItem>
                    <MitreAttackLink href={getMitreTacticUrl(tactic.id)} id={tactic.id} />
                </FlexItem>
            </Flex>
        ),
        name: tactic.description,
        children:
            techniques.length === 0
                ? undefined // avoid unneeded toggle icon for empty array
                : techniques.map((technique) => ({
                      title: (
                          <Flex>
                              <FlexItem>{technique.name}</FlexItem>
                              <FlexItem>
                                  <MitreAttackLink
                                      href={getMitreTechniqueUrl(technique.id)}
                                      id={technique.id}
                                  />
                              </FlexItem>
                          </Flex>
                      ),
                      name: technique.description,
                  })),
    }));
}

export default MitreAttackVectorsView;
