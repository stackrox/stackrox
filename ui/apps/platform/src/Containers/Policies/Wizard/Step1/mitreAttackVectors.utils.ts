import { MitreAttackVector, MitreTechnique } from 'types/mitre.proto';
import { PolicyMitreAttackVector } from 'types/policy.proto';

// MitreAttackVector

export function getMitreAttackVector(
    mitreAttackVectors: MitreAttackVector[],
    tacticId: string
): MitreAttackVector {
    return (
        mitreAttackVectors.find(
            (mitreAttackVector) => mitreAttackVector.tactic.id === tacticId
        ) ?? {
            tactic: {
                id: tacticId,
                name: '',
                description: '',
            },
            techniques: [],
        }
    );
}

export function getMitreTechnique(
    techniques: MitreTechnique[],
    techniqueId: string
): MitreTechnique {
    return (
        techniques.find((technique) => technique.id === techniqueId) ?? {
            id: techniqueId,
            name: '',
            description: '',
        }
    );
}

/*
 * Impure function sorts MITRE ATT&CK vectors and returns its argument.
 */
export function sortMitreAttackVectors(
    mitreAttackVectors: MitreAttackVector[]
): MitreAttackVector[] {
    // Sort tactics by id property.
    mitreAttackVectors.sort(({ tactic: { id: tacticIdA } }, { tactic: { id: tacticIdB } }) => {
        if (tacticIdA < tacticIdB) {
            return -1;
        }
        if (tacticIdB < tacticIdA) {
            return 1;
        }
        return 0;
    });

    // For each tactic, sort techniques by id property.
    mitreAttackVectors.forEach(({ techniques }) => {
        techniques.sort(({ id: techniqueIdA }, { id: techniqueIdB }) => {
            if (techniqueIdA < techniqueIdB) {
                return -1;
            }
            if (techniqueIdB < techniqueIdA) {
                return 1;
            }
            return 0;
        });
    });

    return mitreAttackVectors;
}

// PolicyMitreAttackVector

export function addPolicyAttackVector(
    mitreAttackVectors: PolicyMitreAttackVector[],
    tacticId: string
): PolicyMitreAttackVector[] {
    // Assume that the interface disables options for already-selected tactics.
    return [...mitreAttackVectors, { tactic: tacticId, techniques: [] }];
}

export function deletePolicyAttackVector(
    mitreAttackVectors: PolicyMitreAttackVector[],
    tacticId: string
): PolicyMitreAttackVector[] {
    return mitreAttackVectors.filter(({ tactic }) => tactic !== tacticId);
}

export function replacePolicyAttackVector(
    mitreAttackVectors: PolicyMitreAttackVector[],
    tacticIdPrev: string,
    tacticIdNext: string
): PolicyMitreAttackVector[] {
    // Assume that the interface disables options for already-selected tactics.
    // Do not copy previous techniques, because they might not apply to next tactic.
    return mitreAttackVectors.map((mitreAttackVector) =>
        mitreAttackVector.tactic === tacticIdPrev
            ? { tactic: tacticIdNext, techniques: [] }
            : mitreAttackVector
    );
}

export function addPolicyTechnique(
    mitreAttackVectors: PolicyMitreAttackVector[],
    tacticId: string,
    techniqueId: string
): PolicyMitreAttackVector[] {
    // Assume that the interface disables options for already-selected techniques of the tactic.
    return mitreAttackVectors.map((mitreAttackVector) =>
        mitreAttackVector.tactic === tacticId
            ? { tactic: tacticId, techniques: [...mitreAttackVector.techniques, techniqueId] }
            : mitreAttackVector
    );
}

export function deletePolicyTechnique(
    mitreAttackVectors: PolicyMitreAttackVector[],
    tacticId: string,
    techniqueId: string
): PolicyMitreAttackVector[] {
    return mitreAttackVectors.map((mitreAttackVector) =>
        mitreAttackVector.tactic === tacticId
            ? {
                  tactic: tacticId,
                  techniques: mitreAttackVector.techniques.filter((id) => id !== techniqueId),
              }
            : mitreAttackVector
    );
}

export function replacePolicyTechnique(
    mitreAttackVectors: PolicyMitreAttackVector[],
    tacticId: string,
    techniqueIdPrev: string,
    techniqueIdNext: string
): PolicyMitreAttackVector[] {
    // Assume that the interface disables options for already-selected techniques of the tactic.
    return mitreAttackVectors.map((mitreAttackVector) =>
        mitreAttackVector.tactic === tacticId
            ? {
                  tactic: tacticId,
                  techniques: mitreAttackVector.techniques.map((id) =>
                      id === techniqueIdPrev ? techniqueIdNext : id
                  ),
              }
            : mitreAttackVector
    );
}
