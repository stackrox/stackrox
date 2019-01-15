import buildImage from 'images/enforcement-build.svg';
import deployImage from 'images/enforcement-deploy.svg';
import runImage from 'images/enforcement-runtime.svg';

// Enforcement type mapped to tile properties for enforcement tab.
const lifecycleTileMap = {
    BUILD: {
        image: buildImage,
        label: 'BUILD',
        header: 'Enforcement Behavior',
        description:
            'If enabled, StackRox will fail your CI builds when images match the conditions of this policy. Download the CLI above to get started.'
    },
    DEPLOY: {
        image: deployImage,
        label: 'DEPLOY',
        header: 'Enforcement Behavior',
        description:
            'If enabled, StackRox will automatically block creation of deployments that match the conditions of this policy. In clusters with the StackRox Admission Controller enabled, the Kubernetes API server will block noncompliant deployments. In other clusters, StackRox will edit noncompliant deployments to prevent pods from being scheduled.'
    },
    RUNTIME: {
        image: runImage,
        label: 'RUNTIME',
        header: 'Enforcement Behavior',
        description:
            'If enabled, StackRox will automatically kill any pod that matches the conditions of this policy.'
    }
};

export const lifecycleToEnforcementsMap = {
    BUILD: ['FAIL_BUILD_ENFORCEMENT'],
    DEPLOY: ['SCALE_TO_ZERO_ENFORCEMENT', 'UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT'],
    RUNTIME: ['KILL_POD_ENFORCEMENT']
};

export default lifecycleTileMap;
