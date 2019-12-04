export default (flag, desiredValue) => {
    return Cypress.env(flag) === desiredValue;
};
