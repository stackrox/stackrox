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
