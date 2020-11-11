export default (flag, desiredValue) => {
    const flagToCheck = Cypress.env(flag) || false;
    return flagToCheck === desiredValue;
};
