import PropTypes from 'prop-types';

// The "title" prop is necessary when used with the "useTabs" hook
// eslint-disable-next-line no-unused-vars
const Tab = ({ title, children }) => children;

Tab.propTypes = {
    title: PropTypes.string.isRequired,
    children: PropTypes.node.isRequired,
};

export default Tab;
