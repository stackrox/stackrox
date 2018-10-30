// Enforcement type mapped to tile properties for enforcement tab.
const lifecycleToExplanation = {
    DEPLOY:
        'Deployment data was evaluated against this StackRox policy. Based on your configuration, StackRox has prevented the deployment from starting.',
    RUNTIME:
        'Runtime data was evaluated against this StackRox policy. Based on your configuration, StackRox has taken down affected pods.'
};

export default lifecycleToExplanation;
