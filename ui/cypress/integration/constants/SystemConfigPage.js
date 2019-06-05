export const systemConfigUrl = '/main/systemconfig';

const selectors = {
    navLinks: {
        topNav: '[data-test-id="top-nav-btns"] button',
        menu: '[data-test-id="menu-list"]',
        systemConfig: '[data-test-id="System Config"]',
        logout: '[data-test-id="Logout"]'
    },
    pageHeader: {
        editButton: '[data-test-id="edit-btn"]',
        cancelButton: '[data-test-id="cancel-btn"]',
        saveButton: '[data-test-id="save-btn"]'
    },
    header: {
        widget: '[data-test-id="header-config"]',
        state: '[data-test-id="header-state"]',
        config: {
            toggle: '[data-test-id="header-config"] .form-switch',
            textInput: '[data-test-id="header-config"] textarea',
            colorPickerBtn: '[data-test-id="header-config"] [data-test-id="color-picker"]',
            colorInput: '[data-test-id="header-config"] .chrome-picker input',
            size: {
                input: '[data-test-id="header-config"] .react-select__input input',
                options: '[data-test-id="header-config"] .react-select__option'
            }
        },
        banner: '[data-test-id="header-banner"]'
    },
    footer: {
        widget: '[data-test-id="footer-config"]',
        state: '[data-test-id="footer-state"]',
        config: {
            toggle: '[data-test-id="footer-config"] .form-switch',
            textInput: '[data-test-id="footer-config"] textarea',
            colorPickerBtn: '[data-test-id="footer-config"] [data-test-id="color-picker"]',
            colorInput: '[data-test-id="footer-config"] .chrome-picker input',
            size: {
                input: '[data-test-id="footer-config"] .react-select__input input',
                options: '[data-test-id="footer-config"] .react-select__option'
            }
        },
        banner: '[data-test-id="footer-banner"]'
    },
    loginNotice: {
        widget: '[data-test-id="login-notice-config"]',
        state: '[data-test-id="login-notice-state"]',
        config: {
            toggle: '[data-test-id="login-notice-config"] .form-switch',
            textInput: '[data-test-id="login-notice-config"] textarea'
        },
        banner: '[data-test-id="login-notice"]'
    }
};

export const text = {
    banner: 'Hello this is a sample banner text',
    color: '#000000',
    backgroundColor: '#ffff00'
};

export default selectors;
