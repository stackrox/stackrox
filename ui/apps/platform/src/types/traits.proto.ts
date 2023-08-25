export type Traits = {
    mutabilityMode: TraitsMutabilityMode;
    origin?: TraitsOrigin;
};

export type TraitsMutabilityMode = 'ALLOW_MUTATE' | 'ALLOW_MUTATE_FORCED';
export type TraitsOrigin = 'IMPERATIVE' | 'DECLARATIVE' | 'DEFAULT' | 'DECLARATIVE_ORPHANED';
