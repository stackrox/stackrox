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
            'If enabled, StackRox will fail your CI builds when build data matches parameters from this policy. Download the CLI above to get started.'
    },
    DEPLOY: {
        image: deployImage,
        label: 'DEPLOY',
        header: 'Enforcement Behavior',
        description:
            'If enabled, StackRox will automatically apply one of two types of enforcement actions when deploy data matches parameters from this policy.'
    },
    RUNTIME: {
        image: runImage,
        label: 'RUNTIME',
        header: 'Enforcement Behavior',
        description:
            'If enabled, StackRox will automatically kill pods associated with any runtime data that matches parameters from this policy.'
    }
};

export const lifecycleToEnforcementsMap = {
    BUILD: ['FAIL_BUILD_ENFORCEMENT'],
    DEPLOY: ['SCALE_TO_ZERO_ENFORCEMENT', 'UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT'],
    RUNTIME: ['KILL_POD_ENFORCEMENT']
};

export default lifecycleTileMap;
