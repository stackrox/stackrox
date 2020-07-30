import PropTypes from 'prop-types';

const processKeyProps = {
    deploymentID: PropTypes.string.isRequired,
    containerName: PropTypes.string.isRequired,
    execFilePath: PropTypes.string.isRequired,
    args: PropTypes.string.isRequired,
};

export default processKeyProps;
