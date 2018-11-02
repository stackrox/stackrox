// Enforcement type mapped to tile properties for enforcement tab.
const lifecycleToExplanation = {
    DEPLOY:
        'Deployment data was evaluated against this StackRox policy. If enforcement for this policy was configured before the deployment started, the deployment may be blocked, either with the replica count set to 0, or with an unsatisfiable node constraint labeled "BlockedByStackRoxNext". Please check through your orchestrator.',
    RUNTIME:
        'Runtime data was evaluated against this StackRox policy. Based on your configuration, StackRox has taken down affected pods.'
};

export default lifecycleToExplanation;
