import axios from './instance';

const mitreAttackVectorsURL = '/v1/mitreattackvectors';

// These will just be ids in string format
export type MitreAttackVectorId = {
    tactic: string;
    techniques: string[];
};

export type MitreTactic = {
    id: string;
    name: string;
    description: string;
};

export type MitreTechnique = {
    id: string;
    name: string;
    description: string;
};

export type MitreAttackVector = {
    tactic: MitreTactic;
    techniques: MitreTechnique[];
};

export function fetchMitreAttackVectors(): Promise<MitreAttackVector[]> {
    return axios
        .get<{ mitreAttackVectors: MitreAttackVector[] }>(mitreAttackVectorsURL)
        .then((response) => response?.data?.mitreAttackVectors ?? []);
}
