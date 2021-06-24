export default (flag, desiredValue) => {
    const flagToCheck = Cypress.env(flag) || false;
    return flagToCheck === desiredValue;
};

/*
 * Return whether or not the testing environment has a feature flag.
 */
export function hasFeatureFlag(flag) {
    return Cypress.env(flag) || false;
}
