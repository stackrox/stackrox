import React, { ReactElement } from 'react';
import { Alert, Flex, FlexItem, Spinner, TreeView, TreeViewDataItem } from '@patternfly/react-core';

import { MitreAttackVector } from 'types/mitre.proto';

import MitreAttackLink from './MitreAttackLink';
import { getMitreTacticUrl, getMitreTechniqueUrl } from './MitreAttackVectors.utils';

type MitreAttackVectorsViewProps = {
    isLoading: boolean;
    errorMessage?: string;
    policyMitreAttackVectors: MitreAttackVector[];
};

function MitreAttackVectorsView({
    isLoading,
    errorMessage,
    policyMitreAttackVectors,
}: MitreAttackVectorsViewProps): ReactElement {
    if (isLoading) {
        return (
            <Flex className="pf-u-my-md" justifyContent={{ default: 'justifyContentCenter' }}>
                <FlexItem>
                    <Spinner isSVG />
                </FlexItem>
            </Flex>
        );
    }

    if (errorMessage) {
        return (
            <Alert className="pf-u-my-md" title="Request failed" variant="warning" isInline>
                {errorMessage}
            </Alert>
        );
    }

    if (policyMitreAttackVectors.length === 0) {
        return <div className="pf-u-my-md">Policy has no MITRE ATT&CK vectors</div>;
    }

    return <TreeView data={getData(policyMitreAttackVectors)} variant="compactNoBackground" />;
}

function getData(policyMitreAttackVectors: MitreAttackVector[]): TreeViewDataItem[] {
    return policyMitreAttackVectors.map(({ tactic, techniques }) => ({
        title: (
            <Flex>
                <FlexItem>{tactic.name}</FlexItem>
                <FlexItem>
                    <MitreAttackLink href={getMitreTacticUrl(tactic.id)} id={tactic.id} />
                </FlexItem>
            </Flex>
        ),
        name: tactic.description,
        id: tactic.id,
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
