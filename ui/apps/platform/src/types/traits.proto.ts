export type Traits = {
    // EXPERIMENTAL.
    //
    // MutabilityMode specifies whether and how an object can be modified. Default
    // is ALLOW_MUTATE and means there are no modification restrictions; this is equivalent
    // to the absence of MutabilityMode specification. ALLOW_MUTATE_FORCED forbids all
    // modifying operations except object removal with force bit on.
    //
    // Be careful when changing the state of this field. For example, modifying an
    // object from ALLOW_MUTATE to ALLOW_MUTATE_FORCED is allowed but will prohibit any further
    // changes to it, including modifying it back to ALLOW_MUTATE.
    mutabilityMode?: TraitsMutabilityMode;

    // EXPERIMENTAL.
    // visibility allows to specify whether the object should be visible for certain APIs.
    visibility?: TraitsVisibility;
};

export type TraitsMutabilityMode = 'ALLOW_MUTATE' | 'ALLOW_MUTATE_FORCED';

export type TraitsVisibility = 'VISIBLE' | 'HIDDEN';
