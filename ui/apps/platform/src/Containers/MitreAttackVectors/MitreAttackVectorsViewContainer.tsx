import React, { ReactElement } from 'react';
import { gql, useQuery } from '@apollo/client';

import { fetchMitreAttackVectors } from 'services/MitreService';
import { MitreAttackVector } from 'types/mitre.proto';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import {
    getMitreAttackVector,
    getMitreTechnique,
} from 'Containers/Policies/Wizard/Step1/mitreAttackVectors.utils';
import MitreAttackVectorsView from './MitreAttackVectorsView';

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
    policyId?: string;
    policyFormMitreAttackVectors?: {
        tactic: string;
        techniques: string[];
    }[];
};

function MitreAttackVectorsViewContainer({
    policyId,
    policyFormMitreAttackVectors,
}: MitreAttackVectorsViewProps): ReactElement {
    const {
        loading: isLoading,
        data,
        error,
    } = useQuery<GetMitreAttackVectorsData, GetMitreAttackVectorsVars>(GET_MITRE_ATTACK_VECTORS, {
        variables: {
            id: policyId || '',
        },
    });

    const [allMitreAttackVectors, setAllMitreAttackVectors] = React.useState<MitreAttackVector[]>(
        []
    );
    const [mitreAttackVectorsError, setMitreAttackVectorsError] = React.useState('');

    React.useEffect(() => {
        fetchMitreAttackVectors()
            .then((mitreAttackVectors) => {
                setAllMitreAttackVectors(mitreAttackVectors);
            })
            .catch((errorMessage) => {
                setMitreAttackVectorsError(getAxiosErrorMessage(errorMessage));
            });
        return () => {
            setAllMitreAttackVectors([]);
            setMitreAttackVectorsError('');
        };
    }, []);

    const policyFormAttackVectors = policyFormMitreAttackVectors?.map(
        ({ tactic: tacticId, techniques: techniqueIds }) => {
            const { tactic, techniques } = getMitreAttackVector(allMitreAttackVectors, tacticId);
            return {
                tactic,
                techniques: techniqueIds.map((techniqueId) =>
                    getMitreTechnique(techniques, techniqueId)
                ),
            };
        }
    );

    const policyMitreAttackVectors =
        policyFormAttackVectors || data?.policy?.mitreAttackVectors || [];

    return (
        <MitreAttackVectorsView
            policyMitreAttackVectors={policyMitreAttackVectors}
            isLoading={isLoading}
            errorMessage={error?.message || mitreAttackVectorsError}
        />
    );
}

export default MitreAttackVectorsViewContainer;
