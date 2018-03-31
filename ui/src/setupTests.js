import { configure } from 'enzyme';
import Adapter from 'enzyme-adapter-react-16';

/**
 * Fix for test error "matchMedia not present, legacy browsers require a polyfill"
 * https://github.com/akiran/react-slick/issues/742
 */
if (window.matchMedia) {
    window.matchMedia = window.matchMedia;
} else {
    window.matchMedia = () => ({
        matches: false,
        addListener() {},
        removeListener() {}
    });
}

configure({ adapter: new Adapter() });
