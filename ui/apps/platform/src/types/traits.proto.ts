export type Traits = {
    mutabilityMode: TraitsMutabilityMode;
    origin?: TraitsOrigin;
    visibility?: TraitsVisibility;
};

export type TraitsMutabilityMode = 'ALLOW_MUTATE' | 'ALLOW_MUTATE_FORCED';
export type TraitsOrigin = 'IMPERATIVE' | 'DECLARATIVE' | 'DEFAULT' | 'DECLARATIVE_ORPHANED';
export type TraitsVisibility = 'VISIBLE' | 'HIDDEN';
